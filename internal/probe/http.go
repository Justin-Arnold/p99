package probe

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	p99errors "github.com/Justin-Arnold/p99/internal/errors"
	"github.com/Justin-Arnold/p99/internal/latency"
	"github.com/Justin-Arnold/p99/internal/requestspec"
)

type HTTPRunner struct {
	Config      HTTPConfig
	Body        []byte
	RequestPlan *requestspec.Plan
	Client      *http.Client
}

type observation struct {
	latency time.Duration
	at      time.Time
	status  int
	class   string
	request requestspec.RenderedRequest
}

type requestJob struct {
	record  bool
	request requestspec.RenderedRequest
}

func (r HTTPRunner) Run(ctx context.Context) (RunResult, error) {
	if err := r.validate(); err != nil {
		return RunResult{}, err
	}
	cfg := r.Config
	if r.RequestPlan != nil && cfg.RequestSeed == 0 {
		cfg.RequestSeed = time.Now().UnixNano()
	}

	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	if client.Timeout == 0 {
		client.Timeout = cfg.Timeout
	}

	total := cfg.Warmup + cfg.Duration
	runCtx, cancel := context.WithTimeout(ctx, total+cfg.Timeout+time.Second)
	defer cancel()

	// The scheduler owns pacing and workers only own request execution. Keeping
	// those roles separate makes the concurrency limit independent from RPS.
	jobs := make(chan requestJob)
	observations := make(chan observation, cfg.Concurrency*2)
	var wg sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				obs := r.doRequest(runCtx, client, job.request)
				if job.record {
					observations <- obs
				}
			}
		}()
	}

	startedAt := time.Now()
	renderErr := atomic.Value{}
	go func() {
		r.schedule(runCtx, jobs, false, cfg.Warmup, cfg, &renderErr)
		// Warmup requests exercise caches and connections without polluting the
		// measured distribution.
		startedAt = time.Now()
		r.schedule(runCtx, jobs, true, cfg.Duration, cfg, &renderErr)
		close(jobs)
		wg.Wait()
		close(observations)
	}()

	h := latency.NewHistogram()
	sampler := latency.NewSlowSampler(cfg.SlowSamples)
	errorsByClass := map[string]int{}
	mix := requestMix(r.RequestPlan)
	points := []latency.TimedLatency{}
	var success, failures int

	for obs := range observations {
		h.Record(obs.latency)
		points = append(points, latency.TimedLatency{At: obs.at, Latency: obs.latency})
		sample := latency.SlowSample{
			Latency:   obs.latency,
			Timestamp: obs.at,
			Status:    obs.status,
			Error:     obs.class,
			Request:   obs.request.Name,
			Method:    obs.request.Method,
			URL:       sampleURL(obs.request.URL),
		}
		sampler.Add(sample)
		if obs.request.Name != "" {
			stats := mix[obs.request.Name]
			stats.Count++
			if obs.class == "" {
				stats.Success++
			} else {
				stats.Errors++
				stats.ErrorBreakdown[obs.class]++
			}
		}
		if obs.class == "" {
			success++
		} else {
			failures++
			errorsByClass[obs.class]++
		}
	}
	if v := renderErr.Load(); v != nil {
		return RunResult{}, v.(error)
	}

	endedAt := time.Now()
	return RunResult{
		Version:     ResultVersion,
		Config:      cfg,
		StartedAt:   startedAt,
		EndedAt:     endedAt,
		Summary:     latency.Summarize(h, success, failures),
		Histogram:   h.Buckets(),
		SlowSamples: sampler.Samples(),
		Errors:      errorsByClass,
		RequestMix:  sortedRequestMix(mix, r.RequestPlan),
		Shape:       latency.AnalyzeShape(points),
	}, nil
}

func (r HTTPRunner) validate() error {
	if r.RequestPlan == nil && r.Config.URL == "" {
		return fmt.Errorf("url is required")
	}
	if r.RequestPlan == nil {
		if _, err := url.ParseRequestURI(r.Config.URL); err != nil {
			return fmt.Errorf("invalid url: %w", err)
		}
	}
	if r.RequestPlan == nil && r.Config.Method == "" {
		return fmt.Errorf("method is required")
	}
	if r.Config.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if r.Config.RPS <= 0 {
		return fmt.Errorf("rps must be positive")
	}
	if r.Config.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive")
	}
	if r.Config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if r.Config.SlowSamples < 0 {
		return fmt.Errorf("slow-samples must be non-negative")
	}
	if r.RequestPlan != nil && len(r.RequestPlan.Requests) == 0 {
		return fmt.Errorf("request plan is empty")
	}
	return nil
}

func (r HTTPRunner) schedule(ctx context.Context, jobs chan<- requestJob, record bool, duration time.Duration, cfg HTTPConfig, renderErr *atomic.Value) {
	if duration <= 0 {
		return
	}
	deadline := time.NewTimer(duration)
	defer deadline.Stop()
	rng := rand.New(rand.NewSource(cfg.RequestSeed))
	if !record {
		rng = rand.New(rand.NewSource(cfg.RequestSeed - 1))
	}

	if cfg.RPS <= 0 {
		for {
			select {
			case <-ctx.Done():
				return
			case <-deadline.C:
				return
			default:
				job, err := r.nextJob(record, cfg, rng)
				if err != nil {
					renderErr.Store(err)
					return
				}
				select {
				case jobs <- job:
				case <-ctx.Done():
					return
				case <-deadline.C:
					return
				}
			}
		}
	}

	first, err := r.nextJob(record, cfg, rng)
	if err != nil {
		renderErr.Store(err)
		return
	}
	select {
	case jobs <- first:
	case <-ctx.Done():
		return
	case <-deadline.C:
		return
	}

	// Send immediately, then tick. Without the first request, very short runs can
	// report no data even when the target RPS is sensible.
	interval := time.Duration(float64(time.Second) / cfg.RPS)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			return
		case <-ticker.C:
			job, err := r.nextJob(record, cfg, rng)
			if err != nil {
				renderErr.Store(err)
				return
			}
			select {
			case jobs <- job:
			case <-ctx.Done():
				return
			case <-deadline.C:
				return
			}
		}
	}
}

func (r HTTPRunner) nextJob(record bool, cfg HTTPConfig, rng *rand.Rand) (requestJob, error) {
	req := requestspec.RenderedRequest{
		Method:         cfg.Method,
		URL:            cfg.URL,
		Headers:        cfg.Headers,
		Body:           append([]byte(nil), r.Body...),
		ExpectedStatus: cfg.ExpectedStatus,
	}
	if r.RequestPlan != nil {
		rendered, err := r.RequestPlan.Render(rng)
		if err != nil {
			return requestJob{}, err
		}
		req = rendered
	}
	return requestJob{record: record, request: req}, nil
}

func (r HTTPRunner) doRequest(ctx context.Context, client *http.Client, request requestspec.RenderedRequest) observation {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, request.Method, request.URL, bytes.NewReader(request.Body))
	if err != nil {
		return observation{latency: time.Since(start), at: start, class: p99errors.Unknown, request: request}
	}
	for k, v := range request.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return observation{latency: time.Since(start), at: start, class: p99errors.Classify(err), request: request}
	}
	defer resp.Body.Close()

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return observation{latency: time.Since(start), at: start, status: resp.StatusCode, class: p99errors.BodyRead, request: request}
	}
	obs := observation{latency: time.Since(start), at: start, status: resp.StatusCode, request: request}
	if !statusOK(resp.StatusCode, request.ExpectedStatus) {
		obs.class = p99errors.ClassifyStatus(resp.StatusCode)
	}
	return obs
}

func statusOK(status int, expected []int) bool {
	if len(expected) == 0 {
		return status >= 200 && status < 400
	}
	for _, want := range expected {
		if status == want {
			return true
		}
	}
	return false
}

func requestMix(plan *requestspec.Plan) map[string]*RequestStats {
	if plan == nil {
		return nil
	}
	mix := map[string]*RequestStats{}
	for _, req := range plan.Requests {
		mix[req.Name] = &RequestStats{Name: req.Name, Weight: req.Weight, ErrorBreakdown: map[string]int{}}
	}
	return mix
}

func sortedRequestMix(mix map[string]*RequestStats, plan *requestspec.Plan) []RequestStats {
	if plan == nil {
		return nil
	}
	out := make([]RequestStats, 0, len(plan.Requests))
	for _, req := range plan.Requests {
		stats := mix[req.Name]
		if len(stats.ErrorBreakdown) == 0 {
			stats.ErrorBreakdown = nil
		}
		out = append(out, *stats)
	}
	return out
}

func sampleURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	if strings.TrimSpace(path) == "" {
		return raw
	}
	// Slow samples avoid scheme/host so saved reports are easier to share without
	// leaking more environment detail than the request path already contains.
	return path
}

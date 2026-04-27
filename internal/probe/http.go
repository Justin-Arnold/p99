package probe

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	p99errors "github.com/justin/p99/internal/errors"
	"github.com/justin/p99/internal/latency"
)

type HTTPRunner struct {
	Config HTTPConfig
	Body   []byte
	Client *http.Client
}

type observation struct {
	latency time.Duration
	at      time.Time
	status  int
	class   string
}

func (r HTTPRunner) Run(ctx context.Context) (RunResult, error) {
	if err := r.validate(); err != nil {
		return RunResult{}, err
	}

	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: r.Config.Timeout}
	}
	if client.Timeout == 0 {
		client.Timeout = r.Config.Timeout
	}

	total := r.Config.Warmup + r.Config.Duration
	runCtx, cancel := context.WithTimeout(ctx, total+r.Config.Timeout+time.Second)
	defer cancel()

	jobs := make(chan bool)
	observations := make(chan observation, r.Config.Concurrency*2)
	var wg sync.WaitGroup
	for i := 0; i < r.Config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for record := range jobs {
				obs := r.doRequest(runCtx, client)
				if record {
					observations <- obs
				}
			}
		}()
	}

	startedAt := time.Now()
	go func() {
		r.schedule(runCtx, jobs, false, r.Config.Warmup)
		startedAt = time.Now()
		r.schedule(runCtx, jobs, true, r.Config.Duration)
		close(jobs)
		wg.Wait()
		close(observations)
	}()

	h := latency.NewHistogram()
	sampler := latency.NewSlowSampler(r.Config.SlowSamples)
	errorsByClass := map[string]int{}
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
			Method:    r.Config.Method,
			URL:       sampleURL(r.Config.URL),
		}
		sampler.Add(sample)
		if obs.class == "" {
			success++
		} else {
			failures++
			errorsByClass[obs.class]++
		}
	}

	endedAt := time.Now()
	return RunResult{
		Version:     ResultVersion,
		Config:      r.Config,
		StartedAt:   startedAt,
		EndedAt:     endedAt,
		Summary:     latency.Summarize(h, success, failures),
		Histogram:   h.Buckets(),
		SlowSamples: sampler.Samples(),
		Errors:      errorsByClass,
		Shape:       latency.AnalyzeShape(points),
	}, nil
}

func (r HTTPRunner) validate() error {
	if r.Config.URL == "" {
		return fmt.Errorf("url is required")
	}
	if _, err := url.ParseRequestURI(r.Config.URL); err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if r.Config.Method == "" {
		return fmt.Errorf("method is required")
	}
	if r.Config.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if r.Config.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive")
	}
	if r.Config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

func (r HTTPRunner) schedule(ctx context.Context, jobs chan<- bool, record bool, duration time.Duration) {
	if duration <= 0 {
		return
	}
	deadline := time.NewTimer(duration)
	defer deadline.Stop()

	if r.Config.RPS <= 0 {
		for {
			select {
			case <-ctx.Done():
				return
			case <-deadline.C:
				return
			case jobs <- record:
			}
		}
	}

	select {
	case jobs <- record:
	case <-ctx.Done():
		return
	case <-deadline.C:
		return
	}

	interval := time.Duration(float64(time.Second) / r.Config.RPS)
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
			select {
			case jobs <- record:
			case <-ctx.Done():
				return
			case <-deadline.C:
				return
			}
		}
	}
}

func (r HTTPRunner) doRequest(ctx context.Context, client *http.Client) observation {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, r.Config.Method, r.Config.URL, bytes.NewReader(r.Body))
	if err != nil {
		return observation{latency: time.Since(start), at: start, class: p99errors.Unknown}
	}
	for k, v := range r.Config.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return observation{latency: time.Since(start), at: start, class: p99errors.Classify(err)}
	}
	defer resp.Body.Close()

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return observation{latency: time.Since(start), at: start, status: resp.StatusCode, class: p99errors.BodyRead}
	}
	obs := observation{latency: time.Since(start), at: start, status: resp.StatusCode}
	if !r.statusOK(resp.StatusCode) {
		obs.class = p99errors.ClassifyStatus(resp.StatusCode)
	}
	return obs
}

func (r HTTPRunner) statusOK(status int) bool {
	if len(r.Config.ExpectedStatus) == 0 {
		return status >= 200 && status < 400
	}
	for _, want := range r.Config.ExpectedStatus {
		if status == want {
			return true
		}
	}
	return false
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
	return path
}

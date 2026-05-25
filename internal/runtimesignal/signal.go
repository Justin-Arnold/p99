package runtimesignal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BaseURL string        `json:"base_url"`
	Timeout time.Duration `json:"timeout_ns"`
}

type Snapshot struct {
	At              time.Time                  `json:"at"`
	BaseURL         string                     `json:"base_url"`
	Goroutines      *uint64                    `json:"goroutines,omitempty"`
	HeapAllocBytes  *uint64                    `json:"heap_alloc_bytes,omitempty"`
	HeapInUseBytes  *uint64                    `json:"heap_in_use_bytes,omitempty"`
	HeapSysBytes    *uint64                    `json:"heap_sys_bytes,omitempty"`
	NextGCBytes     *uint64                    `json:"next_gc_bytes,omitempty"`
	GCCycles        *uint64                    `json:"gc_cycles,omitempty"`
	GCPauseTotalNS  *uint64                    `json:"gc_pause_total_ns,omitempty"`
	LastGCPauseNS   *uint64                    `json:"last_gc_pause_ns,omitempty"`
	MutexSamples    *uint64                    `json:"mutex_samples,omitempty"`
	BlockSamples    *uint64                    `json:"block_samples,omitempty"`
	SchedulerPauses *RuntimeMetricDistribution `json:"scheduler_pauses,omitempty"`
	DBPoolWait      *DBPoolWait                `json:"db_pool_wait,omitempty"`
	Warnings        []string                   `json:"warnings,omitempty"`
}

type RuntimeMetricDistribution struct {
	Count uint64 `json:"count"`
	SumNS uint64 `json:"sum_ns"`
}

type DBPoolWait struct {
	Count      uint64 `json:"count"`
	DurationNS uint64 `json:"duration_ns"`
}

type Correlation struct {
	Before Snapshot `json:"before"`
	After  Snapshot `json:"after"`
	Delta  Delta    `json:"delta"`
	Notes  []string `json:"notes,omitempty"`
}

type Delta struct {
	Goroutines      *int64                     `json:"goroutines,omitempty"`
	HeapAllocBytes  *int64                     `json:"heap_alloc_bytes,omitempty"`
	HeapInUseBytes  *int64                     `json:"heap_in_use_bytes,omitempty"`
	HeapSysBytes    *int64                     `json:"heap_sys_bytes,omitempty"`
	NextGCBytes     *int64                     `json:"next_gc_bytes,omitempty"`
	GCCycles        *uint64                    `json:"gc_cycles,omitempty"`
	GCPauseTotalNS  *uint64                    `json:"gc_pause_total_ns,omitempty"`
	LastGCPauseNS   *int64                     `json:"last_gc_pause_ns,omitempty"`
	MutexSamples    *int64                     `json:"mutex_samples,omitempty"`
	BlockSamples    *int64                     `json:"block_samples,omitempty"`
	SchedulerPauses *RuntimeMetricDistribution `json:"scheduler_pauses,omitempty"`
	DBPoolWait      *DBPoolWait                `json:"db_pool_wait,omitempty"`
}

type Collector struct {
	Config Config
	Client *http.Client
}

func (c Collector) Collect(ctx context.Context) (Snapshot, error) {
	if c.Config.BaseURL == "" {
		return Snapshot{}, fmt.Errorf("runtime base url is required")
	}
	base, err := url.Parse(c.Config.BaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return Snapshot{}, fmt.Errorf("invalid runtime base url: %q", c.Config.BaseURL)
	}
	timeout := c.Config.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	if client.Timeout == 0 {
		client.Timeout = timeout
	}

	s := Snapshot{At: time.Now(), BaseURL: normalizeBase(base)}
	// Missing debug endpoints should reduce confidence, not fail the whole
	// latency run. Warnings preserve that uncertainty in the saved report.
	c.collectExpvar(ctx, client, &s)
	c.collectGoroutines(ctx, client, &s)
	c.collectProfileSamples(ctx, client, &s, "mutex")
	c.collectProfileSamples(ctx, client, &s, "block")
	return s, nil
}

func Correlate(before, after Snapshot) Correlation {
	c := Correlation{Before: before, After: after, Delta: delta(before, after)}
	c.Notes = notes(c)
	return c
}

func (c Collector) collectExpvar(ctx context.Context, client *http.Client, s *Snapshot) {
	body, err := get(ctx, client, s.BaseURL+"/debug/vars")
	if err != nil {
		s.Warnings = append(s.Warnings, "expvar unavailable: "+err.Error())
		return
	}
	var vars map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&vars); err != nil {
		s.Warnings = append(s.Warnings, "expvar parse failed: "+err.Error())
		return
	}
	if memstats, ok := object(vars["memstats"]); ok {
		s.HeapAllocBytes = uintValue(memstats, "Alloc")
		s.HeapInUseBytes = uintValue(memstats, "HeapInuse")
		s.HeapSysBytes = uintValue(memstats, "HeapSys")
		s.NextGCBytes = uintValue(memstats, "NextGC")
		s.GCCycles = uintValue(memstats, "NumGC")
		s.GCPauseTotalNS = uintValue(memstats, "PauseTotalNs")
		if pauses, ok := array(memstats["PauseNs"]); ok && len(pauses) > 0 {
			if numGC := derefUint(s.GCCycles); numGC > 0 {
				// PauseNs is a ring buffer indexed by GC cycle. NumGC identifies
				// the most recent pause without assuming the buffer starts at zero.
				idx := int((numGC - 1) % uint64(len(pauses)))
				s.LastGCPauseNS = uintFromAny(pauses[idx])
			}
		}
	} else {
		s.Warnings = append(s.Warnings, "expvar memstats missing")
	}
	s.SchedulerPauses = schedulerPauses(vars)
	s.DBPoolWait = dbPoolWait(vars)
}

func (c Collector) collectGoroutines(ctx context.Context, client *http.Client, s *Snapshot) {
	body, err := get(ctx, client, s.BaseURL+"/debug/pprof/goroutine?debug=1")
	if err != nil {
		s.Warnings = append(s.Warnings, "goroutine profile unavailable: "+err.Error())
		return
	}
	re := regexp.MustCompile(`goroutine profile:\s+total\s+(\d+)`)
	match := re.FindSubmatch(body)
	if len(match) != 2 {
		s.Warnings = append(s.Warnings, "goroutine profile did not include a total")
		return
	}
	v, err := strconv.ParseUint(string(match[1]), 10, 64)
	if err != nil {
		s.Warnings = append(s.Warnings, "goroutine total parse failed: "+err.Error())
		return
	}
	s.Goroutines = &v
}

func (c Collector) collectProfileSamples(ctx context.Context, client *http.Client, s *Snapshot, name string) {
	body, err := get(ctx, client, s.BaseURL+"/debug/pprof/"+name+"?debug=1")
	if err != nil {
		s.Warnings = append(s.Warnings, name+" profile unavailable: "+err.Error())
		return
	}
	count := countProfileSamples(body)
	switch name {
	case "mutex":
		s.MutexSamples = &count
	case "block":
		s.BlockSamples = &count
	}
}

func get(ctx context.Context, client *http.Client, raw string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func countProfileSamples(body []byte) uint64 {
	var count uint64
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.Contains(line, "profile:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		// Text pprof output is stable enough to count sample rows, but not stable
		// enough to infer wait duration portably across Go versions.
		if _, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
			count++
		}
	}
	return count
}

func normalizeBase(u *url.URL) string {
	out := *u
	out.Path = strings.TrimSuffix(out.Path, "/")
	out.RawQuery = ""
	out.Fragment = ""
	return out.String()
}

func delta(before, after Snapshot) Delta {
	return Delta{
		Goroutines:      signedUintDelta(before.Goroutines, after.Goroutines),
		HeapAllocBytes:  signedUintDelta(before.HeapAllocBytes, after.HeapAllocBytes),
		HeapInUseBytes:  signedUintDelta(before.HeapInUseBytes, after.HeapInUseBytes),
		HeapSysBytes:    signedUintDelta(before.HeapSysBytes, after.HeapSysBytes),
		NextGCBytes:     signedUintDelta(before.NextGCBytes, after.NextGCBytes),
		GCCycles:        monotonicDelta(before.GCCycles, after.GCCycles),
		GCPauseTotalNS:  monotonicDelta(before.GCPauseTotalNS, after.GCPauseTotalNS),
		LastGCPauseNS:   signedUintDelta(before.LastGCPauseNS, after.LastGCPauseNS),
		MutexSamples:    signedUintDelta(before.MutexSamples, after.MutexSamples),
		BlockSamples:    signedUintDelta(before.BlockSamples, after.BlockSamples),
		SchedulerPauses: distributionDelta(before.SchedulerPauses, after.SchedulerPauses),
		DBPoolWait:      dbPoolWaitDelta(before.DBPoolWait, after.DBPoolWait),
	}
}

func notes(c Correlation) []string {
	var out []string
	if d := c.Delta.GCPauseTotalNS; d != nil && *d > 0 {
		out = append(out, "GC pause time increased during the measured window. Compare this with tail latency before assuming causality.")
	}
	if d := c.Delta.Goroutines; d != nil && *d > 100 {
		out = append(out, "Goroutine count rose materially during the window. This can happen with backlog, leaks, or work queued behind slow dependencies.")
	}
	if d := c.Delta.HeapAllocBytes; d != nil && *d > 64*1024*1024 {
		out = append(out, "Heap allocation rose materially during the window. Check whether allocation pressure lines up with p99 movement.")
	}
	if d := c.Delta.MutexSamples; d != nil && *d > 0 {
		out = append(out, "Mutex profile samples were present. Inspect the mutex profile if p99 also moved.")
	}
	if d := c.Delta.BlockSamples; d != nil && *d > 0 {
		out = append(out, "Block profile samples were present. This can point at channel, select, or synchronization waits.")
	}
	if d := c.Delta.SchedulerPauses; d != nil && d.Count > 0 {
		out = append(out, "Scheduler pause metrics changed during the window. Treat this as a runtime signal to inspect alongside CPU and GC profiles.")
	}
	if d := c.Delta.DBPoolWait; d != nil && (d.Count > 0 || d.DurationNS > 0) {
		out = append(out, "DB pool wait counters changed during the window. This can line up with connection pool saturation or slow queries.")
	}
	if len(out) == 0 {
		out = append(out, "No runtime signal moved enough for a built-in hint. Absence of a hint is not proof that the runtime was uninvolved.")
	}
	return out
}

func object(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func array(v any) ([]any, bool) {
	a, ok := v.([]any)
	return a, ok
}

func uintValue(m map[string]any, key string) *uint64 {
	return uintFromAny(m[key])
}

func uintFromAny(v any) *uint64 {
	switch x := v.(type) {
	case float64:
		if x < 0 {
			return nil
		}
		u := uint64(x)
		return &u
	case json.Number:
		u, err := strconv.ParseUint(string(x), 10, 64)
		if err != nil {
			return nil
		}
		return &u
	case string:
		u, err := strconv.ParseUint(x, 10, 64)
		if err != nil {
			return nil
		}
		return &u
	default:
		return nil
	}
}

func derefUint(v *uint64) uint64 {
	if v == nil {
		return 0
	}
	return *v
}

func signedUintDelta(before, after *uint64) *int64 {
	if before == nil || after == nil {
		return nil
	}
	d := int64(*after) - int64(*before)
	return &d
}

func monotonicDelta(before, after *uint64) *uint64 {
	if before == nil || after == nil || *after < *before {
		// Counter resets happen across restarts or endpoint changes. Returning nil
		// is less misleading than reporting a huge wrapped delta.
		return nil
	}
	d := *after - *before
	return &d
}

func distributionDelta(before, after *RuntimeMetricDistribution) *RuntimeMetricDistribution {
	if before == nil || after == nil || after.Count < before.Count || after.SumNS < before.SumNS {
		return nil
	}
	return &RuntimeMetricDistribution{Count: after.Count - before.Count, SumNS: after.SumNS - before.SumNS}
}

func dbPoolWaitDelta(before, after *DBPoolWait) *DBPoolWait {
	if before == nil || after == nil || after.Count < before.Count || after.DurationNS < before.DurationNS {
		return nil
	}
	return &DBPoolWait{Count: after.Count - before.Count, DurationNS: after.DurationNS - before.DurationNS}
}

func schedulerPauses(vars map[string]any) *RuntimeMetricDistribution {
	for _, key := range []string{
		"/sched/pauses/total/gc:seconds",
		"/sched/pauses/total/other:seconds",
		"scheduler_pauses",
	} {
		if dist := distributionFromAny(vars[key]); dist != nil {
			return dist
		}
	}
	return nil
}

func distributionFromAny(v any) *RuntimeMetricDistribution {
	m, ok := object(v)
	if !ok {
		return nil
	}
	count := firstUint(m, "count", "Count")
	sumNS := firstUint(m, "sum_ns", "SumNS", "sumNanoseconds", "SumNanoseconds")
	if sumNS == nil {
		if sumSeconds := firstFloat(m, "sum_seconds", "SumSeconds"); sumSeconds != nil {
			v := uint64(*sumSeconds * float64(time.Second))
			sumNS = &v
		}
	}
	if count == nil || sumNS == nil {
		return nil
	}
	return &RuntimeMetricDistribution{Count: *count, SumNS: *sumNS}
}

func dbPoolWait(vars map[string]any) *DBPoolWait {
	count := firstUint(vars, "db_pool_wait_count", "DBPoolWaitCount")
	duration := firstUint(vars, "db_pool_wait_duration_ns", "DBPoolWaitDurationNS")
	if count != nil && duration != nil {
		return &DBPoolWait{Count: *count, DurationNS: *duration}
	}
	for _, key := range []string{"db_pool", "db", "sql"} {
		if m, ok := object(vars[key]); ok {
			count = firstUint(m, "wait_count", "WaitCount", "db_pool_wait_count")
			duration = firstUint(m, "wait_duration_ns", "WaitDurationNS", "WaitDuration", "db_pool_wait_duration_ns")
			if count != nil && duration != nil {
				return &DBPoolWait{Count: *count, DurationNS: *duration}
			}
		}
	}
	return nil
}

func firstUint(m map[string]any, keys ...string) *uint64 {
	for _, key := range keys {
		if v := uintFromAny(m[key]); v != nil {
			return v
		}
	}
	return nil
}

func firstFloat(m map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		switch v := m[key].(type) {
		case float64:
			return &v
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err == nil {
				return &f
			}
		case json.Number:
			f, err := v.Float64()
			if err == nil {
				return &f
			}
		}
	}
	return nil
}

package runtimesignal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCollectorCollectsRuntimeSignals(t *testing.T) {
	srv := runtimeServer(10)
	defer srv.Close()

	got, err := Collector{Config: Config{BaseURL: srv.URL, Timeout: time.Second}}.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Goroutines == nil || *got.Goroutines != 17 {
		t.Fatalf("goroutines got %#v", got.Goroutines)
	}
	if got.HeapAllocBytes == nil || *got.HeapAllocBytes != 10*1024 {
		t.Fatalf("heap alloc got %#v", got.HeapAllocBytes)
	}
	if got.GCPauseTotalNS == nil || *got.GCPauseTotalNS != 1000 {
		t.Fatalf("pause total got %#v", got.GCPauseTotalNS)
	}
	if got.MutexSamples == nil || *got.MutexSamples != 1 {
		t.Fatalf("mutex samples got %#v", got.MutexSamples)
	}
	if got.BlockSamples == nil || *got.BlockSamples != 1 {
		t.Fatalf("block samples got %#v", got.BlockSamples)
	}
	if got.SchedulerPauses == nil || got.SchedulerPauses.Count != 10 {
		t.Fatalf("scheduler got %#v", got.SchedulerPauses)
	}
	if got.DBPoolWait == nil || got.DBPoolWait.Count != 10 {
		t.Fatalf("db wait got %#v", got.DBPoolWait)
	}
}

func TestCorrelateComputesDeltasAndNotes(t *testing.T) {
	before := Snapshot{
		Goroutines:     u64(10),
		HeapAllocBytes: u64(1024),
		GCCycles:       u64(1),
		GCPauseTotalNS: u64(100),
		DBPoolWait:     &DBPoolWait{Count: 1, DurationNS: 1000},
	}
	after := Snapshot{
		Goroutines:     u64(200),
		HeapAllocBytes: u64(80 * 1024 * 1024),
		GCCycles:       u64(3),
		GCPauseTotalNS: u64(500),
		DBPoolWait:     &DBPoolWait{Count: 4, DurationNS: 9000},
	}
	got := Correlate(before, after)
	if got.Delta.Goroutines == nil || *got.Delta.Goroutines != 190 {
		t.Fatalf("goroutine delta got %#v", got.Delta.Goroutines)
	}
	if got.Delta.GCCycles == nil || *got.Delta.GCCycles != 2 {
		t.Fatalf("GC delta got %#v", got.Delta.GCCycles)
	}
	if got.Delta.DBPoolWait == nil || got.Delta.DBPoolWait.Count != 3 {
		t.Fatalf("DB delta got %#v", got.Delta.DBPoolWait)
	}
	if len(got.Notes) == 0 || !strings.Contains(strings.Join(got.Notes, " "), "GC") {
		t.Fatalf("notes got %#v", got.Notes)
	}
}

func runtimeServer(seed uint64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/debug/vars":
			fmt.Fprintf(w, `{
				"memstats": {
					"Alloc": %d,
					"HeapInuse": %d,
					"HeapSys": %d,
					"NextGC": %d,
					"NumGC": %d,
					"PauseTotalNs": %d,
					"PauseNs": [%d]
				},
				"/sched/pauses/total/gc:seconds": {"count": %d, "sum_seconds": 0.002},
				"db_pool": {"WaitCount": %d, "WaitDuration": %d}
			}`, seed*1024, seed*2048, seed*4096, seed*8192, seed, seed*100, seed*7, seed, seed, seed*1000)
		case "/debug/pprof/goroutine":
			fmt.Fprintln(w, "goroutine profile: total 17")
		case "/debug/pprof/mutex", "/debug/pprof/block":
			fmt.Fprintln(w, "--- contention:")
			fmt.Fprintln(w, "1 100 @ 0x1 0x2")
		default:
			http.NotFound(w, r)
		}
	}))
}

func u64(v uint64) *uint64 {
	return &v
}

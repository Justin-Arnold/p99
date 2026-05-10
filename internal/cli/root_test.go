package cli

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justin/p99/internal/latency"
	"github.com/justin/p99/internal/output"
	"github.com/justin/p99/internal/probe"
)

func TestHTTPCommandWritesJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "run.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"http", srv.URL, "--duration", "100ms", "--rps", "20", "--concurrency", "1", "--output", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	result, err := output.ReadJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Count == 0 {
		t.Fatal("expected requests in JSON result")
	}
	if !strings.Contains(stdout.String(), "Requests:") {
		t.Fatalf("missing summary output: %s", stdout.String())
	}
}

func TestHTTPCommandWritesAdditionalExports(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	dir := t.TempDir()
	md := filepath.Join(dir, "run.md")
	prom := filepath.Join(dir, "run.prom")
	otel := filepath.Join(dir, "run.otlp.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"http",
		"--duration", "80ms",
		"--rps", "20",
		"--markdown-output", md,
		"--prometheus-output", prom,
		"--otel-output", otel,
		srv.URL,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	for _, tc := range []struct {
		path string
		want string
	}{
		{md, "# p99 report"},
		{prom, "p99_http_requests_total"},
		{otel, `"resourceMetrics"`},
	} {
		data, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), tc.want) {
			t.Fatalf("%s missing %q: %s", tc.path, tc.want, data)
		}
	}
}

func TestHTTPCommandThresholdExit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"http", "--duration", "80ms", "--rps", "20", "--concurrency", "1", "--p99-under", "1ns", srv.URL}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "threshold failed") {
		t.Fatalf("missing threshold failure: %s", stderr.String())
	}
}

func TestReportCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.json")
	if err := output.WriteJSONFile(path, probe.RunResult{
		Version: probe.ResultVersion,
		Summary: latency.Summary{
			Count: 3,
			P99:   10 * time.Millisecond,
		},
		Shape: latency.Shape{Kind: "stable"},
	}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"report", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Requests: 3") {
		t.Fatalf("unexpected report: %s", stdout.String())
	}
}

func TestReportCommandExportsFormats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.json")
	if err := output.WriteJSONFile(path, probe.RunResult{
		Version: probe.ResultVersion,
		EndedAt: time.Unix(10, 0),
		Summary: latency.Summary{
			Count: 1,
			P99:   10 * time.Millisecond,
		},
		Shape: latency.Shape{Kind: "stable"},
	}); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		format string
		want   string
	}{
		{"markdown", "# p99 report"},
		{"prometheus", "p99_http_requests_total"},
		{"otel", `"resourceMetrics"`},
	} {
		var stdout, stderr bytes.Buffer
		code := Run([]string{"report", "--format", tc.format, path}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("%s exit %d, stderr: %s", tc.format, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), tc.want) {
			t.Fatalf("%s output missing %q: %s", tc.format, tc.want, stdout.String())
		}
	}
}

func TestCompareCommandThresholdExit(t *testing.T) {
	dir := t.TempDir()
	before := filepath.Join(dir, "before.json")
	after := filepath.Join(dir, "after.json")
	if err := output.WriteJSONFile(before, probe.RunResult{Summary: latency.Summary{Count: 10, P99: 100 * time.Millisecond}}); err != nil {
		t.Fatal(err)
	}
	if err := output.WriteJSONFile(after, probe.RunResult{Summary: latency.Summary{Count: 10, P99: 130 * time.Millisecond}}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"compare", before, after, "--max-p99-regression", "20%"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit %d, stdout: %s stderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "p99 regression") {
		t.Fatalf("missing regression failure: %s", stderr.String())
	}
}

func TestCompareCommandAdditionalThresholds(t *testing.T) {
	dir := t.TempDir()
	before := filepath.Join(dir, "before.json")
	after := filepath.Join(dir, "after.json")
	if err := output.WriteJSONFile(before, probe.RunResult{Summary: latency.Summary{
		Count:     100,
		P95:       100 * time.Millisecond,
		P999:      150 * time.Millisecond,
		Max:       200 * time.Millisecond,
		ErrorRate: 0.01,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := output.WriteJSONFile(after, probe.RunResult{Summary: latency.Summary{
		Count:     80,
		P95:       140 * time.Millisecond,
		P999:      220 * time.Millisecond,
		Max:       300 * time.Millisecond,
		ErrorRate: 0.03,
	}}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"compare",
		"--max-p95-regression", "20%",
		"--max-p999-regression", "20%",
		"--max-max-regression", "20%",
		"--max-error-rate-regression", "50%",
		"--error-rate-under", "2",
		"--min-request-count", "90",
		"--max-request-drop", "10%",
		"--p95-under", "120ms",
		"--p999-under", "200ms",
		"--max-under", "250ms",
		before,
		after,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit %d, stdout: %s stderr: %s", code, stdout.String(), stderr.String())
	}
	joined := stderr.String()
	for _, want := range []string{"p95 regression", "p999 regression", "max regression", "error rate regression", "request count", "request count drop"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in stderr: %s", want, joined)
		}
	}
}

func TestWatchCommandRunsOneWindow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"watch", "--window", "80ms", "--rps", "20", "--iterations", "1", "--clear=false", srv.URL}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "p99 watch") || !strings.Contains(stdout.String(), "Requests:") {
		t.Fatalf("unexpected watch output: %s", stdout.String())
	}
}

func TestProfileCommandDownloadsProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("seconds"); got != "1" {
			t.Fatalf("seconds got %q", got)
		}
		_, _ = w.Write([]byte("profile bytes"))
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "cpu.pprof")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"profile", "--seconds", "1", "--timeout", "1s", "--top=false", "--output", path, srv.URL + "/debug/pprof/profile"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "profile bytes" {
		t.Fatalf("profile body got %q", data)
	}
	if !strings.Contains(stdout.String(), "Saved:") {
		t.Fatalf("unexpected profile output: %s", stdout.String())
	}
}

func TestProfileCommandCorrelatesProbe(t *testing.T) {
	profileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("profile bytes"))
	}))
	defer profileSrv.Close()
	probeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer probeSrv.Close()

	path := filepath.Join(t.TempDir(), "cpu.pprof")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"profile",
		"--seconds", "1",
		"--timeout", "1s",
		"--top=false",
		"--output", path,
		"--probe", probeSrv.URL,
		"--probe-duration", "80ms",
		"--probe-rps", "20",
		profileSrv.URL + "/debug/pprof/profile",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Correlated HTTP probe") {
		t.Fatalf("missing correlated probe output: %s", stdout.String())
	}
}

func TestProfileCommandWithRuntime(t *testing.T) {
	profileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("profile bytes"))
	}))
	defer profileSrv.Close()
	runtimeSrv := cliRuntimeServer()
	defer runtimeSrv.Close()

	path := filepath.Join(t.TempDir(), "cpu.pprof")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"profile",
		"--seconds", "1",
		"--timeout", "1s",
		"--top=false",
		"--output", path,
		"--runtime", runtimeSrv.URL,
		profileSrv.URL + "/debug/pprof/profile",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Runtime signals:") {
		t.Fatalf("missing runtime output: %s", stdout.String())
	}
}

func TestRuntimeCommand(t *testing.T) {
	runtimeSrv := cliRuntimeServer()
	defer runtimeSrv.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"runtime", "--duration", "1ms", runtimeSrv.URL}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Runtime signals:") {
		t.Fatalf("missing runtime output: %s", stdout.String())
	}
}

func TestHTTPCommandWritesRuntimeJSON(t *testing.T) {
	probeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer probeSrv.Close()
	runtimeSrv := cliRuntimeServer()
	defer runtimeSrv.Close()

	path := filepath.Join(t.TempDir(), "run.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"http", "--duration", "80ms", "--rps", "20", "--runtime", runtimeSrv.URL, "--output", path, probeSrv.URL}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	result, err := output.ReadJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Runtime == nil || result.Runtime.Delta.Goroutines == nil {
		t.Fatalf("missing runtime correlation: %#v", result.Runtime)
	}
	if !strings.Contains(stdout.String(), "Runtime signals:") {
		t.Fatalf("missing runtime summary: %s", stdout.String())
	}
}

func cliRuntimeServer() *httptest.Server {
	var calls uint64
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/debug/vars" {
			calls++
			alloc := calls * 1024
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
				"db_pool_wait_count": %d,
				"db_pool_wait_duration_ns": %d
			}`, alloc, alloc*2, alloc*4, alloc*8, calls, calls*100, calls*10, calls, calls*1000)
			return
		}
		switch r.URL.Path {
		case "/debug/pprof/goroutine":
			fmt.Fprintf(w, "goroutine profile: total %d\n", 10+calls)
		case "/debug/pprof/mutex", "/debug/pprof/block":
			fmt.Fprintln(w, "1 100 @ 0x1 0x2")
		default:
			http.NotFound(w, r)
		}
	}))
}

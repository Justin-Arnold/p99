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

func TestReportCommandDetails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.json")
	if err := output.WriteJSONFile(path, probe.RunResult{
		Version: probe.ResultVersion,
		Summary: latency.Summary{
			Count:   2,
			Success: 1,
			Errors:  1,
			P99:     200 * time.Millisecond,
		},
		Histogram: []latency.Bucket{
			{UpperBoundNS: int64(100 * time.Millisecond), Count: 1},
			{UpperBoundNS: -1, Count: 1},
		},
		SlowSamples: []latency.SlowSample{{
			Latency:   200 * time.Millisecond,
			Timestamp: time.Unix(10, 0),
			Status:    500,
			Error:     "HTTP_5xx",
			Method:    "GET",
			URL:       "/slow",
		}},
		Errors: map[string]int{"HTTP_5xx": 1},
		Shape:  latency.Shape{Kind: "spiky", Notes: []string{"tail moved"}, Hints: []string{"check retries"}},
	}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"report", "--details", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	for _, want := range []string{"Histogram:", "Slow samples:", "Shape analysis:", "Detail:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("missing %q in output: %s", want, stdout.String())
		}
	}
}

func TestReportMarkdownIncludesDetails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.json")
	if err := output.WriteJSONFile(path, probe.RunResult{
		Version:     probe.ResultVersion,
		Summary:     latency.Summary{Count: 1, P99: 200 * time.Millisecond},
		Histogram:   []latency.Bucket{{UpperBoundNS: -1, Count: 1}},
		SlowSamples: []latency.SlowSample{{Latency: 200 * time.Millisecond, Timestamp: time.Unix(10, 0), Method: "GET", URL: "/slow"}},
		Shape:       latency.Shape{Kind: "bimodal", Notes: []string{"two bands"}},
	}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"report", "--format", "markdown", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	for _, want := range []string{"## Shape Analysis", "## Histogram", "## Slow Samples"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("missing %q in markdown: %s", want, stdout.String())
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

func TestVersionCommand(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, Date
	Version, Commit, Date = "v1.2.3", "abc123", "2026-05-25T00:00:00Z"
	defer func() {
		Version, Commit, Date = oldVersion, oldCommit, oldDate
	}()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	for _, want := range []string{"p99 v1.2.3", "commit: abc123", "built: 2026-05-25T00:00:00Z"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("missing %q in version output: %s", want, stdout.String())
		}
	}
}

func TestCommandHelp(t *testing.T) {
	for _, args := range [][]string{
		{"help", "http"},
		{"http", "--help"},
	} {
		var stdout, stderr bytes.Buffer
		code := Run(args, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("%v exit %d, stderr: %s", args, code, stderr.String())
		}
		for _, want := range []string{"Usage:", "p99 http [flags] URL", "--duration value", "--runtime-timeout value"} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("%v missing %q in help output: %s", args, want, stdout.String())
			}
		}
	}
}

func TestCompletionCommand(t *testing.T) {
	for _, tc := range []struct {
		shell    string
		want     string
		wantFlag string
	}{
		{"bash", "complete -F _p99_completion p99", "--duration"},
		{"zsh", "#compdef p99", "--duration"},
		{"fish", "complete -c p99", "-l duration"},
	} {
		var stdout, stderr bytes.Buffer
		code := Run([]string{"completion", tc.shell}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("%s exit %d, stderr: %s", tc.shell, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), tc.want) || !strings.Contains(stdout.String(), tc.wantFlag) {
			t.Fatalf("%s completion missing expected content: %s", tc.shell, stdout.String())
		}
	}
}

func TestCLINicetyValidation(t *testing.T) {
	dir := t.TempDir()
	before := filepath.Join(dir, "before.json")
	after := filepath.Join(dir, "after.json")
	if err := output.WriteJSONFile(before, probe.RunResult{Summary: latency.Summary{Count: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := output.WriteJSONFile(after, probe.RunResult{Summary: latency.Summary{Count: 1}}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"http rps", []string{"http", "--rps", "0", "http://example.com"}, "invalid rps"},
		{"watch iterations", []string{"watch", "--iterations", "-1", "http://example.com"}, "invalid iterations"},
		{"spans slow samples", []string{"spans", "--slow-samples", "-1", "spans.json"}, "invalid slow-samples"},
		{"compare min count", []string{"compare", "--min-request-count", "-1", before, after}, "invalid min-request-count"},
	}
	for _, tc := range tests {
		var stdout, stderr bytes.Buffer
		code := Run(tc.args, &stdout, &stderr)
		if code != 2 {
			t.Fatalf("%s exit %d, stdout: %s stderr: %s", tc.name, code, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), tc.want) {
			t.Fatalf("%s missing %q in stderr: %s", tc.name, tc.want, stderr.String())
		}
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

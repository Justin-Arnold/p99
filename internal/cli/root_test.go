package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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

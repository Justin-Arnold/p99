package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpansCommandReportsAndWritesJSON(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "spans.json")
	output := filepath.Join(dir, "report.json")
	if err := os.WriteFile(input, []byte(`[
	  {
	    "traceId": "trace-1",
	    "spanId": "root",
	    "name": "GET /checkout",
	    "kind": "SERVER",
	    "service": "api",
	    "startTimeUnixNano": "1000000000",
	    "endTimeUnixNano": "1300000000",
	    "attributes": {"http.route": "/checkout"}
	  },
	  {
	    "traceId": "trace-1",
	    "spanId": "redis",
	    "parentSpanId": "root",
	    "name": "GET cart",
	    "kind": "CLIENT",
	    "service": "api",
	    "startTimeUnixNano": "1050000000",
	    "endTimeUnixNano": "1250000000",
	    "attributes": {"peer.service": "redis"}
	  }
	]`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"spans", "--json-out", output, input}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Dependencies:") || !strings.Contains(stdout.String(), "redis") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"trace_count": 1`)) {
		t.Fatalf("unexpected json: %s", data)
	}
}

func TestSpansCommandExportsFormats(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "spans.json")
	if err := os.WriteFile(input, []byte(`[
	  {
	    "traceId": "trace-1",
	    "spanId": "root",
	    "name": "GET /checkout",
	    "kind": "SERVER",
	    "service": "api",
	    "startTimeUnixNano": "1000000000",
	    "endTimeUnixNano": "1300000000",
	    "attributes": {"http.route": "/checkout"}
	  }
	]`), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		format string
		want   string
	}{
		{"markdown", "# p99 span report"},
		{"prometheus", "p99_span_requests_total"},
		{"otel", `"resourceMetrics"`},
	} {
		var stdout, stderr bytes.Buffer
		code := Run([]string{"spans", "--format", tc.format, input}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("%s exit %d, stderr: %s", tc.format, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), tc.want) {
			t.Fatalf("%s output missing %q: %s", tc.format, tc.want, stdout.String())
		}
	}
}

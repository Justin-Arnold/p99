package spans

import (
	"strings"
	"testing"
	"time"
)

func TestParseOTLPResourceSpans(t *testing.T) {
	input := `{
	  "resourceSpans": [{
	    "resource": {"attributes": [{"key": "service.name", "value": {"stringValue": "api"}}]},
	    "scopeSpans": [{"spans": [
	      {
	        "traceId": "trace-1",
	        "spanId": "root",
	        "name": "GET /users/{id}",
	        "kind": "SPAN_KIND_SERVER",
	        "startTimeUnixNano": "1000000000",
	        "endTimeUnixNano": "1100000000",
	        "attributes": [
	          {"key": "http.route", "value": {"stringValue": "/users/{id}"}},
	          {"key": "http.response.status_code", "value": {"intValue": "200"}}
	        ]
	      },
	      {
	        "traceId": "trace-1",
	        "spanId": "db",
	        "parentSpanId": "root",
	        "name": "SELECT users",
	        "kind": 3,
	        "startTimeUnixNano": "1020000000",
	        "endTimeUnixNano": "1090000000",
	        "attributes": [
	          {"key": "db.system", "value": {"stringValue": "postgresql"}},
	          {"key": "db.name", "value": {"stringValue": "users"}}
	        ]
	      }
	    ]}]
	  }]
	}`
	got, warnings, err := ParseJSON(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings got %#v", warnings)
	}
	if len(got) != 2 {
		t.Fatalf("span count got %d", len(got))
	}
	if got[0].Service != "api" {
		t.Fatalf("service got %q", got[0].Service)
	}
	if got[1].Kind != "CLIENT" {
		t.Fatalf("kind got %q", got[1].Kind)
	}
	if got[0].Duration() != 100*time.Millisecond {
		t.Fatalf("duration got %s", got[0].Duration())
	}
}

func TestAnalyzeRoutesDependenciesAndSlowTraces(t *testing.T) {
	base := time.Unix(0, 0)
	report := Analyze([]Span{
		{
			TraceID: "t1", SpanID: "root", Name: "GET /search", Kind: "SERVER", Service: "api",
			Start: base, End: base.Add(300 * time.Millisecond),
			Attributes: map[string]string{"http.route": "/search"},
		},
		{
			TraceID: "t1", SpanID: "db", ParentSpanID: "root", Name: "SELECT", Kind: "CLIENT", Service: "api",
			Start: base.Add(20 * time.Millisecond), End: base.Add(220 * time.Millisecond),
			Attributes: map[string]string{"db.system": "postgresql", "db.name": "catalog"},
		},
		{
			TraceID: "t2", SpanID: "root2", Name: "GET /search", Kind: "SERVER", Service: "api",
			Start: base.Add(time.Second), End: base.Add(time.Second + 100*time.Millisecond),
			Attributes: map[string]string{"http.route": "/search", "http.response.status_code": "500"},
		},
	}, nil, AnalyzeOptions{SlowSamples: 1})

	if report.RequestCount != 2 {
		t.Fatalf("request count got %d", report.RequestCount)
	}
	if report.Summary.Errors != 1 {
		t.Fatalf("errors got %d", report.Summary.Errors)
	}
	if len(report.Routes) != 1 || report.Routes[0].Name != "/search" {
		t.Fatalf("routes got %#v", report.Routes)
	}
	if report.Routes[0].Summary.Errors != 1 {
		t.Fatalf("route errors got %d", report.Routes[0].Summary.Errors)
	}
	if len(report.Dependencies) != 1 || report.Dependencies[0].Name != "db:postgresql:catalog" {
		t.Fatalf("dependencies got %#v", report.Dependencies)
	}
	if len(report.SlowTraces) != 1 || report.SlowTraces[0].TraceID != "t1" {
		t.Fatalf("slow traces got %#v", report.SlowTraces)
	}
}

func TestAnalyzeAllowsDisablingSlowTraces(t *testing.T) {
	base := time.Unix(0, 0)
	report := Analyze([]Span{{
		TraceID:    "t1",
		SpanID:     "root",
		Name:       "GET /search",
		Kind:       "SERVER",
		Service:    "api",
		Start:      base,
		End:        base.Add(300 * time.Millisecond),
		Attributes: map[string]string{"http.route": "/search"},
	}}, nil, AnalyzeOptions{SlowSamples: 0})

	if len(report.SlowTraces) != 0 {
		t.Fatalf("slow traces got %#v, want none", report.SlowTraces)
	}
}

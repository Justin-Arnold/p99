package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPRunnerSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	result, err := HTTPRunner{Config: HTTPConfig{
		URL:         srv.URL,
		Method:      http.MethodGet,
		Duration:    120 * time.Millisecond,
		RPS:         20,
		Concurrency: 2,
		Timeout:     time.Second,
		SlowSamples: 3,
	}}.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Count == 0 {
		t.Fatal("expected recorded requests")
	}
	if result.Summary.Errors != 0 {
		t.Fatalf("errors got %d, want 0", result.Summary.Errors)
	}
	if len(result.SlowSamples) == 0 {
		t.Fatal("expected slow samples")
	}
}

func TestHTTPRunner500Classified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	result, err := HTTPRunner{Config: HTTPConfig{
		URL:         srv.URL,
		Method:      http.MethodGet,
		Duration:    80 * time.Millisecond,
		RPS:         20,
		Concurrency: 1,
		Timeout:     time.Second,
		SlowSamples: 2,
	}}.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Errors["HTTP_5xx"] == 0 {
		t.Fatalf("expected HTTP_5xx errors, got %#v", result.Errors)
	}
}

func TestHTTPRunnerTimeoutClassified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	result, err := HTTPRunner{Config: HTTPConfig{
		URL:         srv.URL,
		Method:      http.MethodGet,
		Duration:    80 * time.Millisecond,
		RPS:         20,
		Concurrency: 1,
		Timeout:     20 * time.Millisecond,
		SlowSamples: 2,
	}}.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Errors["timeout"] == 0 {
		t.Fatalf("expected timeout errors, got %#v", result.Errors)
	}
}

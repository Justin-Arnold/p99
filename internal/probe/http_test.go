package probe

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

func TestHTTPRunnerSendsBodyHeadersAndAcceptsExpectedStatus(t *testing.T) {
	seen := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		seen <- r.Header.Get("X-P99-Test") + "|" + string(body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	result, err := HTTPRunner{
		Config: HTTPConfig{
			URL:            srv.URL,
			Method:         http.MethodPost,
			Duration:       80 * time.Millisecond,
			RPS:            10,
			Concurrency:    1,
			Timeout:        time.Second,
			Headers:        map[string]string{"X-P99-Test": "present"},
			ExpectedStatus: []int{http.StatusCreated},
			SlowSamples:    2,
		},
		Body: []byte(`{"ok":true}`),
	}.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Errors != 0 {
		t.Fatalf("errors got %d, breakdown %#v", result.Summary.Errors, result.Errors)
	}
	got := <-seen
	if got != `present|{"ok":true}` {
		t.Fatalf("server saw %q", got)
	}
}

func TestHTTPRunnerBodyReadErrorClassified(t *testing.T) {
	result, err := HTTPRunner{
		Config: HTTPConfig{
			URL:         "http://example.test",
			Method:      http.MethodGet,
			Duration:    80 * time.Millisecond,
			RPS:         10,
			Concurrency: 1,
			Timeout:     time.Second,
			SlowSamples: 2,
		},
		Client: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     make(http.Header),
					Body:       errReader{},
					Request:    req,
				}, nil
			}),
		},
	}.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Errors["body/read_error"] == 0 {
		t.Fatalf("expected body/read_error, got %#v", result.Errors)
	}
}

func TestHTTPRunnerTLSFailureClassified(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	result, err := HTTPRunner{Config: HTTPConfig{
		URL:         srv.URL,
		Method:      http.MethodGet,
		Duration:    80 * time.Millisecond,
		RPS:         10,
		Concurrency: 1,
		Timeout:     time.Second,
		SlowSamples: 2,
	}}.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Errors["TLS"] == 0 {
		t.Fatalf("expected TLS errors, got %#v", result.Errors)
	}
}

func TestHTTPRunnerMixedLatencyDistribution(t *testing.T) {
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1)%5 == 0 {
			time.Sleep(20 * time.Millisecond)
		} else {
			time.Sleep(time.Millisecond)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	result, err := HTTPRunner{Config: HTTPConfig{
		URL:         srv.URL,
		Method:      http.MethodGet,
		Duration:    300 * time.Millisecond,
		RPS:         120,
		Concurrency: 4,
		Timeout:     time.Second,
		SlowSamples: 5,
	}}.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Count < 20 {
		t.Fatalf("request count got %d, want at least 20", result.Summary.Count)
	}
	if result.Summary.Max <= result.Summary.Min {
		t.Fatalf("expected mixed latencies, min=%s max=%s", result.Summary.Min, result.Summary.Max)
	}
	if len(result.SlowSamples) == 0 {
		t.Fatal("expected slow samples")
	}
	for i := 1; i < len(result.SlowSamples); i++ {
		if result.SlowSamples[i].Latency > result.SlowSamples[i-1].Latency {
			t.Fatalf("slow samples not sorted descending: %#v", result.SlowSamples)
		}
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func (errReader) Close() error {
	return nil
}

func TestSampleURLIncludesPathAndQuery(t *testing.T) {
	got := sampleURL("https://example.test/search?q=tail")
	if !strings.Contains(got, "/search?q=tail") {
		t.Fatalf("sample url got %q", got)
	}
}

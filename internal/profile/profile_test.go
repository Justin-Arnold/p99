package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFetcherDownloadsProfile(t *testing.T) {
	var seconds string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seconds = r.URL.Query().Get("seconds")
		_, _ = w.Write([]byte("profile bytes"))
	}))
	defer srv.Close()

	path := t.TempDir() + "/cpu.pprof"
	result, err := Fetcher{
		Config: Config{URL: srv.URL + "/debug/pprof/profile", Seconds: 5, Timeout: time.Second, Output: path, Top: false, TopCount: 5},
	}.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if seconds != "5" {
		t.Fatalf("seconds query got %q", seconds)
	}
	if result.Bytes != int64(len("profile bytes")) {
		t.Fatalf("bytes got %d", result.Bytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "profile bytes" {
		t.Fatalf("profile body got %q", data)
	}
}

func TestFetcherRunsTop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("profile bytes"))
	}))
	defer srv.Close()

	result, err := Fetcher{
		Config: Config{URL: srv.URL, Seconds: 1, Timeout: time.Second, Output: t.TempDir() + "/cpu.pprof", Top: true, TopCount: 3},
		TopRunner: func(ctx context.Context, profilePath string, count int) (string, error) {
			if count != 3 {
				t.Fatalf("count got %d", count)
			}
			if !strings.HasSuffix(profilePath, "cpu.pprof") {
				t.Fatalf("profile path got %q", profilePath)
			}
			return "top output", nil
		},
	}.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Top != "top output" {
		t.Fatalf("top got %q", result.Top)
	}
}

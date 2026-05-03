package profile

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type Config struct {
	URL       string        `json:"url"`
	Seconds   int           `json:"seconds"`
	Timeout   time.Duration `json:"timeout_ns"`
	Output    string        `json:"output"`
	Top       bool          `json:"top"`
	TopCount  int           `json:"top_count"`
	UserAgent string        `json:"user_agent,omitempty"`
}

type Result struct {
	URL        string    `json:"url"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
	OutputPath string    `json:"output_path"`
	Bytes      int64     `json:"bytes"`
	Top        string    `json:"top,omitempty"`
	TopError   string    `json:"top_error,omitempty"`
}

type TopRunner func(ctx context.Context, profilePath string, count int) (string, error)

type Fetcher struct {
	Config    Config
	Client    *http.Client
	TopRunner TopRunner
}

func (f Fetcher) Fetch(ctx context.Context) (Result, error) {
	if err := f.validate(); err != nil {
		return Result{}, err
	}

	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: f.Config.Timeout}
	}
	if client.Timeout == 0 {
		client.Timeout = f.Config.Timeout
	}

	outputPath, cleanup, err := outputPath(f.Config.Output)
	if err != nil {
		return Result{}, err
	}
	defer cleanup(false)

	started := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, profileURL(f.Config.URL, f.Config.Seconds), nil)
	if err != nil {
		return Result{}, err
	}
	if f.Config.UserAgent != "" {
		req.Header.Set("User-Agent", f.Config.UserAgent)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("profile endpoint returned %s", resp.Status)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return Result{}, err
	}
	written, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		return Result{}, copyErr
	}
	if closeErr != nil {
		return Result{}, closeErr
	}

	result := Result{
		URL:        req.URL.String(),
		StartedAt:  started,
		EndedAt:    time.Now(),
		OutputPath: outputPath,
		Bytes:      written,
	}

	if f.Config.Top {
		runner := f.TopRunner
		if runner == nil {
			runner = GoToolTop
		}
		top, err := runner(ctx, outputPath, f.Config.TopCount)
		if err != nil {
			result.TopError = err.Error()
		} else {
			result.Top = top
		}
	}

	cleanup(true)
	return result, nil
}

func (f Fetcher) validate() error {
	if f.Config.URL == "" {
		return fmt.Errorf("profile url is required")
	}
	if _, err := url.ParseRequestURI(f.Config.URL); err != nil {
		return fmt.Errorf("invalid profile url: %w", err)
	}
	if f.Config.Seconds <= 0 {
		return fmt.Errorf("seconds must be positive")
	}
	if f.Config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if f.Config.TopCount <= 0 {
		return fmt.Errorf("top count must be positive")
	}
	return nil
}

func profileURL(raw string, seconds int) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	if q.Get("seconds") == "" {
		q.Set("seconds", strconv.Itoa(seconds))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func outputPath(path string) (string, func(keep bool), error) {
	if path != "" {
		return path, func(bool) {}, nil
	}
	f, err := os.CreateTemp("", "p99-profile-*.pprof")
	if err != nil {
		return "", nil, err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		return "", nil, err
	}
	return name, func(keep bool) {
		if !keep {
			_ = os.Remove(name)
		}
	}, nil
}

func GoToolTop(ctx context.Context, profilePath string, count int) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "tool", "pprof", "-top", "-nodecount", strconv.Itoa(count), profilePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, string(out))
	}
	return string(out), nil
}

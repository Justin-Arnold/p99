package requestspec

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadYAMLAndRenderWithDatasetsChoicesEnvAndBodyFile(t *testing.T) {
	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "body.json.tpl")
	if err := os.WriteFile(bodyPath, []byte(`{"q":"{{ term }}","filter":"{{ filter }}","token":"{{ env.P99_TOKEN }}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(dir, "requests.yaml")
	if err := os.WriteFile(specPath, []byte(`
version: 1
base_url: http://example.test
defaults:
  headers:
    Authorization: "Bearer {{ env.P99_TOKEN }}"
  expect_status: [200, 201]
datasets:
  terms: [alpha, beta]
requests:
  - name: search
    weight: 3
    method: POST
    path: /search
    query:
      q: "{{ term }}"
      filter: "{{ filter }}"
    headers:
      X-Term: "{{ term }}"
    vars:
      term:
        dataset: terms
      filter:
        choices: [recent, popular]
    body_file: body.json.tpl
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("P99_TOKEN", "secret")

	plan, err := Load(specPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if plan.TotalWeight != 3 {
		t.Fatalf("weight got %d, want 3", plan.TotalWeight)
	}
	req, err := plan.Render(rand.New(rand.NewSource(4)))
	if err != nil {
		t.Fatal(err)
	}
	if req.Name != "search" || req.Method != "POST" {
		t.Fatalf("unexpected request: %#v", req)
	}
	if !strings.HasPrefix(req.URL, "http://example.test/search?") {
		t.Fatalf("url got %q", req.URL)
	}
	if !strings.Contains(req.URL, "filter=") || !strings.Contains(req.URL, "q=") {
		t.Fatalf("url missing query values: %q", req.URL)
	}
	if req.Headers["Authorization"] != "Bearer secret" {
		t.Fatalf("authorization header got %q", req.Headers["Authorization"])
	}
	if !strings.Contains(string(req.Body), `"token":"secret"`) {
		t.Fatalf("body did not render env placeholder: %s", req.Body)
	}
	if len(req.ExpectedStatus) != 2 {
		t.Fatalf("expected statuses got %#v", req.ExpectedStatus)
	}
}

func TestLoadJSONWithBaseOverride(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "requests.json")
	if err := os.WriteFile(specPath, []byte(`{
  "version": 1,
  "base_url": "http://wrong.test",
  "requests": [
    {"name": "detail", "method": "GET", "path": "/users/123"}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := Load(specPath, "http://right.test")
	if err != nil {
		t.Fatal(err)
	}
	req, err := plan.Render(rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "http://right.test/users/123" {
		t.Fatalf("url got %q", req.URL)
	}
}

func TestValidationErrors(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		spec string
		want string
	}{
		{
			name: "duplicate",
			spec: `version: 1
base_url: http://example.test
requests:
  - {name: a, method: GET, path: /a}
  - {name: a, method: GET, path: /b}
`,
			want: "duplicate request name",
		},
		{
			name: "bad weight",
			spec: `version: 1
base_url: http://example.test
requests:
  - {name: a, weight: -1, method: GET, path: /a}
`,
			want: "weight must be positive",
		},
		{
			name: "unknown placeholder",
			spec: `version: 1
base_url: http://example.test
requests:
  - {name: a, method: GET, path: "/a?q={{ missing }}"}
`,
			want: "unknown placeholder",
		},
		{
			name: "empty random source",
			spec: `version: 1
base_url: http://example.test
requests:
  - name: a
    method: GET
    path: /a
    vars:
      term:
        choices: []
`,
			want: "must use dataset or choices",
		},
		{
			name: "missing body file",
			spec: `version: 1
base_url: http://example.test
requests:
  - {name: a, method: POST, path: /a, body_file: missing.json}
`,
			want: "read body_file",
		},
		{
			name: "bad url",
			spec: `version: 1
requests:
  - {name: a, method: GET, url: "://bad"}
`,
			want: "invalid url",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".yaml")
			if err := os.WriteFile(path, []byte(tc.spec), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path, "")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error got %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestSeededRenderingIsDeterministic(t *testing.T) {
	spec := Spec{
		Version: Version,
		BaseURL: "http://example.test",
		Datasets: map[string][]string{
			"terms": []string{"alpha", "beta", "gamma"},
		},
		Requests: []Request{{
			Name:   "search",
			Weight: 1,
			Method: "GET",
			Path:   "/search",
			Query:  map[string]string{"q": "{{ term }}"},
			Vars:   map[string]Var{"term": {Dataset: "terms"}},
		}},
	}
	plan, err := Compile(spec, "requests.yaml", "")
	if err != nil {
		t.Fatal(err)
	}
	a, err := plan.Render(rand.New(rand.NewSource(99)))
	if err != nil {
		t.Fatal(err)
	}
	b, err := plan.Render(rand.New(rand.NewSource(99)))
	if err != nil {
		t.Fatal(err)
	}
	if a.URL != b.URL {
		t.Fatalf("seeded render differed: %q vs %q", a.URL, b.URL)
	}
}

func TestPathTemplateRendersAfterValidation(t *testing.T) {
	spec := Spec{
		Version: Version,
		BaseURL: "http://example.test",
		Requests: []Request{{
			Name:   "detail",
			Method: "GET",
			Path:   "/users/{{ id }}",
			Vars:   map[string]Var{"id": {Choices: []string{"123"}}},
		}},
	}
	plan, err := Compile(spec, "requests.yaml", "")
	if err != nil {
		t.Fatal(err)
	}
	req, err := plan.Render(rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "http://example.test/users/123" {
		t.Fatalf("url got %q", req.URL)
	}
}

package requestspec

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const Version = 1

var placeholderRE = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

type Spec struct {
	Version  int                 `json:"version" yaml:"version"`
	BaseURL  string              `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Defaults Defaults            `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Datasets map[string][]string `json:"datasets,omitempty" yaml:"datasets,omitempty"`
	Requests []Request           `json:"requests" yaml:"requests"`
}

type Defaults struct {
	Headers      map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	ExpectStatus []int             `json:"expect_status,omitempty" yaml:"expect_status,omitempty"`
}

type Request struct {
	Name         string            `json:"name" yaml:"name"`
	Weight       int               `json:"weight,omitempty" yaml:"weight,omitempty"`
	Method       string            `json:"method" yaml:"method"`
	URL          string            `json:"url,omitempty" yaml:"url,omitempty"`
	Path         string            `json:"path,omitempty" yaml:"path,omitempty"`
	Query        map[string]string `json:"query,omitempty" yaml:"query,omitempty"`
	Headers      map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body         string            `json:"body,omitempty" yaml:"body,omitempty"`
	BodyFile     string            `json:"body_file,omitempty" yaml:"body_file,omitempty"`
	ExpectStatus []int             `json:"expect_status,omitempty" yaml:"expect_status,omitempty"`
	Vars         map[string]Var    `json:"vars,omitempty" yaml:"vars,omitempty"`
}

type Var struct {
	Dataset string   `json:"dataset,omitempty" yaml:"dataset,omitempty"`
	Choices []string `json:"choices,omitempty" yaml:"choices,omitempty"`
}

type Plan struct {
	Source      string
	BaseURL     string
	Requests    []Template
	TotalWeight int
}

type Template struct {
	Name         string
	Weight       int
	Method       string
	urlTemplate  string
	query        map[string]string
	headers      map[string]string
	bodyTemplate string
	statuses     []int
	vars         map[string]compiledVar
}

type compiledVar struct {
	values []string
}

type RenderedRequest struct {
	Name           string
	Weight         int
	Method         string
	URL            string
	Headers        map[string]string
	Body           []byte
	ExpectedStatus []int
}

func Load(path, baseOverride string) (Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, err
	}
	var spec Spec
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		err = json.Unmarshal(data, &spec)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &spec)
	default:
		return Plan{}, fmt.Errorf("request spec must be .json, .yaml, or .yml")
	}
	if err != nil {
		return Plan{}, err
	}
	return Compile(spec, path, baseOverride)
}

func Compile(spec Spec, source, baseOverride string) (Plan, error) {
	if spec.Version != Version {
		return Plan{}, fmt.Errorf("request spec version must be %d", Version)
	}
	if len(spec.Requests) == 0 {
		return Plan{}, fmt.Errorf("request spec requires at least one request")
	}
	baseURL := strings.TrimSpace(spec.BaseURL)
	if strings.TrimSpace(baseOverride) != "" {
		baseURL = strings.TrimSpace(baseOverride)
	}
	if baseURL != "" {
		if _, err := url.ParseRequestURI(baseURL); err != nil {
			return Plan{}, fmt.Errorf("invalid base_url: %w", err)
		}
	}

	seen := map[string]bool{}
	templates := make([]Template, 0, len(spec.Requests))
	totalWeight := 0
	for _, req := range spec.Requests {
		tpl, err := compileRequest(req, spec, source, baseURL, seen)
		if err != nil {
			return Plan{}, err
		}
		templates = append(templates, tpl)
		totalWeight += tpl.Weight
	}
	return Plan{Source: source, BaseURL: baseURL, Requests: templates, TotalWeight: totalWeight}, nil
}

func compileRequest(req Request, spec Spec, source, baseURL string, seen map[string]bool) (Template, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Template{}, fmt.Errorf("request name is required")
	}
	if seen[name] {
		return Template{}, fmt.Errorf("duplicate request name %q", name)
	}
	seen[name] = true

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		return Template{}, fmt.Errorf("request %q method is required", name)
	}
	if !validMethod(method) {
		return Template{}, fmt.Errorf("request %q has invalid method %q", name, method)
	}
	weight := req.Weight
	if weight == 0 {
		weight = 1
	}
	if weight <= 0 {
		return Template{}, fmt.Errorf("request %q weight must be positive", name)
	}
	if req.URL != "" && req.Path != "" {
		return Template{}, fmt.Errorf("request %q must use url or path, not both", name)
	}
	if req.URL == "" && req.Path == "" {
		return Template{}, fmt.Errorf("request %q requires url or path", name)
	}
	if req.Body != "" && req.BodyFile != "" {
		return Template{}, fmt.Errorf("request %q must use body or body_file, not both", name)
	}

	urlTemplate, err := requestURL(req, baseURL)
	if err != nil {
		return Template{}, fmt.Errorf("request %q: %w", name, err)
	}
	headers := mergeHeaders(spec.Defaults.Headers, req.Headers)
	statuses := req.ExpectStatus
	if len(statuses) == 0 {
		statuses = spec.Defaults.ExpectStatus
	}
	if err := validateStatuses(name, statuses); err != nil {
		return Template{}, err
	}

	bodyTemplate := req.Body
	if req.BodyFile != "" {
		path := req.BodyFile
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(source), path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return Template{}, fmt.Errorf("request %q read body_file: %w", name, err)
		}
		bodyTemplate = string(data)
	}

	vars, err := compileVars(name, req.Vars, spec.Datasets)
	if err != nil {
		return Template{}, err
	}
	fields := []string{urlTemplate, bodyTemplate}
	for _, v := range req.Query {
		fields = append(fields, v)
	}
	for _, v := range headers {
		fields = append(fields, v)
	}
	if err := validatePlaceholders(name, fields, vars); err != nil {
		return Template{}, err
	}

	return Template{
		Name:         name,
		Weight:       weight,
		Method:       method,
		urlTemplate:  urlTemplate,
		query:        cloneStringMap(req.Query),
		headers:      headers,
		bodyTemplate: bodyTemplate,
		statuses:     append([]int(nil), statuses...),
		vars:         vars,
	}, nil
}

func requestURL(req Request, baseURL string) (string, error) {
	if req.URL != "" {
		parsed, err := url.ParseRequestURI(sanitizePlaceholders(req.URL))
		if err != nil {
			return "", fmt.Errorf("invalid url: %w", err)
		}
		if !parsed.IsAbs() || parsed.Host == "" {
			return "", fmt.Errorf("url must be absolute")
		}
		return req.URL, nil
	}
	if baseURL == "" {
		return "", fmt.Errorf("base_url is required when using path")
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	ref, err := url.Parse(sanitizePlaceholders(req.Path))
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if ref.IsAbs() {
		return "", fmt.Errorf("path must be relative")
	}
	basePath := strings.TrimRight(base.String(), "/")
	rawPath := req.Path
	if !strings.HasPrefix(rawPath, "/") {
		rawPath = "/" + rawPath
	}
	_ = base.ResolveReference(ref)
	return basePath + rawPath, nil
}

func compileVars(name string, vars map[string]Var, datasets map[string][]string) (map[string]compiledVar, error) {
	out := map[string]compiledVar{}
	for key, v := range vars {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("request %q has an empty var name", name)
		}
		hasDataset := strings.TrimSpace(v.Dataset) != ""
		hasChoices := len(v.Choices) > 0
		if hasDataset == hasChoices {
			return nil, fmt.Errorf("request %q var %q must use dataset or choices", name, key)
		}
		values := append([]string(nil), v.Choices...)
		if hasDataset {
			ds, ok := datasets[v.Dataset]
			if !ok {
				return nil, fmt.Errorf("request %q var %q references missing dataset %q", name, key, v.Dataset)
			}
			values = append([]string(nil), ds...)
		}
		if len(values) == 0 {
			return nil, fmt.Errorf("request %q var %q has no values", name, key)
		}
		for _, value := range values {
			if value == "" {
				return nil, fmt.Errorf("request %q var %q contains an empty value", name, key)
			}
		}
		out[key] = compiledVar{values: values}
	}
	return out, nil
}

func validatePlaceholders(name string, fields []string, vars map[string]compiledVar) error {
	for _, field := range fields {
		for _, match := range placeholderRE.FindAllStringSubmatch(field, -1) {
			key := match[1]
			if strings.HasPrefix(key, "env.") {
				if strings.TrimPrefix(key, "env.") == "" {
					return fmt.Errorf("request %q has invalid env placeholder", name)
				}
				continue
			}
			if _, ok := vars[key]; !ok {
				return fmt.Errorf("request %q references unknown placeholder %q", name, key)
			}
		}
	}
	return nil
}

func validateStatuses(name string, statuses []int) error {
	for _, status := range statuses {
		if status < 100 || status > 599 {
			return fmt.Errorf("request %q has invalid status code %d", name, status)
		}
	}
	return nil
}

func (p Plan) Render(rng *rand.Rand) (RenderedRequest, error) {
	if len(p.Requests) == 0 || p.TotalWeight <= 0 {
		return RenderedRequest{}, fmt.Errorf("request plan is empty")
	}
	n := rng.Intn(p.TotalWeight)
	for _, req := range p.Requests {
		if n < req.Weight {
			return req.Render(rng)
		}
		n -= req.Weight
	}
	return p.Requests[len(p.Requests)-1].Render(rng)
}

func (t Template) Render(rng *rand.Rand) (RenderedRequest, error) {
	values := map[string]string{}
	for name, v := range t.vars {
		values[name] = v.values[rng.Intn(len(v.values))]
	}
	rawURL := renderTemplate(t.urlTemplate, values)
	u, err := url.Parse(rawURL)
	if err != nil {
		return RenderedRequest{}, err
	}
	q := u.Query()
	for key, value := range t.query {
		q.Set(key, renderTemplate(value, values))
	}
	u.RawQuery = q.Encode()
	headers := map[string]string{}
	for key, value := range t.headers {
		headers[key] = renderTemplate(value, values)
	}
	body := []byte(renderTemplate(t.bodyTemplate, values))
	return RenderedRequest{
		Name:           t.Name,
		Weight:         t.Weight,
		Method:         t.Method,
		URL:            u.String(),
		Headers:        headers,
		Body:           body,
		ExpectedStatus: append([]int(nil), t.statuses...),
	}, nil
}

func renderTemplate(text string, values map[string]string) string {
	return placeholderRE.ReplaceAllStringFunc(text, func(match string) string {
		parts := placeholderRE.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		key := parts[1]
		if strings.HasPrefix(key, "env.") {
			return os.Getenv(strings.TrimPrefix(key, "env."))
		}
		return values[key]
	})
}

func sanitizePlaceholders(text string) string {
	return placeholderRE.ReplaceAllString(text, "x")
}

func mergeHeaders(defaults, overrides map[string]string) map[string]string {
	out := cloneStringMap(defaults)
	for key, value := range overrides {
		out[key] = value
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func validMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

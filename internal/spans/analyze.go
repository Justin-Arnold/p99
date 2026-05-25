package spans

import (
	"sort"
	"strings"
	"time"

	"github.com/Justin-Arnold/p99/internal/latency"
)

type AnalyzeOptions struct {
	Source      string
	SlowSamples int
}

type traceData struct {
	id    string
	spans []Span
	root  Span
}

func Analyze(spans []Span, warnings []string, opts AnalyzeOptions) Report {
	// Span data is already post-facto evidence, so analysis favors grouping and
	// comparison over trying to infer exclusive time from potentially overlapping
	// spans.
	traces := groupTraces(spans)
	requests := requestTraces(traces)
	requestHist := latency.NewHistogram()
	routeHists := map[string]*latency.Histogram{}
	routeCounts := map[string]int{}
	routeFailures := map[string]int{}
	serviceHists := map[string]*latency.Histogram{}
	serviceCounts := map[string]int{}
	serviceFailures := map[string]int{}
	depHists := map[string]*latency.Histogram{}
	depTotals := map[string]time.Duration{}
	depCounts := map[string]int{}
	depTypes := map[string]string{}
	depServices := map[string]string{}
	var success, failures int
	var totalRequestTime time.Duration
	for _, trace := range requests {
		root := trace.root
		duration := root.Duration()
		if duration <= 0 {
			warnings = append(warnings, "trace "+trace.id+" has a non-positive request duration")
			continue
		}
		requestHist.Record(duration)
		totalRequestTime += duration
		failed := spanFailed(root)
		if failed {
			failures++
		} else {
			success++
		}

		route := routeName(root)
		recordGroup(routeHists, routeCounts, route, duration)
		if failed {
			routeFailures[route]++
		}
		service := serviceName(root)
		recordGroup(serviceHists, serviceCounts, service, duration)
		if failed {
			serviceFailures[service]++
		}

		var depTime time.Duration
		var depSamples []DependencySample
		for _, span := range trace.spans {
			if span.SpanID == root.SpanID || !isDependency(span) {
				continue
			}
			name, depType := dependencyName(span)
			d := span.Duration()
			if d <= 0 {
				continue
			}
			if depHists[name] == nil {
				depHists[name] = latency.NewHistogram()
			}
			depHists[name].Record(d)
			depTotals[name] += d
			depCounts[name]++
			depTypes[name] = depType
			depServices[name] = span.Service
			// Dependency spans can overlap. This total is represented dependency
			// time, not exclusive wall-clock time.
			depTime += d
			depSamples = append(depSamples, DependencySample{Name: name, Type: depType, Duration: d})
		}
		sort.Slice(depSamples, func(i, j int) bool { return depSamples[i].Duration > depSamples[j].Duration })
		if len(depSamples) > 5 {
			depSamples = depSamples[:5]
		}
	}

	report := Report{
		Version:      ReportVersion,
		Source:       opts.Source,
		TraceCount:   len(traces),
		SpanCount:    len(spans),
		RequestCount: requestHist.Count(),
		Summary:      latency.Summarize(requestHist, success, failures),
		Routes:       groups(routeHists, routeCounts, routeFailures),
		Services:     groups(serviceHists, serviceCounts, serviceFailures),
		Dependencies: dependencies(depHists, depCounts, depTotals, depTypes, depServices, totalRequestTime),
		Warnings:     warnings,
	}
	report.SlowTraces = slowTraces(requests, opts.SlowSamples)
	report.Hints = hints(report)
	return report
}

func groupTraces(spans []Span) map[string][]Span {
	traces := map[string][]Span{}
	for _, span := range spans {
		if span.TraceID == "" || span.SpanID == "" {
			continue
		}
		traces[span.TraceID] = append(traces[span.TraceID], span)
	}
	for traceID := range traces {
		sort.Slice(traces[traceID], func(i, j int) bool {
			return traces[traceID][i].Start.Before(traces[traceID][j].Start)
		})
	}
	return traces
}

func requestTraces(traces map[string][]Span) []traceData {
	out := make([]traceData, 0, len(traces))
	for traceID, traceSpans := range traces {
		root, ok := requestRoot(traceSpans)
		if !ok {
			continue
		}
		out = append(out, traceData{id: traceID, spans: traceSpans, root: root})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].root.Start.Before(out[j].root.Start) })
	return out
}

func requestRoot(spans []Span) (Span, bool) {
	var fallback Span
	var haveFallback bool
	for _, span := range spans {
		if span.Kind == "SERVER" {
			return span, true
		}
		if span.ParentSpanID == "" && (!haveFallback || span.Duration() > fallback.Duration()) {
			// Some exported traces omit span kind. The longest root span is usually
			// the closest available proxy for the request boundary.
			fallback = span
			haveFallback = true
		}
	}
	return fallback, haveFallback
}

func isDependency(span Span) bool {
	switch span.Kind {
	case "CLIENT", "PRODUCER", "CONSUMER":
		return true
	}
	// Attribute fallback keeps older or normalized trace exports useful even when
	// span kind is missing or vendor-specific.
	if span.Attributes["db.system"] != "" || span.Attributes["rpc.system"] != "" || span.Attributes["peer.service"] != "" {
		return true
	}
	return false
}

func dependencyName(span Span) (string, string) {
	if db := span.Attributes["db.system"]; db != "" {
		target := firstNonEmpty(span.Attributes["db.name"], span.Attributes["db.namespace"], span.Attributes["server.address"], span.Attributes["net.peer.name"])
		return joinName("db", db, target), "db"
	}
	if rpc := span.Attributes["rpc.system"]; rpc != "" {
		target := firstNonEmpty(span.Attributes["rpc.service"], span.Attributes["peer.service"], span.Attributes["server.address"])
		return joinName("rpc", rpc, target), "rpc"
	}
	if peer := span.Attributes["peer.service"]; peer != "" {
		return peer, "peer"
	}
	if target := firstNonEmpty(span.Attributes["server.address"], span.Attributes["net.peer.name"], span.Attributes["http.host"], span.Attributes["url.domain"]); target != "" {
		return target, strings.ToLower(firstNonEmpty(span.Kind, "dependency"))
	}
	return span.Name, strings.ToLower(firstNonEmpty(span.Kind, "dependency"))
}

func routeName(span Span) string {
	return firstNonEmpty(
		span.Attributes["http.route"],
		span.Attributes["url.path"],
		span.Attributes["http.target"],
		span.Attributes["http.url"],
		span.Name,
		"unknown",
	)
}

func serviceName(span Span) string {
	return firstNonEmpty(span.Service, span.Attributes["service.name"], "unknown")
}

func spanFailed(span Span) bool {
	if strings.EqualFold(span.StatusCode, "ERROR") || span.StatusCode == "2" {
		return true
	}
	code := firstNonEmpty(span.Attributes["http.response.status_code"], span.Attributes["http.status_code"])
	return strings.HasPrefix(code, "5") || strings.HasPrefix(code, "4")
}

func failureClass(span Span) string {
	if !spanFailed(span) {
		return ""
	}
	if span.StatusCode != "" {
		return span.StatusCode
	}
	return firstNonEmpty(span.Attributes["http.response.status_code"], span.Attributes["http.status_code"], "error")
}

func recordGroup(hists map[string]*latency.Histogram, counts map[string]int, name string, d time.Duration) {
	if hists[name] == nil {
		hists[name] = latency.NewHistogram()
	}
	hists[name].Record(d)
	counts[name]++
}

func groups(hists map[string]*latency.Histogram, counts, failures map[string]int) []GroupSummary {
	out := make([]GroupSummary, 0, len(hists))
	for name, hist := range hists {
		errorCount := failures[name]
		out = append(out, GroupSummary{Name: name, Count: counts[name], Summary: latency.Summarize(hist, counts[name]-errorCount, errorCount)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Summary.P99 == out[j].Summary.P99 {
			return out[i].Name < out[j].Name
		}
		return out[i].Summary.P99 > out[j].Summary.P99
	})
	return out
}

func dependencies(hists map[string]*latency.Histogram, counts map[string]int, totals map[string]time.Duration, types, services map[string]string, totalRequestTime time.Duration) []DependencySummary {
	out := make([]DependencySummary, 0, len(hists))
	for name, hist := range hists {
		var share float64
		if totalRequestTime > 0 {
			// This share can exceed 100% when dependency spans overlap. Keeping it
			// visible is more useful than hiding concurrency in the trace.
			share = float64(totals[name]) / float64(totalRequestTime)
		}
		out = append(out, DependencySummary{
			Name:            name,
			Type:            types[name],
			Service:         services[name],
			Count:           counts[name],
			Total:           totals[name],
			Summary:         latency.Summarize(hist, counts[name], 0),
			ShareOfRequests: share,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total == out[j].Total {
			return out[i].Name < out[j].Name
		}
		return out[i].Total > out[j].Total
	})
	return out
}

func slowTraces(traces []traceData, limit int) []SlowTrace {
	out := make([]SlowTrace, 0, len(traces))
	for _, trace := range traces {
		var depTime time.Duration
		var deps []DependencySample
		for _, span := range trace.spans {
			if span.SpanID == trace.root.SpanID || !isDependency(span) {
				continue
			}
			name, depType := dependencyName(span)
			d := span.Duration()
			depTime += d
			deps = append(deps, DependencySample{Name: name, Type: depType, Duration: d})
		}
		sort.Slice(deps, func(i, j int) bool { return deps[i].Duration > deps[j].Duration })
		if len(deps) > 5 {
			deps = deps[:5]
		}
		out = append(out, SlowTrace{
			TraceID:         trace.id,
			Route:           routeName(trace.root),
			Service:         serviceName(trace.root),
			RootSpan:        trace.root.Name,
			Latency:         trace.root.Duration(),
			Start:           trace.root.Start,
			DependencyTime:  depTime,
			TopDependencies: deps,
			Error:           spanFailed(trace.root),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Latency > out[j].Latency })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func hints(report Report) []string {
	var out []string
	if len(report.Dependencies) > 0 && report.Dependencies[0].ShareOfRequests > 0.5 {
		out = append(out, "One dependency accounts for more than half of request time in this input. Inspect that dependency before assuming application CPU is the bottleneck.")
	}
	if report.Summary.Errors > 0 {
		out = append(out, "Some request spans are marked as errors. Compare their slow traces with successful traces before treating latency as a single distribution.")
	}
	if len(report.Routes) > 1 && report.Routes[0].Summary.P99 > report.Summary.P99 {
		out = append(out, "Route-level p99 differs from the overall p99. Segmenting by route is likely useful for this input.")
	}
	if len(out) == 0 {
		out = append(out, "No span/dependency pattern was strong enough for a built-in hint. Use the route and dependency tables as investigation starting points.")
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func joinName(parts ...string) string {
	var out []string
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, ":")
}

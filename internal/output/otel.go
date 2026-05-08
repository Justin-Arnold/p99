package output

import (
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/justin/p99/internal/latency"
	"github.com/justin/p99/internal/probe"
	"github.com/justin/p99/internal/spans"
)

type otlpExport struct {
	ResourceMetrics []otlpResourceMetrics `json:"resourceMetrics"`
}

type otlpResourceMetrics struct {
	Resource     otlpResource      `json:"resource"`
	ScopeMetrics []otlpScopeMetric `json:"scopeMetrics"`
}

type otlpResource struct {
	Attributes []otlpAttribute `json:"attributes,omitempty"`
}

type otlpScopeMetric struct {
	Scope   otlpScope    `json:"scope"`
	Metrics []otlpMetric `json:"metrics"`
}

type otlpScope struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type otlpMetric struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Unit        string     `json:"unit,omitempty"`
	Gauge       *otlpGauge `json:"gauge,omitempty"`
	Sum         *otlpSum   `json:"sum,omitempty"`
}

type otlpGauge struct {
	DataPoints []otlpPoint `json:"dataPoints"`
}

type otlpSum struct {
	AggregationTemporality string      `json:"aggregationTemporality"`
	IsMonotonic            bool        `json:"isMonotonic"`
	DataPoints             []otlpPoint `json:"dataPoints"`
}

type otlpPoint struct {
	Attributes   []otlpAttribute `json:"attributes,omitempty"`
	TimeUnixNano string          `json:"timeUnixNano"`
	AsDouble     *float64        `json:"asDouble,omitempty"`
	AsInt        *int64          `json:"asInt,omitempty"`
}

type otlpAttribute struct {
	Key   string    `json:"key"`
	Value otlpValue `json:"value"`
}

type otlpValue struct {
	StringValue string `json:"stringValue,omitempty"`
}

func WriteOTel(w io.Writer, result probe.RunResult) error {
	ts := result.EndedAt
	if ts.IsZero() {
		ts = time.Now()
	}
	metrics := []otlpMetric{
		sumMetric("p99.http.requests", "Total HTTP probe requests.", "1", ts, int64(result.Summary.Count), nil),
		sumMetric("p99.http.errors", "Total HTTP probe errors.", "1", ts, int64(result.Summary.Errors), nil),
		gaugeMetric("p99.http.error_rate", "HTTP probe error rate.", "1", ts, result.Summary.ErrorRate, nil),
	}
	metrics = append(metrics, latencyMetrics("p99.http.latency", "HTTP probe latency quantiles.", ts, result.Summary)...)
	for class, count := range result.Errors {
		metrics = append(metrics, sumMetric("p99.http.errors.by_class", "HTTP probe errors by class.", "1", ts, int64(count), attrs("class", class)))
	}
	return writeOTLP(w, "p99.http", metrics)
}

func WriteSpanOTel(w io.Writer, report spans.Report) error {
	ts := time.Now()
	metrics := []otlpMetric{
		sumMetric("p99.spans.requests", "Total request traces in the span input.", "1", ts, int64(report.RequestCount), nil),
		sumMetric("p99.spans.errors", "Request traces marked as errors.", "1", ts, int64(report.Summary.Errors), nil),
		gaugeMetric("p99.spans.error_rate", "Request trace error rate.", "1", ts, report.Summary.ErrorRate, nil),
	}
	metrics = append(metrics, latencyMetrics("p99.spans.request.latency", "Request trace latency quantiles.", ts, report.Summary)...)
	for _, route := range report.Routes {
		metrics = append(metrics, labeledLatencyMetrics("p99.spans.route.latency", "Route latency quantiles.", ts, route.Summary, attrs("route", route.Name))...)
	}
	for _, dep := range report.Dependencies {
		depAttrs := attrs("dependency", dep.Name, "type", dep.Type)
		metrics = append(metrics, labeledLatencyMetrics("p99.spans.dependency.latency", "Dependency span latency quantiles.", ts, dep.Summary, depAttrs)...)
		metrics = append(metrics, gaugeMetric("p99.spans.dependency.time", "Total represented dependency time.", "s", ts, dep.Total.Seconds(), depAttrs))
	}
	return writeOTLP(w, "p99.spans", metrics)
}

func writeOTLP(w io.Writer, scope string, metrics []otlpMetric) error {
	doc := otlpExport{ResourceMetrics: []otlpResourceMetrics{{
		Resource: otlpResource{Attributes: attrs("service.name", "p99")},
		ScopeMetrics: []otlpScopeMetric{{
			Scope:   otlpScope{Name: scope},
			Metrics: metrics,
		}},
	}}}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

func latencyMetrics(name, desc string, ts time.Time, s latency.Summary) []otlpMetric {
	return labeledLatencyMetrics(name, desc, ts, s, nil)
}

func labeledLatencyMetrics(name, desc string, ts time.Time, s latency.Summary, base []otlpAttribute) []otlpMetric {
	return []otlpMetric{
		gaugeMetric(name, desc, "s", ts, s.P50.Seconds(), appendAttrs(base, "quantile", "0.5")),
		gaugeMetric(name, desc, "s", ts, s.P95.Seconds(), appendAttrs(base, "quantile", "0.95")),
		gaugeMetric(name, desc, "s", ts, s.P99.Seconds(), appendAttrs(base, "quantile", "0.99")),
		gaugeMetric(name, desc, "s", ts, s.P999.Seconds(), appendAttrs(base, "quantile", "0.999")),
		gaugeMetric(name, desc, "s", ts, s.Max.Seconds(), appendAttrs(base, "quantile", "1")),
	}
}

func gaugeMetric(name, desc, unit string, ts time.Time, value float64, attributes []otlpAttribute) otlpMetric {
	return otlpMetric{Name: name, Description: desc, Unit: unit, Gauge: &otlpGauge{DataPoints: []otlpPoint{doublePoint(ts, value, attributes)}}}
}

func sumMetric(name, desc, unit string, ts time.Time, value int64, attributes []otlpAttribute) otlpMetric {
	return otlpMetric{Name: name, Description: desc, Unit: unit, Sum: &otlpSum{AggregationTemporality: "AGGREGATION_TEMPORALITY_CUMULATIVE", IsMonotonic: true, DataPoints: []otlpPoint{intPoint(ts, value, attributes)}}}
}

func doublePoint(ts time.Time, value float64, attributes []otlpAttribute) otlpPoint {
	return otlpPoint{TimeUnixNano: unixNanoString(ts), AsDouble: &value, Attributes: attributes}
}

func intPoint(ts time.Time, value int64, attributes []otlpAttribute) otlpPoint {
	return otlpPoint{TimeUnixNano: unixNanoString(ts), AsInt: &value, Attributes: attributes}
}

func unixNanoString(ts time.Time) string {
	return strconv.FormatInt(ts.UnixNano(), 10)
}

func attrs(kv ...string) []otlpAttribute {
	var out []otlpAttribute
	for i := 0; i+1 < len(kv); i += 2 {
		out = append(out, otlpAttribute{Key: kv[i], Value: otlpValue{StringValue: kv[i+1]}})
	}
	return out
}

func appendAttrs(base []otlpAttribute, kv ...string) []otlpAttribute {
	out := append([]otlpAttribute(nil), base...)
	return append(out, attrs(kv...)...)
}

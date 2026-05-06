package spans

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

func ParseJSON(r io.Reader) ([]Span, []string, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return nil, nil, err
	}

	var warnings []string
	spans, err := parseAny(root, "", &warnings)
	if err != nil {
		return nil, warnings, err
	}
	if len(spans) == 0 {
		return nil, warnings, fmt.Errorf("no spans found")
	}
	return spans, warnings, nil
}

func parseAny(v any, inheritedService string, warnings *[]string) ([]Span, error) {
	switch x := v.(type) {
	case []any:
		var out []Span
		for _, item := range x {
			spans, err := parseAny(item, inheritedService, warnings)
			if err != nil {
				return nil, err
			}
			out = append(out, spans...)
		}
		return out, nil
	case map[string]any:
		if resourceSpans, ok := array(x["resourceSpans"]); ok {
			var out []Span
			for _, resourceSpan := range resourceSpans {
				rs, ok := object(resourceSpan)
				if !ok {
					continue
				}
				service := resourceServiceName(rs, inheritedService)
				if scopeSpans, ok := array(rs["scopeSpans"]); ok {
					for _, scopeSpan := range scopeSpans {
						ss, ok := object(scopeSpan)
						if !ok {
							continue
						}
						spans, err := parseAny(ss["spans"], service, warnings)
						if err != nil {
							return nil, err
						}
						out = append(out, spans...)
					}
				}
				if instrumentationLibrarySpans, ok := array(rs["instrumentationLibrarySpans"]); ok {
					for _, ils := range instrumentationLibrarySpans {
						m, ok := object(ils)
						if !ok {
							continue
						}
						spans, err := parseAny(m["spans"], service, warnings)
						if err != nil {
							return nil, err
						}
						out = append(out, spans...)
					}
				}
			}
			return out, nil
		}
		if spans, ok := array(x["spans"]); ok {
			return parseAny(spans, inheritedService, warnings)
		}
		if looksLikeSpan(x) {
			span, err := parseSpan(x, inheritedService)
			if err != nil {
				*warnings = append(*warnings, err.Error())
				return nil, nil
			}
			return []Span{span}, nil
		}
	}
	return nil, nil
}

func parseSpan(m map[string]any, inheritedService string) (Span, error) {
	start, err := parseSpanTime(first(m, "startTimeUnixNano", "start_time_unix_nano", "startTime", "start_time"))
	if err != nil {
		return Span{}, fmt.Errorf("span %q has invalid start time: %w", stringValue(m["spanId"]), err)
	}
	end, err := parseSpanTime(first(m, "endTimeUnixNano", "end_time_unix_nano", "endTime", "end_time"))
	if err != nil {
		return Span{}, fmt.Errorf("span %q has invalid end time: %w", stringValue(m["spanId"]), err)
	}

	attrs := parseAttributes(m["attributes"])
	service := inheritedService
	if v := attrs["service.name"]; v != "" {
		service = v
	}
	if v := stringValue(m["service"]); v != "" {
		service = v
	}
	if v := stringValue(m["serviceName"]); v != "" {
		service = v
	}

	statusCode, statusText := parseStatus(m["status"])
	return Span{
		TraceID:      stringValue(first(m, "traceId", "trace_id")),
		SpanID:       stringValue(first(m, "spanId", "span_id")),
		ParentSpanID: stringValue(first(m, "parentSpanId", "parent_span_id")),
		Name:         stringValue(m["name"]),
		Kind:         normalizeKind(stringValue(m["kind"])),
		Service:      service,
		Start:        start,
		End:          end,
		Attributes:   attrs,
		StatusCode:   statusCode,
		StatusText:   statusText,
	}, nil
}

func looksLikeSpan(m map[string]any) bool {
	return first(m, "traceId", "trace_id") != nil && first(m, "spanId", "span_id") != nil && m["name"] != nil
}

func resourceServiceName(m map[string]any, fallback string) string {
	if resource, ok := object(m["resource"]); ok {
		attrs := parseAttributes(resource["attributes"])
		if service := attrs["service.name"]; service != "" {
			return service
		}
	}
	return fallback
}

func parseAttributes(v any) map[string]string {
	out := map[string]string{}
	switch x := v.(type) {
	case []any:
		for _, item := range x {
			m, ok := object(item)
			if !ok {
				continue
			}
			key := stringValue(m["key"])
			if key == "" {
				continue
			}
			out[key] = attributeValue(m["value"])
		}
	case map[string]any:
		for k, v := range x {
			out[k] = attributeValue(v)
		}
	}
	return out
}

func attributeValue(v any) string {
	if m, ok := object(v); ok {
		for _, key := range []string{"stringValue", "intValue", "doubleValue", "boolValue", "bytesValue"} {
			if raw, ok := m[key]; ok {
				return stringValue(raw)
			}
		}
		if arr, ok := object(m["arrayValue"]); ok {
			values, _ := array(arr["values"])
			parts := make([]string, 0, len(values))
			for _, value := range values {
				parts = append(parts, attributeValue(value))
			}
			return strings.Join(parts, ",")
		}
	}
	return stringValue(v)
}

func parseStatus(v any) (string, string) {
	m, ok := object(v)
	if !ok {
		return "", ""
	}
	return strings.ToUpper(stringValue(m["code"])), stringValue(m["message"])
}

func parseSpanTime(v any) (time.Time, error) {
	switch x := v.(type) {
	case json.Number:
		n, err := strconv.ParseInt(string(x), 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(0, n).UTC(), nil
	case float64:
		return time.Unix(0, int64(x)).UTC(), nil
	case string:
		if x == "" {
			return time.Time{}, fmt.Errorf("empty time")
		}
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return time.Unix(0, n).UTC(), nil
		}
		t, err := time.Parse(time.RFC3339Nano, x)
		if err != nil {
			return time.Time{}, err
		}
		return t, nil
	default:
		return time.Time{}, fmt.Errorf("missing time")
	}
}

func normalizeKind(kind string) string {
	kind = strings.TrimPrefix(strings.ToUpper(kind), "SPAN_KIND_")
	switch kind {
	case "1":
		return "INTERNAL"
	case "2":
		return "SERVER"
	case "3":
		return "CLIENT"
	case "4":
		return "PRODUCER"
	case "5":
		return "CONSUMER"
	}
	if kind == "" || kind == "UNSPECIFIED" {
		return ""
	}
	return kind
}

func first(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return nil
}

func stringValue(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return string(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprint(x)
	}
}

func object(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func array(v any) ([]any, bool) {
	a, ok := v.([]any)
	return a, ok
}

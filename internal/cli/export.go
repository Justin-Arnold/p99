package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/justin/p99/internal/output"
	"github.com/justin/p99/internal/probe"
	"github.com/justin/p99/internal/spans"
)

func writeRunFormat(w io.Writer, format string, result probe.RunResult) error {
	return writeReportFormat(w, format, result, output.ReportOptions{})
}

func writeReportFormat(w io.Writer, format string, result probe.RunResult, opts output.ReportOptions) error {
	switch format {
	case "text":
		if opts.Details || opts.Histogram || opts.SlowSamples || opts.Shape || opts.Runtime {
			output.WriteReport(w, result, opts)
		} else {
			output.WriteSummary(w, result)
		}
	case "json":
		return output.WriteJSON(w, result)
	case "markdown", "md":
		output.WriteMarkdown(w, result)
	case "prometheus", "prom":
		output.WritePrometheus(w, result)
	case "otel", "otlp":
		return output.WriteOTel(w, result)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
	return nil
}

func writeSpanFormat(w io.Writer, format string, report spans.Report) error {
	switch format {
	case "text":
		output.WriteSpanReport(w, report)
	case "json":
		return writeSpanJSON(w, report)
	case "markdown", "md":
		output.WriteSpanMarkdown(w, report)
	case "prometheus", "prom":
		output.WriteSpanPrometheus(w, report)
	case "otel", "otlp":
		return output.WriteSpanOTel(w, report)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
	return nil
}

func writeFile(path string, write func(io.Writer) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := write(f); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

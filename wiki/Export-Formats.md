# Export Formats

`p99` supports several output formats.

## JSON

JSON is the canonical saved format.

Use JSON when you want:

- full fidelity
- slow samples
- histograms
- runtime snapshots
- future comparison
- machine-readable analysis

## Markdown

Markdown is for humans.

Use it when you want:

- pull request artifacts
- incident notes
- a report to paste into documentation
- readable summaries for teammates

## Prometheus Text

Prometheus text exposition is for metric-shaped artifacts.

Use it when:

- another tool expects Prometheus text
- you want to archive metrics from a run
- CI wants to publish simple numeric values

Prometheus output is not a full replacement for JSON. It preserves metric views, not every run detail.

## OpenTelemetry-Compatible JSON

OTel output is a metrics-oriented JSON structure following OpenTelemetry-style naming and data point shapes.

Use it when:

- an integration wants OTLP-style metrics
- you want p99 run data in an observability pipeline
- you need a portable metrics document

It is not the same as the original span input format.

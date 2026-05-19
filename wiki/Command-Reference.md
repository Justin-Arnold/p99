# Command Reference

`p99` is organized around subcommands. Each command solves a different part of the latency investigation workflow.

## Commands

| Command | Purpose |
| --- | --- |
| `p99 http` | Probe an HTTP endpoint and produce latency/error summaries. |
| `p99 watch` | Continuously run short probe windows and redraw a live summary. |
| `p99 profile` | Capture a Go pprof profile and optionally correlate it with a probe/runtime window. |
| `p99 runtime` | Collect Go runtime signal movement over a time window. |
| `p99 spans` | Analyze OpenTelemetry-style span JSON for route and dependency timing. |
| `p99 report` | Read a saved run JSON file and print or export a report. |
| `p99 compare` | Compare two saved run JSON files and enforce budgets. |

## Choosing a Command

Use `http` when you need a synthetic probe.

Use `watch` when you want repeated feedback during an active change.

Use `report` when you already have a saved run.

Use `compare` when you need a before/after answer or CI gate.

Use `profile` when a Go service may be spending CPU in a slow path.

Use `runtime` when you want to check GC, goroutines, heap, blocking, or pool-wait signals.

Use `spans` when trace data is available and you want route/dependency breakdowns.

Each command has a dedicated page with examples and flag explanations.

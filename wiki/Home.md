# p99

`p99` is a command-line latency profiler focused on tail latency.

It is built around one practical question:

> Where is time actually going, and how bad is the tail?

Many performance tools make it easy to see averages. Averages are often a poor fit for user experience. If 99 requests finish in 30ms and one request takes 3s, the average can look acceptable while a real user had a bad experience. `p99` keeps the tail visible.

## What p99 Does

`p99` can:

- probe HTTP endpoints under controlled load
- report p50, p90, p95, p99, p999, max latency, and error rate
- retain bounded samples of the slowest requests
- save JSON run files
- compare two saved runs
- enforce latency and error budgets in CI
- watch an endpoint in repeated live windows
- capture Go pprof profiles
- correlate probe windows with Go runtime signals
- analyze OpenTelemetry-style span JSON and dependency timing
- export reports as JSON, Markdown, Prometheus text, or OpenTelemetry-compatible JSON metrics

## What p99 Is Not

`p99` does not prove root cause by itself. It is designed to narrow the search space.

If p99 gets worse while GC pause time also increases, that is a useful correlation. It is not proof that GC caused the tail. If dependency spans account for most request time, that is a strong lead. It is not proof that the dependency is the only problem.

`p99` tries to say what it can support and avoid overclaiming.

## Who It Is For

`p99` is useful if you:

- operate HTTP services
- need CI latency budgets
- want a local load probe that saves comparable run files
- need to explain tail latency in concrete terms
- work on Go services and want pprof/runtime signals next to latency results
- have span data and want route/dependency timing summaries

You do not need to be an expert in latency profiling to use it. The wiki explains the concepts as they appear.

## Where To Go Next

New to the topic:

1. Read [Learning p99](Learning-p99).
2. Use [Concepts](Concepts) as a glossary while you work.
3. Run the examples in [Quick Start](Quick-Start).
4. Read [Command http](Command-http) and [Command report](Command-report).

Already familiar with profiling:

1. Start with [Command Reference](Command-Reference).
2. Use [Command compare](Command-compare) for CI budgets.
3. Use [Command profile](Command-profile), [Command runtime](Command-runtime), and [Command spans](Command-spans) for deeper investigation.

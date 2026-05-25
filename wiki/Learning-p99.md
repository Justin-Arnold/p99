# Learning p99

This guide teaches `p99` and the profiling ideas behind it at the same time.

You do not need to know what p99, pprof, runtime signals, histograms, or spans are before starting. The goal is to build a mental model you can use when an endpoint feels slow and you need to investigate without guessing.

## The Big Idea

Performance work starts with one deceptively simple question:

> What did users actually experience?

That question is harder than it sounds. A service can feel fast most of the time and still hurt real users. Averages can look healthy while a small group of requests are painfully slow. A dependency can be fine for 95 percent of calls and occasionally stall badly enough to dominate the user experience.

`p99` is built for that uncomfortable part of the distribution: the tail.

It helps you move through a practical investigation:

1. Measure what users would experience.
2. Look at the distribution, not just one number.
3. Save the evidence.
4. Compare before and after.
5. Correlate latency with runtime, pprof, or span signals.
6. Turn the result into a repeatable budget.

## What Profiling Means

Profiling means collecting measurements that help explain where time, CPU, memory, or waiting is going.

There are different kinds of profiling:

- latency profiling asks how long requests take
- CPU profiling asks where code spends CPU time
- memory profiling asks where allocations come from
- runtime profiling asks whether GC, goroutines, locks, scheduler pauses, or pools moved during a window
- trace or span profiling asks which route, service, or dependency contributed time

`p99` starts with latency because latency is closest to what users notice. It can then bring in other signals when you need them.

## Step 1: Measure One Endpoint

Start with the smallest useful measurement:

```sh
p99 http https://api.example.com/search
```

This sends requests for the default duration and prints a summary.

You will see values like:

```text
Requests: 10 (10 success, 0 errors)
Error rate: 0.000%
Latency: min 18.000ms  p50 22.000ms  p90 35.000ms  p95 41.000ms  p99 75.000ms  p999 75.000ms  max 75.000ms
Shape: insufficient_signal
```

Read this from left to right:

- `count` tells you how much evidence you have
- `success` and `errors` tell you whether the probe mostly worked
- `p50` is the middle request
- `p95`, `p99`, and `p999` show the tail
- `max` is the slowest request observed
- `shape` is a conservative description of the distribution

If the run is short or has few requests, `p99` may say `insufficient_signal`. That is not a failure. It means the tool is refusing to over-explain weak evidence.

## Step 2: Understand Percentiles

A percentile answers:

> What latency were N percent of requests at or below?

If `p99` is `400ms`, then 99 percent of observed requests completed in `400ms` or less. The remaining 1 percent were slower.

That small 1 percent matters. In a high-traffic service, 1 percent can still be many users. In a page with many backend calls, one slow call can hold up the entire page.

Useful landmarks:

- `p50` shows the typical request
- `p95` shows the beginning of the uncomfortable tail
- `p99` shows rare but important pain
- `p999` is for very high volume or very strict systems
- `max` is the worst thing you saw, but it may be one odd event

Do not treat p99 as an answer by itself. It is a lens. Use it with request count, error rate, slow samples, and comparisons.

## Step 3: Add Controlled Load

A single request at a time does not tell you how a service behaves under pressure. Add a request rate and concurrency:

```sh
p99 http \
  --rps 100 \
  --duration 2m \
  --concurrency 20 \
  https://api.example.com/search
```

The important flags are:

- `--rps` sets the target request rate
- `--duration` controls how long measurements run
- `--concurrency` limits in-flight requests
- `--timeout` defines when a request has taken too long

This is not a full replacement for every load-testing tool. It is a focused way to ask: under this level of pressure, what happens to the tail?

## Step 4: Watch Error Rate

Latency without error rate can lie.

A broken service can look fast because it fails immediately. A service under overload can return quick 500s while successful requests disappear. Always read latency and errors together.

In `p99`, errors include:

- timeouts
- DNS failures
- connection refused
- TLS failures
- HTTP 4xx responses
- HTTP 5xx responses
- response body read errors
- unknown transport errors

If error rate moves, investigate it before celebrating a lower p99.

## Step 5: Save the Evidence

Terminal output is useful for humans, but saved runs are useful for investigation.

```sh
p99 http \
  --duration 30s \
  --rps 50 \
  --output before.json \
  https://api.example.com/search
```

The JSON file contains:

- probe configuration
- timestamps
- summary stats
- histogram data
- retained slow samples
- error breakdown
- shape analysis
- runtime correlation if collected

Saving the run matters because performance work is comparative. You want to know whether something changed, not just whether one run looked good or bad.

## Step 6: Read a Report

Use `report` when you want to inspect a saved run:

```sh
p99 report before.json
```

Ask for deeper detail when the summary raises questions:

```sh
p99 report --details before.json
```

The detailed report shows histogram buckets, slow samples, and shape explanation.

Slow samples are concrete examples of the worst observed requests. They help answer:

- when did the slow request happen?
- what path was involved?
- did it return an error status?
- was it a timeout, 5xx, body read problem, or something else?

Slow samples do not prove root cause, but they give you handles for logs, traces, and service-specific debugging.

## Step 7: Learn the Shape

`p99` tries to describe the distribution conservatively.

Common shapes:

- `stable`: the tail is not dramatically worse than the median
- `spiky`: most requests are fine, but a small number are much slower
- `bimodal`: requests appear to cluster into two bands
- `degrading_over_time`: later requests are slower than earlier requests
- `insufficient_signal`: there is not enough evidence for a stronger claim

Examples of what shape can suggest:

- spiky can happen with retries, lock contention, dependency stalls, or noisy neighbors
- bimodal can happen when cache hits and misses are mixed together
- degrading over time can happen with queue buildup, resource exhaustion, GC pressure, or downstream slowdown

These are hints, not verdicts. The right next step is to collect another signal.

## Step 8: Compare Before and After

After a code change, config change, deploy, or dependency change, run another probe:

```sh
p99 http \
  --duration 30s \
  --rps 50 \
  --output after.json \
  https://api.example.com/search
```

Compare the two runs:

```sh
p99 compare before.json after.json
```

This shows movement in:

- p50
- p95
- p99
- p999
- max
- error rate
- request count

This is the core habit: measure before, change one thing, measure after, compare.

## Step 9: Turn Learning Into Budgets

Once you know what acceptable looks like, encode it.

For one run:

```sh
p99 http \
  --duration 30s \
  --rps 50 \
  --p99-under 250ms \
  --error-rate-under 0.5 \
  https://api.example.com/search
```

For before/after comparison:

```sh
p99 compare before.json after.json \
  --max-p99-regression 20% \
  --error-rate-under 0.5 \
  --min-request-count 1000
```

Budgets are useful in CI because they make performance changes visible. They should be realistic. A budget that fails randomly will teach people to ignore it.

## Step 10: Add Go Runtime Signals

If the target is a Go service exposing debug endpoints, correlate the probe with runtime movement:

```sh
p99 http \
  --duration 30s \
  --rps 50 \
  --runtime http://localhost:8080 \
  https://api.example.com/search
```

Runtime signals can include:

- goroutine count
- heap allocation
- GC cycles
- GC pause time
- mutex profile samples
- block profile samples
- scheduler pause metrics
- DB pool wait counters if exposed

This answers a different question:

> Did Go runtime signals move during the same window as tail latency?

Correlation is not causation. If p99 worsens and GC pause time rises, that is a lead. It is not proof. Use it to decide what to inspect next.

## Step 11: Capture a pprof Profile

Latency tells you what users experienced. CPU profiling can help explain where Go code spent CPU during a window.

```sh
p99 profile \
  --seconds 30 \
  --output cpu.pprof \
  http://localhost:8080/debug/pprof/profile
```

You can also capture a profile while probing an endpoint:

```sh
p99 profile \
  --seconds 30 \
  --probe https://api.example.com/search \
  --probe-rps 50 \
  --probe-concurrency 10 \
  http://localhost:8080/debug/pprof/profile
```

Use this when runtime or latency signals suggest the service itself may be doing expensive work. If spans show most time is in a downstream dependency, pprof may be less useful as the first next step.

## Step 12: Use Spans for Route and Dependency Timing

When trace data is available, `p99 spans` changes the question again:

> Which routes, services, or dependencies account for the latency?

```sh
p99 spans traces.json
```

Span analysis can show:

- request latency from root/server spans
- route-level summaries
- service-level summaries
- dependency summaries
- slow traces
- hints about dependency-heavy requests

Dependency time from spans can overlap. Treat it as represented dependency time, not always exclusive wall-clock time.

## A Practical Investigation Loop

Use this loop when something feels slow:

1. Run `p99 http` to get a first look.
2. Increase duration and RPS until the run has enough evidence.
3. Save the run with `--output`.
4. Read it with `p99 report --details`.
5. Look at p99, p999, max, error rate, shape, and slow samples.
6. Compare against a known-good run if you have one.
7. Add runtime correlation if the service is Go and debug endpoints are available.
8. Capture pprof if CPU or runtime signals look relevant.
9. Analyze spans if trace data is available.
10. Turn the result into budgets once the signal is stable.

## How to Avoid Misreading Results

Common traps:

- treating one short run as truth
- looking at p99 without request count
- ignoring errors because latency looks good
- comparing runs with different RPS, duration, or concurrency
- assuming correlation proves cause
- mixing routes together when one route dominates the tail
- using a budget that is too tight for normal variance

Good investigations are boring in the best way: repeatable inputs, saved evidence, clear comparisons, and modest conclusions.

## What to Read Next

Read these pages in order if you are learning from the beginning:

1. [Concepts](Concepts)
2. [Quick Start](Quick-Start)
3. [Command http](Command-http)
4. [Run Files and Reports](Run-Files-and-Reports)
5. [Command compare](Command-compare)
6. [Command runtime](Command-runtime)
7. [Command profile](Command-profile)
8. [Command spans](Command-spans)

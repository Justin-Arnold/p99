# Concepts

This page explains the terms `p99` uses and why they matter.

## Latency

Latency is how long one operation takes. For `p99 http`, the operation is one HTTP request. For `p99 spans`, request latency comes from a server span or root span in a trace.

Latency is usually shown as a distribution because one number cannot describe the whole experience.

## Percentiles

A percentile answers: "What latency are N percent of requests at or below?"

- p50: half of requests are at or below this latency
- p90: 90 percent are at or below this latency
- p95: 95 percent are at or below this latency
- p99: 99 percent are at or below this latency
- p999: 99.9 percent are at or below this latency
- max: the slowest observed request

If p99 is 400ms, then 99 percent of measured requests completed in 400ms or less, and 1 percent took longer.

## Tail Latency

Tail latency is the slow end of the distribution. It is where rare but painful requests live.

Tail latency matters because users and systems often experience the slowest path, not the average path:

- a page may wait for the slowest API call
- a batch may wait for the slowest shard
- a retry may hide failures while making the tail worse
- a dependency may be fast most of the time but occasionally stall

## Why Averages Can Mislead

Average latency compresses all requests into one number. It can hide a small number of very slow requests.

Example:

- 99 requests at 20ms
- 1 request at 2000ms

The average is about 40ms. That sounds fine. But one user waited 2 seconds. p99 and max make that visible.

## Error Rate

Error rate is the fraction of requests that failed. In `p99`, failure can mean a transport error, timeout, TLS error, unexpected HTTP status, body read failure, or another classified error.

Latency without error rate can be misleading. A service can look fast because it is failing quickly.

## Slow Samples

Slow samples are bounded records of the slowest observed requests. They include latency, timestamp, method, URL/path, status code, and error class where available.

They solve a simple problem: after seeing a bad p99, you need concrete examples to inspect.

## Histograms

A histogram groups latencies into buckets. It helps show the shape of the distribution.

`p99` keeps exact observations for small runs. Longer or high-throughput runs compact into significant-figure buckets so memory usage stays bounded.

This means compacted percentile values are bucket-resolution estimates. That is usually the right tradeoff for long probes and CI.

## Shape Analysis

Shape analysis is a conservative description of the latency distribution.

Current shapes include:

- stable
- spiky
- bimodal
- degrading over time
- insufficient signal

Shape hints are not conclusions. They are prompts for the next investigation step.

## Runtime Correlation

Runtime correlation means collecting Go runtime signals before and after a measured window and comparing movement.

Examples:

- goroutine count rose
- heap allocation increased
- GC pause time increased
- mutex/block profile samples appeared
- DB pool wait counters moved

These signals can line up with tail pain, but they do not prove causality.

## Span and Dependency Timing

Span analysis uses trace data to break latency down by route, service, and dependency spans.

This is useful when the question changes from "is the endpoint slow?" to "which route or dependency path is slow?"

Dependency time from spans may overlap. Read it as "time represented by dependency spans," not exclusive wall-clock time.

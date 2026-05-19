# Command: p99 compare

`p99 compare` compares two saved HTTP run JSON files.

## Problem It Solves

Performance work is usually about change: did this deploy, patch, branch, or configuration change make the tail better or worse? `compare` makes regressions visible and can fail CI when budgets are violated.

## Basic Use

```sh
p99 compare before.json after.json
```

## CI Budget Example

```sh
p99 compare before.json after.json \
  --max-p95-regression 10% \
  --max-p99-regression 20% \
  --error-rate-under 0.5
```

## Regression Flags

Regression flags compare the percent change from `before` to `after`.

### `--max-p50-regression`

Fail if p50 regresses by more than this percent.

Use it to catch broad latency movement affecting typical requests.

### `--max-p95-regression`

Fail if p95 regresses by more than this percent.

Use it when you care about the upper end of normal user experience.

### `--max-p99-regression`

Fail if p99 regresses by more than this percent.

Use it as the main tail-latency budget.

### `--max-p999-regression`

Fail if p999 regresses by more than this percent.

Use it for very tail-sensitive systems. It needs enough request volume to be meaningful.

### `--max-max-regression`

Fail if max latency regresses by more than this percent.

Use it to catch worst-case changes. Treat max carefully because it is sensitive to outliers.

### `--max-error-rate-regression`

Fail if error rate regresses by more than this percent.

Use it when relative error movement matters.

## Absolute Budget Flags

Absolute flags check the `after` run directly.

### `--p50-under`

Fail if after p50 exceeds a duration.

### `--p95-under`

Fail if after p95 exceeds a duration.

### `--p99-under`

Fail if after p99 exceeds a duration.

### `--p999-under`

Fail if after p999 exceeds a duration.

### `--max-under`

Fail if after max latency exceeds a duration.

### `--error-rate-under` and `--max-error-rate`

Fail if after error rate exceeds a percent.

`0.5` means 0.5 percent.

## Request Volume Flags

### `--min-request-count`

Fail if the after run has fewer requests than this count.

Use it to avoid trusting a comparison based on too little data.

### `--max-request-drop`

Fail if request count drops by more than this percent.

Use it when reduced throughput would make a latency comparison misleading.

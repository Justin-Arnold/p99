# Command: p99 watch

`p99 watch` repeatedly runs short HTTP probe windows and redraws a terminal summary.

## Problem It Solves

During a deploy, configuration change, or local performance experiment, you often need continuous feedback rather than one saved run. `watch` shows whether the tail is improving, degrading, or becoming unstable across windows.

## Basic Use

```sh
p99 watch --window 5s https://api.example.com/search
```

## Watch a Request Mix

```sh
p99 watch --request-spec requests.yaml --seed 123 --window 5s
```

Use this when you want the live dashboard to exercise the same weighted search, detail, or write requests that users produce.

## Flags

### `--window` and `--duration`

Length of each watch refresh window.

Example:

```sh
p99 watch --window 10s https://api.example.com/search
```

Longer windows give more stable percentiles. Shorter windows react faster.

### `--rps`

Target requests per second for each window.

Use it to keep pressure consistent while watching.

### `--concurrency`

Maximum in-flight requests.

Use it to bound probe pressure when the target slows down.

### `--timeout`

Per-request timeout.

Use it to match the user or service budget you care about.

### `--method`

HTTP method for the watched request.

Use it for non-GET endpoints.

### `--header` and `-H`

Request headers. Can be repeated.

Use it for auth, routing, content type, or tenant selection.

### `--body-file`

Request body file.

Use it for POST/PUT/PATCH watch probes.

### `--status`

Expected status code. Can be repeated or comma-separated.

Use it when the watched endpoint has a specific success code.

### `--slow-samples`

Number of slow request samples retained per window.

Use it when you want concrete slow examples while watching.

### `--iterations`

Number of windows to run. `0` means run until interrupted.

Example:

```sh
p99 watch --iterations 3 --window 5s https://api.example.com/search
```

Use it for scripted checks or tests.

### `--clear`

Whether to clear the terminal between refreshes.

Example:

```sh
p99 watch --clear=false --iterations 3 https://api.example.com/search
```

Use `--clear=false` when capturing output in logs.

### `--runtime`

Collect Go runtime signal movement for each watch window.

Use it when watching a Go service and you want latency next to runtime changes.

### `--runtime-timeout`

Timeout for runtime endpoint requests.

Use it to keep runtime collection from dominating each watch window.

### `--request-spec`

Read a YAML or JSON request spec and use it for each watch window.

Use it when a live check needs more than one static request. With a spec, the URL, method, headers, body, and expected status codes come from the spec file.

### `--base-url`

Override `base_url` from the request spec.

Use it to run the same request mix against a different environment.

### `--seed`

Seed request selection and randomized variables.

Use it to make watch windows repeat the same generated sequence. Omit it when exact replay does not matter.

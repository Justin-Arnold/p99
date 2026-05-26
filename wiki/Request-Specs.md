# Request Specs

Request specs describe HTTP traffic that is too complex for a single command line.

Use them when you need:

- a larger POST, PUT, or PATCH body
- many query parameters
- several request shapes in one run
- weighted traffic, such as mostly search with some detail lookups
- randomized search terms, filters, tenants, or IDs
- repeatable random traffic with `--seed`

Specs can be YAML or JSON. YAML is easier to read for humans; JSON is useful when another tool generates the spec.

## Basic Shape

```yaml
version: 1
base_url: https://api.example.com

defaults:
  headers:
    Authorization: "Bearer {{ env.API_TOKEN }}"
    Content-Type: application/json
  expect_status: [200]

datasets:
  search_terms:
    - latency profiler
    - tail latency
    - pprof

requests:
  - name: user-detail
    weight: 1
    method: GET
    path: /users/123

  - name: search
    weight: 4
    method: POST
    path: /search
    query:
      q: "{{ term }}"
      filter: "{{ filter }}"
    vars:
      term:
        dataset: search_terms
      filter:
        choices: [recent, popular, archived]
    body_file: fixtures/search-body.json.tpl
```

Run it with:

```sh
p99 http --request-spec requests.yaml --seed 123 --duration 2m --rps 100
```

## Why Weights Matter

Weights describe the traffic mix. A request with `weight: 4` is selected about four times as often as a request with `weight: 1`.

This is not a guarantee for very short runs. It is a random selection process, so longer runs get closer to the configured mix.

## Random Values

Use `datasets` for reusable values:

```yaml
datasets:
  terms: [alpha, beta, gamma]

requests:
  - name: search
    method: GET
    path: /search
    query:
      q: "{{ term }}"
    vars:
      term:
        dataset: terms
```

Use inline `choices` for values that only matter to one request:

```yaml
vars:
  filter:
    choices: [recent, popular, archived]
```

`p99` chooses a value each time the request is rendered.

## Template Values

Template placeholders use `{{ name }}`. They can appear in:

- `url`
- `path`
- query values
- headers
- inline `body`
- `body_file` contents

Environment variables use `{{ env.NAME }}`:

```yaml
headers:
  Authorization: "Bearer {{ env.API_TOKEN }}"
```

Use environment placeholders for secrets so token values do not need to live in the spec file.

## Bodies

Use inline bodies for small requests:

```yaml
body: '{"q":"{{ term }}"}'
```

Use `body_file` for larger bodies:

```yaml
body_file: fixtures/search.json.tpl
```

Body files are resolved relative to the spec file. They can contain the same `{{ name }}` and `{{ env.NAME }}` placeholders as the spec.

## Base URLs

Use `base_url` when requests have relative paths:

```yaml
base_url: https://api.example.com
requests:
  - name: detail
    method: GET
    path: /users/123
```

Override it from the CLI:

```sh
p99 http --request-spec requests.yaml --base-url https://staging.example.com
```

This lets one reviewed spec run against different environments.

## Seeded Replay

Use `--seed` to replay the same random sequence:

```sh
p99 http --request-spec requests.yaml --seed 123
```

If `--seed` is omitted or set to `0`, `p99` generates a seed and stores it in the run JSON. Use that saved seed when you want to reproduce a surprising result.

## Watch and Profile

Request specs also work with live watch mode:

```sh
p99 watch --request-spec requests.yaml --seed 123 --window 5s
```

For profile correlation, use `--probe-spec`:

```sh
p99 profile \
  --seconds 30 \
  --probe-spec requests.yaml \
  --probe-rps 100 \
  --seed 123 \
  http://localhost:8080/debug/pprof/profile
```

## Validation

`p99` validates specs before probing.

It rejects:

- unsupported spec versions
- missing requests
- duplicate request names
- missing methods
- invalid methods
- missing `url` or `path`
- both `url` and `path` on the same request
- both `body` and `body_file` on the same request
- missing body files
- invalid status codes
- missing datasets
- empty choices
- unknown template placeholders

Validation happens before any request is sent so broken specs fail fast.

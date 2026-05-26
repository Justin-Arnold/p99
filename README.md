# p99

`p99` is a latency profiler focused on tail latency.

It helps answer:

> Where is time actually going, and how bad is the tail?

`p99` can probe HTTP endpoints, report latency percentiles, retain slow request samples, compare saved runs, inspect Go runtime signals, capture pprof profiles, and analyze OpenTelemetry-style span JSON.

The main signals are percentiles and distributions, not averages. The CLI reports p50, p90, p95, p99, p999, max latency, error rate, and error classes.

## Install

Install the latest release binary:

```sh
curl -fsSL https://raw.githubusercontent.com/Justin-Arnold/p99/main/scripts/install.sh | sh
```

With Go:

```sh
go install github.com/Justin-Arnold/p99/cmd/p99@latest
```

From a local checkout:

```sh
go build ./cmd/p99
```

With Nix:

```sh
nix run github:Justin-Arnold/p99 -- help
nix build github:Justin-Arnold/p99
```

As a flake input on NixOS or nix-darwin:

```nix
{
  imports = [ inputs.p99.nixosModules.default ];
  programs.p99 = {
    enable = true;
    enableCompletions = true;
  };
}
```

For nix-darwin, use `inputs.p99.darwinModules.default`.

## Basic Usage

Probe an endpoint:

```sh
p99 http https://api.example.com/users/123
```

Probe with load and save a run file:

```sh
p99 http --rps 100 --duration 2m --concurrency 20 --output run.json https://api.example.com/search
```

Read a saved run:

```sh
p99 report run.json
```

Compare two saved runs:

```sh
p99 compare before.json after.json --max-p99-regression 20%
```

Print build information:

```sh
p99 version
```

Install shell completion:

```sh
p99 completion zsh > ~/.zsh/completions/_p99
```

Watch a live endpoint:

```sh
p99 watch --rps 50 --window 5s https://api.example.com/search
```

Analyze spans:

```sh
p99 spans traces.json
```

## Documentation

The full documentation lives in the GitHub Wiki. The source pages are kept in [wiki/](wiki/) so they can be reviewed and versioned with the code.

Start with [wiki/Home.md](wiki/Home.md), then read the command pages for detailed explanations of every command and flag.

## Development

Run the test suite:

```sh
go test ./...
```

Run Nix checks:

```sh
nix flake check --all-systems --no-build
```

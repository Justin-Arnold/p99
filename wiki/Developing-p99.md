# Developing p99

This page explains how to work on `p99`.

## Requirements

You need Go. Nix is optional but recommended for reproducible builds.

## Run Tests

```sh
go test ./...
```

With Nix:

```sh
nix shell nixpkgs#go -c go test ./...
```

## Run Vet

```sh
go vet ./...
```

## Build

```sh
go build ./cmd/p99
```

With Nix:

```sh
nix build .#p99
```

## Check Flake Outputs

```sh
nix flake check --all-systems --no-build
```

## Package Layout

```text
cmd/p99/
internal/cli/
internal/probe/
internal/latency/
internal/output/
internal/compare/
internal/errors/
internal/profile/
internal/runtimesignal/
internal/spans/
internal/timeutil/
```

## Design Principles

- Prefer clear interfaces over clever abstractions.
- Keep comments focused on why.
- Keep tests local and deterministic.
- Avoid real network dependencies in tests.
- Treat tail latency as a distribution problem.
- Treat correlation as evidence, not proof.

## Adding a Command

1. Add the command implementation under `internal/cli`.
2. Wire it in `internal/cli/root.go`.
3. Put reusable logic in a focused internal package.
4. Add unit tests for the package.
5. Add CLI-level tests for exit behavior and output.
6. Add a wiki page for the command and its flags.

## Adding an Export Format

1. Add writer functions under `internal/output`.
2. Add tests with stable string checks.
3. Wire the format through CLI export helpers.
4. Document when the format should and should not be used.

## Testing Guidance

Use `httptest.Server` for HTTP behavior.

Avoid real DNS, internet calls, sleeps longer than needed, or timing assumptions that are too tight.

Prefer controlled handlers, fake transports, and broad timing expectations.

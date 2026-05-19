# Releasing p99

Releases are built from Git tags with GoReleaser.

## What a Release Produces

A release publishes:

- Linux amd64 binary archive
- Linux arm64 binary archive
- macOS amd64 binary archive
- macOS arm64 binary archive
- checksum file
- changelog
- Homebrew formula update

The archive names are stable because the install script and downstream packaging rely on them:

```text
p99_linux_amd64.tar.gz
p99_linux_arm64.tar.gz
p99_darwin_amd64.tar.gz
p99_darwin_arm64.tar.gz
checksums.txt
```

## Release Flow

Create and push a version tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The `release` workflow runs on tags that start with `v`.

Before publishing, the workflow runs the GoReleaser hooks:

```sh
go mod tidy
go test ./...
```

Those hooks are intentionally boring. A release should prove that the module is tidy and the test suite passes before publishing binaries.

## Homebrew Tap

The release workflow is configured to publish a formula to:

```text
justin/homebrew-tap
```

The formula lives under:

```text
Formula/p99.rb
```

The workflow needs a repository secret named:

```text
HOMEBREW_TAP_TOKEN
```

That token must be able to write to the tap repository. `GITHUB_TOKEN` can publish the release in this repository, but it cannot normally push to a separate tap repository.

## Local Checks

Validate the GoReleaser configuration before cutting a tag:

```sh
goreleaser check
```

Run a snapshot release locally when you want to inspect the generated archives without publishing:

```sh
goreleaser release --snapshot --clean
```

Snapshot output is written under `dist/`.

## Why This Matters

Most users should not need Go or Nix to install `p99`.

Release binaries make `p99` practical for CI images, production debugging hosts, and laptops where the user just wants the tool. Homebrew gives macOS and Linux users a familiar upgrade path. The install script gives the same release artifacts a small dependency-free path for machines where Homebrew is not available.

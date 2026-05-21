# Command: completion

`p99 completion` prints shell completion scripts.

```sh
p99 completion bash
p99 completion zsh
p99 completion fish
```

## What Problem It Solves

`p99` has enough commands and flags that completion is useful for daily work. Completion reduces flag typos and makes the command surface easier to discover from the shell.

## Shells

### bash

```sh
p99 completion bash > ~/.local/share/bash-completion/completions/p99
```

The exact directory depends on how your bash completion is configured.

### zsh

```sh
mkdir -p ~/.zsh/completions
p99 completion zsh > ~/.zsh/completions/_p99
```

Make sure that directory is present in `fpath` before `compinit` runs.

### fish

```sh
p99 completion fish > ~/.config/fish/completions/p99.fish
```

Fish loads completion files from that directory automatically.

## NixOS and nix-darwin

When using the flake module, prefer declarative completion installation:

```nix
{
  programs.p99 = {
    enable = true;
    enableCompletions = true;
    completionShells = [ "zsh" ];
  };
}
```

The module generates completion files from the same `p99 completion` command and installs them through `environment.systemPackages`.

## Flags

This command has no flags. It accepts one positional shell name: `bash`, `zsh`, or `fish`.

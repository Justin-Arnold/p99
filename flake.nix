{
  description = "p99 latency profiler";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      lib = nixpkgs.lib;
      systems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];
      forAllSystems = f: lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      p99Module =
        { config, lib, pkgs, ... }:
        let
          cfg = config.programs.p99;
          defaultPackage = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
        in
        {
          options.programs.p99 = {
            enable = lib.mkEnableOption "p99 latency profiler";

            package = lib.mkOption {
              type = lib.types.package;
              default = defaultPackage;
              defaultText = lib.literalExpression "inputs.p99.packages.\${pkgs.stdenv.hostPlatform.system}.default";
              description = "The p99 package to install.";
            };
          };

          config = lib.mkIf cfg.enable {
            environment.systemPackages = [ cfg.package ];
          };
        };
    in
    {
      packages = forAllSystems (pkgs: {
        p99 = pkgs.callPackage ./nix/package.nix { };
        default = self.packages.${pkgs.stdenv.hostPlatform.system}.p99;
      });

      checks = forAllSystems (pkgs: {
        package = self.packages.${pkgs.stdenv.hostPlatform.system}.p99;

        module =
          let
            evaluated = lib.evalModules {
              specialArgs = { inherit pkgs; };
              modules = [
                (
                  { lib, ... }:
                  {
                    options.environment.systemPackages = lib.mkOption {
                      type = lib.types.listOf lib.types.package;
                      default = [ ];
                    };
                  }
                )
                p99Module
                {
                  programs.p99.enable = true;
                }
              ];
            };
            packageCount = builtins.length evaluated.config.environment.systemPackages;
          in
          pkgs.runCommand "p99-module-check" { } ''
            test ${toString packageCount} -eq 1
            touch "$out"
          '';
      });

      apps = forAllSystems (pkgs: {
        p99 = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.p99}/bin/p99";
          meta = {
            description = "Run the p99 latency profiler";
          };
        };
        default = self.apps.${pkgs.stdenv.hostPlatform.system}.p99;
      });

      overlays.default = final: _prev: {
        p99 = final.callPackage ./nix/package.nix { };
      };

      nixosModules = {
        p99 = p99Module;
        default = self.nixosModules.p99;
      };

      darwinModules = {
        p99 = p99Module;
        default = self.darwinModules.p99;
      };
    };
}

{
  buildGoModule,
  lib,
}:

buildGoModule {
  pname = "p99";
  version = "0.2.0";

  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../cmd
      ../internal
      ../go.mod
      ../README.md
    ];
  };

  vendorHash = null;

  subPackages = [ "cmd/p99" ];

  meta = {
    description = "Latency profiler focused on tail latency";
    homepage = "https://github.com/Justin-Arnold/p99";
    mainProgram = "p99";
    platforms = lib.platforms.darwin ++ lib.platforms.linux;
  };
}

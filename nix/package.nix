{
  buildGoModule,
  lib,
}:

buildGoModule {
  pname = "p99";
  version = "0.1.0";

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
    homepage = "https://github.com/justin/p99";
    mainProgram = "p99";
    platforms = lib.platforms.darwin ++ lib.platforms.linux;
  };
}

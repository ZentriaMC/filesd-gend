{
  description = "filesd-gend";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    docker-tools.url = "github:ZentriaMC/docker-tools";

    docker-tools.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, flake-utils, docker-tools }:
    let
      supportedSystems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];
    in
    flake-utils.lib.eachSystem supportedSystems (system:
      let
        rev = self.rev or "dirty";
        pkgs = nixpkgs.legacyPackages.${system};
      in
      rec {
        packages.filesd-gend = pkgs.callPackage ./default.nix {
          inherit rev;
        };

        packages.dockerImage = pkgs.callPackage
          ({ lib, runCommandNoCC, dockerTools, dumb-init, cacert, tzdata, filesd-gend, name ? "filesd-gend", tag ? filesd-gend.version }: dockerTools.buildLayeredImage {
            inherit name tag;
            config = {
              Env = [
                "PATH=/usr/bin"
              ];
              ExposedPorts = {
                "5555/tcp" = { };
              };
              Labels = {
                "org.opencontainers.image.source" = "https://github.com/ZentriaMC/filesd-gend";
              };
              Entrypoint = [ "/usr/bin/dumb-init" "--" ];
              Cmd = [ "filesd-gend" ];
            };

            contents =
              let
                inherit (docker-tools.lib) setupFHSScript symlinkCACerts;

                fhsScript = setupFHSScript {
                  inherit pkgs;
                  targetDir = "$out/usr";
                  paths = {
                    bin = [
                      dumb-init
                      filesd-gend
                    ];
                  };
                };
              in
              [
                (runCommandNoCC "filesd-gend-nix-base" { } ''
                  ${fhsScript}
                  ln -s usr/bin $out/bin
                  ln -s bin $out/usr/sbin
                  ln -s usr/bin $out/sbin
                  ln -s usr/lib $out/lib
                  ln -s usr/lib $out/lib64

                  ${symlinkCACerts { inherit cacert; targetDir = "$out"; }}

                  ln -s ${tzdata}/share/zoneinfo $out/etc/zoneinfo
                  ln -s /etc/zoneinfo/UTC $out/etc/localtime
                  echo "ID=distroless" > $out/etc/os-release
                '')
              ];
          })
          {
            inherit (packages) filesd-gend;
          };

        defaultPackage = packages.filesd-gend;

        devShell = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.golangci-lint
            pkgs.gopls
          ];
        };
      });
}

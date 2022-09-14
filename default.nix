{ lib, buildGoModule, rev }:
buildGoModule rec {
  pname = "filesd-gend";
  version = rev;

  src = lib.cleanSource ./.;

  vendorSha256 = "sha256-E7qVEYPqFWf/iEWfW5MiaLJgg4Oed5iN+eTo/SyJRRY=";
  subPackages = [ "cmd/filesd-gend" ];

  meta = with lib; {
    description = "Generates a service discovery JSON file for Prometheus";
    homepage = "https://github.com/ZentriaMC/filesd-gend";
    license = licenses.gpl3;
    platforms = platforms.unix;
  };
}

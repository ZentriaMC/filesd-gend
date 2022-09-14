{ lib, buildGo119Module, rev }:
buildGo119Module rec {
  pname = "filesd-gend";
  version = rev;

  src = lib.cleanSource ./.;

  vendorSha256 = "sha256-5cMr5TquP+PjtClpqOFlotjMtTkiTvMIKDt9OQpze5Q=";
  subPackages = [ "cmd/filesd-gend" ];

  meta = with lib; {
    description = "Generates a service discovery JSON file for Prometheus";
    homepage = "https://github.com/ZentriaMC/filesd-gend";
    license = licenses.gpl3;
    platforms = platforms.unix;
  };
}

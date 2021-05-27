{ lib, stdenv, buildGoModule, zfs }:
buildGoModule rec {
  pname = "filesd-gend";
  version = "0.0.1";

  src = lib.cleanSource ./.;

  vendorSha256 = "1jd3nc8j6gb6ska39v1mcjxazhmhfkzzjvwq5r2akp24cis9pawa";
  subPackages = [ "cmd/filesd-gend" ];

  meta = with lib; {
    description = "Generates a service discovery JSON file for Prometheus";
    homepage = "https://github.com/ZentriaMC/filesd-gend";
    license = licenses.gpl3;
    platforms = platforms.linux;
  };
}

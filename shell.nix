with import <nixpkgs> {};

stdenv.mkDerivation {
name = "firecracker";

buildInputs = [
  go
  docker
  openssh
  lighttpd    
];

shellHook = ''
  cp docs/nixshell/lighttpd.conf /tmp/
  lighttpd -f /tmp/lighttpd.conf 
  '';
}


with import <nixpkgs> {};

stdenv.mkDerivation {
name = "firecracker";

buildInputs = [
		syft
		grype
		docker
		trivy
		stdenv.cc.cc
		docker-credential-helpers
		go
];

SOURCE_DATE_EPOCH = 315532800;
PROJDIR = "${toString ./.}";
S_NETWORK="host";

shellHook = ''
		export LD_LIBRARY_PATH="${pkgs.stdenv.cc.cc.lib}/lib"
		export PATH=/tmp/bin:/home/andreas/go/bin/:$PATH
		export GOTMPDIR=/tmp
		export TMPDIR=/tmp
		mkdir /tmp/bin
		'';
}

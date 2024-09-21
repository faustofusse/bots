{
  description = "pkg-config solution";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable"; # nixpkgs-unstable # 24.05
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = import nixpkgs { inherit system; config.allowUnfree = true; }; in {
        devShell = with pkgs; mkShell rec {
          buildInputs = [ dbus alsa-lib google-chrome ];
          nativeBuildInputs = [ pkg-config ];
        };
      }
    );
}

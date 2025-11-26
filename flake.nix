
{
  description = "A basic flake with a shell";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  inputs.systems.url = "github:nix-systems/default";
  inputs.flake-utils = {
    url = "github:numtide/flake-utils";
    inputs.systems.follows = "systems";
  };

  outputs =
    { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "sesh";
          version = "0.1.0";

          src = ./.;

          vendorHash = "sha256-XtKA7DrjUeAZDPMFa6Z6dkDLwsAz8N/91yRJoi9yewA=";
          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            wrapProgram $out/bin/sesh \
              --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.git pkgs.fzf ]}
          '';

          meta = with pkgs.lib; {
            description = "A modern git workspace and session manager";
            homepage = "https://github.com/benoctopus/sesh";
            license = licenses.mit;
            mainProgram = "sesh";
          };
        };

        devShells.default = pkgs.mkShell { packages = [
          pkgs.git
          pkgs.go_1_24
          pkgs.go-task
          pkgs.gopls
          pkgs.golines
          pkgs.golangci-lint
          pkgs.jq
          pkgs.yq
          pkgs.gofumpt
          pkgs.uutils-coreutils-noprefix
          pkgs.shellcheck
          pkgs.cobra-cli
          pkgs.tree
          pkgs.fzf
        ]; };
      }
    );
}

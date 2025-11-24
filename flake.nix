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
        devShells.default = pkgs.mkShell { packages = [ 
          pkgs.bashInteractive 
          pkgs.git
          pkgs.go_1_25
          pkgs.go-task
          pkgs.claude-code
          pkgs.gopls
          pkgs.go-lines
          pkgs.golangci-lint
          pkgs.jq
          pkgs.yq
          pkgs.gofumpt
          pkgs.uutils-coreutils-noprefix
          pkgs.shellcheck
        ]; };
      }
    );
}

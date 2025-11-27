{
  description = "A modern git workspace and session manager that integrates git worktrees with terminal multiplexers";
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
        # Use ARM64 Linux for Apple Silicon, x86_64 for Intel
        linuxSystem = if pkgs.stdenv.isAarch64 then "aarch64-linux" else "x86_64-linux";
        pkgsLinux = nixpkgs.legacyPackages.${linuxSystem};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "sesh";
          version = "0.1.8";

          src = ./.;

          vendorHash = "sha256-XtKA7DrjUeAZDPMFa6Z6dkDLwsAz8N/91yRJoi9yewA=";
          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            wrapProgram $out/bin/sesh \
              --prefix PATH : ${
                pkgs.lib.makeBinPath [
                  pkgs.git
                  pkgs.fzf
                  pkgs.screen
                ]
              }
          '';
          checkPhase = "";
          meta = with pkgs.lib; {
            description = "A modern git workspace and session manager that integrates git worktrees with terminal multiplexers (tmux, zellij)";
            homepage = "https://github.com/benoctopus/sesh";
            license = licenses.mit;
            mainProgram = "sesh";
            maintainers = [ ];
            platforms = platforms.unix;
          };
        };

        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.git
            pkgs.go_1_24
            pkgs.ranger
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
            pkgs.tmux
            pkgs.zellij
          ];
        };
      }
    );
}

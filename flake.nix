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
        packages.default = pkgs.rustPlatform.buildRustPackage {
          pname = "sesh";
          version = "0.1.0";

          src = ./.;

          cargoLock = {
            lockFile = ./Cargo.lock;
          };

          nativeBuildInputs = [
            pkgs.pkg-config
            pkgs.makeWrapper
          ];

          buildInputs = [
            pkgs.openssl
            pkgs.libgit2
            pkgs.sqlite
          ];

          postInstall = ''
            wrapProgram $out/bin/sesh \
              --prefix PATH : ${
                pkgs.lib.makeBinPath [
                  pkgs.git
                  pkgs.fzf
                  pkgs.tmux
                ]
              }
          '';

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
            # Rust toolchain
            pkgs.rustc
            pkgs.cargo
            pkgs.clippy
            pkgs.rustfmt
            pkgs.rust-analyzer

            # Build dependencies
            pkgs.pkg-config
            pkgs.openssl
            pkgs.libgit2
            pkgs.sqlite

            # Testing tools
            pkgs.git
            pkgs.fzf
            pkgs.tmux
            pkgs.zellij

            # Development utilities
            pkgs.jq
            pkgs.yq
            pkgs.tree
            pkgs.shellcheck
            pkgs.go-task
          ];

          # Set environment variables for build dependencies
          LIBGIT2_SYS_USE_PKG_CONFIG = "1";
          PKG_CONFIG_PATH = "${pkgs.openssl.dev}/lib/pkgconfig:${pkgs.libgit2}/lib/pkgconfig:${pkgs.sqlite.dev}/lib/pkgconfig";
        };
      }
    );
}

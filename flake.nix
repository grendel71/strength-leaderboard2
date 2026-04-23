{
  description = "Strength Leaderboard - Go + HTMX";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-migrate
            sqlc
            postgresql
          ];

          shellHook = ''
            echo "Strength Leaderboard dev shell"
            echo "Go $(go version | cut -d' ' -f3)"
          '';
        };
      }
    );
}

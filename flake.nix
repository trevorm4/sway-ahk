{
  description = "sway-ahk: key automation for Sway";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: rec {
        sway-ahk = pkgs.buildGoModule {
          pname = "sway-ahk";
          version = "0.1.0";
          src = self;
          vendorHash = "sha256-g+yaVIx4jxpAQ/+WrGKxhVeliYx7nLQe/zsGpxV4Fn4=";
          nativeBuildInputs = [ pkgs.makeWrapper ];
          postInstall = ''
            wrapProgram $out/bin/sway-ahk --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.libinput ]}
          '';
        };
        default = sway-ahk;
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            libinput
          ];
        };
      });
    };
}

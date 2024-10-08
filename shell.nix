let
  nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-unstable";
  pkgs = import nixpkgs {
    config = { };
    overlays = [ ];
  };
in
pkgs.mkShellNoCC {
  packages = with pkgs; [
    cobra-cli
    goreleaser
    vhs
  ];
  shellHook = ''
    export PATH=$PATH:$PWD/dist/ytui_linux_amd64_v1/
  '';
}

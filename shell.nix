{ pkgs ? import <nixpkgs> {} }:
with pkgs;
mkShell rec {
    name = "go";
    
    tools = [
      go
      buf
      protoc-gen-go
    ];
    
    libs = [
      stdenv.cc.cc
    ];

    buildInputs = tools ++ libs;
    LD_LIBRARY_PATH = lib.makeLibraryPath libs;
    GOROOT = "${pkgs.go}/share/go";
}

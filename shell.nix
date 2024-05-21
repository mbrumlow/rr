{ pkgs ? import <nixpkgs> {} }:
let

  pkgs = import (builtins.fetchTarball {
    url = "https://github.com/NixOS/nixpkgs/archive/eb090f7b923b1226e8beb954ce7c8da99030f4a8.tar.gz";
  }) {};
  
  
in 
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

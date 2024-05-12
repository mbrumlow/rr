{ pkgs ? import <nixpkgs> {} }:
let
in
with pkgs;
mkShell rec {
  name = "ge";
  tools = [
    buf
    protoc-gen-go
  ];
  libs = [
  ];

  shellHook = '''';

  buildInputs = tools ++ libs;
  LD_LIBRARY_PATH = lib.makeLibraryPath libs;
}

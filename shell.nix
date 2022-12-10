{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [ bash go_1_19 git protobuf protoc-gen-go protoc-gen-go-grpc ];
}

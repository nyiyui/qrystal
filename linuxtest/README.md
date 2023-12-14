# Non-NixOS Integration Testing

See `/test.nix` for integration testing using NixOS's testing library.

## Goals

- test install scripts, etc for non-NixOS Linux systems

## Non-goals

- test Qrystal networking (i.e. does Qrystal itself work?)
  - setting up tests with custom network setups (e.g. multiple nodes in a shared LAN) is too much duplicate work for both NixOS tests and this.

## TODO

- do we actually run this in a VM, or just a container?
  - if it's just installation and basic (i.e. no network access) testing, then a container build script should suffice

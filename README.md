# Qrystal

[Website/Docs](https://nyiyui.ca/qrystal) /
[Github.com](https://github.com/nyiyui/qrystal)

Qrystal /kristl/ sets up several WireGuard tunnels between servers.
In addition, it provides centralised configuration management.

## Installation

Make sure to open the appropriate ports (defaults listed below):
- CS: 39252/tcp for Nodes and 39253 for utilities
- The WireGuard ports for UDP (from the expected peers)

### Most Linux distros

```sh
$ git clone https://github.com/nyiyui/qrystal
$ cd qrystal
$ mkdir build && cd build
$ make src=.. -f ../Makefile
# make src=.. -f ../Makefile install
```

Then, enable/start `qrystal-runner.service` and/or `qrystal-cs.service`
(depending on what you want to run).

### NixOS

Flakes are recommended. See `flake.nix` for options.

## Installation from Generic Archive

```
# make pre_install # if Qrystal services are already running
# make src=. install
# systemctl start qrystal-runner # for Node
# systemctl start qrystal-cs # for CS
```

## TODO

- confine qrystal-node and qrystal-cs (using systemd's options)
- configure existing interfaces without disrupting connections (as much as possible)
- two CNs on the same Node seem to have trouble syncing PublicKeys
  - seems like PublicKeys are mixed up between CNs…what‽

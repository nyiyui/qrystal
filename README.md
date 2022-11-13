# Qrystal

![Jekyll CD Status](https://github.com/nyiyui/qrystal/workflows/Jekyll/badge.svg)
![makepkg CI Status](https://github.com/nyiyui/qrystal/workflows/makepkg/badge.svg)
![Build CI Status](https://github.com/nyiyui/qrystal/workflows/Build/badge.svg)

[Website and Docs](https://nyiyui.ca/qrystal) /
[On Github.com](https://github.com/nyiyui/qrystal)

Qrystal sets up several WireGuard tunnels between servers. In addition, it provides centralised configuration management.

## Installation from Generic Archive

```
# make pre_install # if Qrystal services are already running
# make src=. install
# systemctl start qrystal-runner # for Node
# systemctl start qrystal-cs # for CS
```

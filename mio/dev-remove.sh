#!/bin/sh

set -eu

name="$1"
1>&2 echo "removing $name"

wg-quick down "$name"
rm -f "/etc/wireguard/$name"

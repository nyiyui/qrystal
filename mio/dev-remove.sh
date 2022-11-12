#!/bin/sh

set -eu

name="$1"

log=$(mktemp)
wg-quick down "$name" 2> $log || 1>&2 cat $log
rm $log
rm -f "/etc/wireguard/$name"

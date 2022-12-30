#!/bin/sh

set -eu

name="$1"
wg_quick="$2"

log=$(mktemp)
$wg_quick down "$name" 2> $log || 1>&2 cat $log
rm $log
rm -f "/etc/wireguard/$name"

#!/bin/sh

set -eu

name="$1"
wg_quick="$2"
persist="$3"

log=$(mktemp)
$wg_quick down "$name" 2> $log || 1>&2 cat $log

if [[ "$persist" == 'systemd' ]]; then
  unit_name="wg-quick@$name.service"
  systemctl disable "$unit_name" 2> $log || (1>&2 cat $log; 1>&2 cat "$config_path")
fi

rm $log
rm -f "/etc/wireguard/$name"

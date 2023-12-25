#!/bin/sh

set -eu

name="$1"
wg_quick="$2"
persist="$3"

if [ -z "$wg_quick" ]; then
  wg_quick="$(which wg-quick)"
fi

log=$(mktemp)
$wg_quick down "$name" 2> $log || 1>&2 cat $log

if [[ "$persist" == 'systemd' ]]; then
  unit_name="wg-quick@$name.service"
  systemctl disable "$unit_name" 2> $log || (1>&2 cat $log)
fi

rm $log
rm -f "/etc/wireguard/$name"

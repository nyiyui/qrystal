set -eux

name="$1"

wg-quick down "$name"
rm -f "/etc/wireguard/$name"

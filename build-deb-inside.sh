#!/bin/bash

set -eux

go version

origpath="$PATH"

apt-get update
apt-get install -y debmake

rm -rf /source/build2 # just in case
rm -f /source/cs-push # ditto

mkdir /build

tar -czvf /build/orig.tar.gz /source

cd /source/debian
make
ver=$(cat ./pkgver)
rm ./pkgver

cd /source
path="/build/qrystal-$ver"
cp -r /source "$path"
cp /build/orig.tar.gz "/build/qrystal_$ver.orig.tar.gz"
cd "$path"
debmake
debuild --prepend-path "$origpath"

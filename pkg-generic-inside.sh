#!/bin/bash

set -eux

apt-get -qq update
apt-get -qq install -y make
git config --global --add safe.directory /source

prefix=/source

current=$(cd /source && git rev-parse @)

mkdir /build
cd /build
make -f $prefix/Makefile src=/source pkgdir=/build all cs-push
cp -r $prefix/config .
mkdir ./mio
cp $prefix/mio/dev-*.sh ./mio/
cp $prefix/Makefile .
cp $prefix/README.md .
echo $current > ./BUILD_COMMIT
ls -al /build

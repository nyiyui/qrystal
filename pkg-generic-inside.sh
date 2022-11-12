#!/bin/bash

set -eux

apt-get -qq update
apt-get -qq install -y make

prefix=/source

current=$(cd /source && git rev-parse @)

mkdir /build
cd /build
make -f $prefix/Makefile src=/source pkgdir=/build build2 cs-push
cp -r $prefix/config .
mkdir ./mio
cp $prefix/mio/dev-*.sh ./mio/
cp $prefix/Makefile .
echo $current > ./BUILD_COMMIT
ls -al /build
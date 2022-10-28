#!/bin/bash

set -eux

apt-get -qq update
apt-get -qq install -y make

prefix=/source

mkdir /build
cd /build
make -f $prefix/Makefile src=/source pkgdir=/build build2 cs-push
cp -r $prefix/config .
cp $prefix/mio/dev-*.sh .
ls -al /build

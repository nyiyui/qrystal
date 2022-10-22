#!/bin/bash

set -x

image='archlinux'
container=$(docker create -it $image)
docker cp . $container:'/build'
docker start $container
docker exec -w '/build' $container 'bash' '-c' 'pacman -Sy --noconfirm go fakeroot base-devel && useradd pkgmaker && su pkgmaker makepkg'
docker cp $container:'/build/build2' .
docker kill $container
docker rm $container


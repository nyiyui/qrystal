#!/bin/bash

set -x

image='golang:1.19-buster'
container=$(docker create -it $image)
docker cp . $container:'/source'
docker start $container
docker exec $container 'bash' '/source/pkg-generic-inside.sh'
td=$(mktemp -d)
docker cp $container:'/build' "$td" && tar cavf ./build.tar.zst -C "$td" .
rm -r "$td"
docker kill $container
docker rm $container

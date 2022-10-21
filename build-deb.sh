#!/bin/bash

set -x

image='golang:1.19-buster'
container=$(docker create -it $image)
docker cp . $container:'/source'
docker start $container
docker exec $container 'bash' '/source/build-deb-inside.sh'
docker kill $container
docker rm $container

#!/bin/bash

set -x

image='golang:1.19-buster'
container=$(docker create -it $image)
docker cp . $container:'/build'
docker start $container
docker exec -w '/build' $container 'bash' '-c' 'rm -rf build2 && make build2'
docker cp $container:'/build/build2' .
docker kill $container
docker rm $container

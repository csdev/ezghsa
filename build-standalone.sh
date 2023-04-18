#!/bin/bash
set -eo pipefail

docker-compose build app
cid="$(docker create csang/ezghsa:latest)"
trap "docker rm $cid" EXIT
docker cp "${cid}:/opt/go/bin/ezghsa" .

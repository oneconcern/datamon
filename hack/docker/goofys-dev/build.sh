#!/bin/bash
set -e

TAG=$(date '+%Y%m%d')
docker build --pull -t "reg.onec.co/goofys:$TAG" -t "reg.onec.co/goofys:latest" .
docker push "reg.onec.co/goofys:$TAG"

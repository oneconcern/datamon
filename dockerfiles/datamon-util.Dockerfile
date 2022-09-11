# A utility image designed to be a lightweight tool for interacting
# with datamon as a part of kubernetes workloads
FROM reg.onec.co/datamon-alpine-base:latest as datamon

FROM golang:1.19-alpine as gcsfuse
RUN apk add --no-cache git
ENV GOPATH /go
RUN go install github.com/googlecloudplatform/gcsfuse@latest

FROM alpine:3.14
RUN apk add --no-cache ca-certificates fuse bash rsync && rm -rf /tmp/*
COPY --from=gcsfuse /go/bin/gcsfuse /usr/local/bin
COPY --from=datamon /stage /
ENV ZONEINFO /zoneinfo.zip

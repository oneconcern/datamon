# A base container with a datamon binary
#
# This image remains local and is not released
FROM golang:alpine as base

RUN mkdir -p /stage/data /stage/etc/ssl/certs &&\
  apk add --no-cache --quiet musl-dev gcc ca-certificates mailcap upx tzdata zip git make bash ncurses &&\
  update-ca-certificates &&\
  cp /etc/ssl/certs/ca-certificates.crt /stage/etc/ssl/certs/ca-certificates.crt &&\
  cp /etc/mime.types /stage/etc/mime.types

# https://golang.org/src/time/zoneinfo.go Copy the zoneinfo installed by musl-dev
WORKDIR /usr/share/zoneinfo
RUN zip -qr -0 /stage/zoneinfo.zip .

ARG version
ARG commit
ARG dirty

ENV VERSION ${version}
ENV GIT_COMMIT ${commit}
ENV GIT_DIRTY ${dirty}
ENV IMPORT_PATH github.com/oneconcern/datamon/cmd/datamon/cmd

ADD . /build
WORKDIR /build

#
# Build a base image with a compressed, statically linked datamon binary in /stage/usr/bin
#
RUN LDFLAGS='-s -w -linkmode external -extldflags "-static"' && \
    LDFLAGS="$LDFLAGS -X '${IMPORT_PATH}.Version=${VERSION}'" && \
    LDFLAGS="$LDFLAGS -X '${IMPORT_PATH}.BuildDate=$(date -u -R)'" && \
    LDFLAGS="$LDFLAGS -X '${IMPORT_PATH}.GitCommit=${GIT_COMMIT}'" && \
    LDFLAGS="$LDFLAGS -X '${IMPORT_PATH}.GitState=${GIT_DIRTY}'" && \
    go mod download && \
    go build -o /stage/usr/bin/datamon --ldflags "$LDFLAGS" ./cmd/datamon && \
    upx /stage/usr/bin/datamon &&\
    md5sum /stage/usr/bin/datamon

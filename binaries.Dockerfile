FROM golang:1.13-alpine3.10 as base

RUN mkdir -p /stage/data /stage/etc/ssl/certs &&\
  apk add --no-cache --quiet musl-dev gcc ca-certificates mailcap upx tzdata zip git &&\
  update-ca-certificates &&\
  cp /etc/ssl/certs/ca-certificates.crt /stage/etc/ssl/certs/ca-certificates.crt &&\
  cp /etc/mime.types /stage/etc/mime.types

# https://golang.org/src/time/zoneinfo.go Copy the zoneinfo installed by musl-dev
WORKDIR /usr/share/zoneinfo
RUN zip -qr -0 /stage/zoneinfo.zip .

ARG version_import_path
ARG version
ARG commit
ARG dirty

ENV VERSION_IMPORT_PATH ${version_import_path}
ENV VERSION ${version}
ENV GIT_COMMIT ${commit}
ENV GIT_DIRTY ${dirty}

ADD . /datamon
WORKDIR /datamon

RUN go mod download && \
    go get -u github.com/gobuffalo/packr/v2/packr2 && \
    (cd ./pkg/web && packr2)

# .{os} extension binaries are those distributed via github releases
RUN LDFLAGS='-s -w -linkmode external -extldflags "-static"' && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}Version=${VERSION}'" && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}BuildDate=$(date -u -R)'" && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}GitCommit=${GIT_COMMIT}'" && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}GitState=${GIT_DIRTY}'" && \
  go build -o /stage/usr/bin/datamon.linux --ldflags "$LDFLAGS" ./cmd/datamon

RUN LDFLAGS='-s -w -linkmode internal' && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}Version=${VERSION}'" && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}BuildDate=$(date -u -R)'" && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}GitCommit=${GIT_COMMIT}'" && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}GitState=${GIT_DIRTY}'" && \
  CGO_ENABLED=0 GOOS=darwin GOHOSTOS=linux go build -o /stage/usr/bin/datamon.mac --ldflags "$LDFLAGS" ./cmd/datamon

# additional binaries are provided for building distributable docker images
RUN cp /stage/usr/bin/datamon.linux /stage/usr/bin/datamon && \
    upx /stage/usr/bin/datamon && \
    md5sum /stage/usr/bin/datamon

RUN go build -o /stage/usr/bin/migrate --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/backup2blobs && \
    upx /stage/usr/bin/migrate && md5sum /stage/usr/bin/migrate

RUN go build -o /stage/usr/bin/datamon_metrics --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/metrics &&\
    upx /stage/usr/bin/datamon_metrics && md5sum /stage/usr/bin/datamon_metrics

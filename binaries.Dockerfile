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

ENV IMPORT_PATH ${version_import_path:-"github.com/oneconcern/datamon/cmd/datamon/cmd."}
ENV VERSION ${version:-"dev"}
ENV GIT_COMMIT ${commit}
ENV GIT_DIRTY ${dirty}

ADD . /datamon
WORKDIR /datamon

RUN go get -u github.com/gobuffalo/packr/v2/packr2 && \
    go get -u github.com/mitchellh/gox && \
    (cd ./pkg/web && packr2) && \
    go mod download

# .{os} extension binaries are those distributed via github releases
ENV LDFLAGS "-s -w -X '${IMPORT_PATH}Version=${VERSION}' -X '${IMPORT_PATH}GitCommit=${GIT_COMMIT}'"
ENV TARGET "/stage/usr/bin"

# Ref: https://github.com/mitchellh/gox/issues/55 for CGO_ENABLED=0
RUN CGO_ENABLED=0 LDFLAGS="${LDFLAGS} '-X{IMPORT_PATH}BuildDate=$(date -u -R)'" \
    gox -os "linux darwin" -arch "amd64" -output "${TARGET}/{{.Dir}}_{{.OS}}_{{.Arch}}" -ldflags "$LDFLAGS" ./cmd/datamon
RUN CGO_ENABLED=0 LDFLAGS="${LDFLAGS} '-X{IMPORT_PATH}BuildDate=$(date -u -R)'" \
    gox -os "linux"        -arch "amd64" -output "${TARGET}/migrate_{{.OS}}_{{.Arch}}"  -ldflags "$LDFLAGS" ./cmd/backup2blobs
RUN CGO_ENABLED=0 LDFLAGS="${LDFLAGS} '-X{IMPORT_PATH}BuildDate=$(date -u -R)'" \
    gox -os "linux"        -arch "amd64" -output "${TARGET}/datamon_{{.Dir}}_{{.OS}}_{{.Arch}}" -ldflags "$LDFLAGS" ./cmd/metrics
RUN CGO_ENABLED=0 LDFLAGS="${LDFLAGS} '-X{IMPORT_PATH}BuildDate=$(date -u -R)'" \
    gox -os "linux"        -arch "amd64" -output "${TARGET}/datamon_{{.Dir}}_{{.OS}}_{{.Arch}}" -ldflags "$LDFLAGS" ./cmd/sidecar_param

# compatibility with previous released artifacts
RUN if [ -f ${TARGET}/datamon_darwin_amd64 ] ; then  cp ${TARGET}/datamon_darwin_amd64 ${TARGET}/datamon.mac ;fi && \
    if [ -f ${TARGET}/datamon_linux_amd64 ] ; then  cp ${TARGET}/datamon_linux_amd64 ${TARGET}/datamon.linux ;fi && \
    if [ -f ${TARGET}/datamon_metrics_linux_amd64 ] ; then  cp ${TARGET}/datamon_linux_amd64 ${TARGET}/datamon_metrics ;fi && \
    cd ${TARGET};for bin in `ls -1` ; do upx ${bin} && md5sum ${bin} >> ${bin}.md5 && sha256sum ${bin} > ${bin}.sha256 ; done && \
    if [ -f ${TARGET}/datamon_linux_amd64 ] ; then  cp ${TARGET}/datamon_linux_amd64 ${TARGET}/datamon ;fi && \
    if [ -f ${TARGET}/migrate_linux_amd64 ] ; then  cp ${TARGET}/datamon_linux_amd64 ${TARGET}/migrate ;fi && \
    if [ -f ${TARGET}/datamon_sidecar_param_linux_amd64 ] ; then  cp ${TARGET}/datamon_sidecar_param_linux_amd64 ${TARGET}/datamon_sidecar_param ;fi


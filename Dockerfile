FROM golang:alpine as base

ADD hack/create-netrc.sh /usr/bin/create-netrc

RUN mkdir -p /stage/data /stage/etc/ssl/certs &&\
  create-netrc &&\
  apk add --no-cache --quiet musl-dev gcc ca-certificates mailcap upx tzdata zip git &&\
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

ADD . /datamon
WORKDIR /datamon

RUN LDFLAGS='-s -w -linkmode external -extldflags "-static"' && \
  LDFLAGS="$LDFLAGS -X 'github.com/oneconcern/datamon/cmd/datamon/cmd.Version=${VERSION}'" && \
  LDFLAGS="$LDFLAGS -X 'github.com/oneconcern/datamon/cmd/datamon/cmd.BuildDate=$(date -u -R)'" && \
  LDFLAGS="$LDFLAGS -X 'github.com/oneconcern/datamon/cmd/datamon/cmd.GitCommit=${GIT_COMMIT}'" && \
  LDFLAGS="$LDFLAGS -X 'github.com/oneconcern/datamon/cmd/datamon/cmd.GitState=${GIT_DIRTY}'" && \
  go build -o /stage/usr/bin/datamon --ldflags "$LDFLAGS" ./cmd/datamon && \
  upx /stage/usr/bin/datamon && \
  md5sum /stage/usr/bin/datamon

#Build the dist image
FROM scratch
COPY --from=base /stage /
ENV ZONEINFO /zoneinfo.zip
ENTRYPOINT [ "datamon" ]
CMD ["--help"]


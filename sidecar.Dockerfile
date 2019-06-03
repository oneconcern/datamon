FROM golang:alpine as base

ADD hack/create-netrc.sh /usr/bin/create-netrc

RUN mkdir -p /stage/data /stage/etc/ssl/certs &&\
  create-netrc &&\
  apk add --no-cache musl-dev gcc ca-certificates mailcap upx tzdata zip git bash fuse &&\
  update-ca-certificates &&\
  cp /etc/ssl/certs/ca-certificates.crt /stage/etc/ssl/certs/ca-certificates.crt &&\
  cp /etc/mime.types /stage/etc/mime.types

# https://golang.org/src/time/zoneinfo.go Copy the zoneinfo installed by musl-dev
WORKDIR /usr/share/zoneinfo
RUN zip -r -0 /stage/zoneinfo.zip .

ADD . /datamon
WORKDIR /datamon

RUN go build -o /stage/usr/bin/datamon --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/datamon
RUN upx /stage/usr/bin/datamon
RUN md5sum /stage/usr/bin/datamon


# dist-alike during development/debug
RUN cp /stage/usr/bin/datamon /usr/bin/datamon

ADD ./hack/fuse-demo/datamon.yaml /root/.datamon/datamon.yaml

# Build the dist image
FROM ubuntu:latest
RUN apt-get update && apt-get install -y --no-install-recommends fuse &&\
  apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN echo "allow_root" >> /etc/fuse.conf

COPY --from=base /stage /
ENV ZONEINFO /zoneinfo.zip

ADD ./hack/fuse-demo/datamon.yaml /root/.datamon/datamon.yaml

RUN useradd -u 1020 -ms /bin/bash developer
RUN groupadd -g 2000 developers
RUN usermod -g developers developer
RUN chown -R developer:developers /usr/bin/datamon
USER developer
ENTRYPOINT [ "datamon" ]

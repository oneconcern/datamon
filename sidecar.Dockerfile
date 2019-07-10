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

# RUN go build -o /stage/usr/bin/datamon --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/datamon
# RUN upx /stage/usr/bin/datamon
# RUN md5sum /stage/usr/bin/datamon


# # dist-alike during development/debug
# RUN cp /stage/usr/bin/datamon /usr/bin/datamon

# ADD ./hack/fuse-demo/datamon.yaml /root/.datamon/datamon.yaml

# Build the dist image
FROM debian:buster
RUN apt-get update && apt-get install -y --no-install-recommends fuse &&\
  apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN echo "allow_root" >> /etc/fuse.conf

RUN echo "direct_io" >> /etc/fuse.conf
RUN echo "sync_read" >> /etc/fuse.conf

# COPY --from=base /stage /
# ENV ZONEINFO /zoneinfo.zip



ADD ./hack/fuse-demo/datamon.yaml /root/.datamon/datamon.yaml

ADD hack/fuse-demo/wrap_datamon.sh .
ADD hack/fuse-demo/wrap_application.sh .

# USER root
RUN chmod a+x wrap_datamon.sh
# USER developer

RUN useradd -u 1020 -ms /bin/bash developer
RUN groupadd -g 2000 developers
RUN usermod -g developers developer
# RUN chown -R developer:developers /usr/bin/datamon

# USER developer



RUN apt-get update &&\
    curl -sL https://deb.nodesource.com/setup_10.x  | bash &&\
    apt-get install -y \
        git \
        golang-go &&\
    apt-get autoremove -yqq &&\
    apt-get clean -y &&\
    apt-get autoclean -yqq &&\
    rm -rf \
        /tmp/* \
        /var/tmp/* \
        /var/lib/apt/lists/* \
        /usr/share/doc/* \
        /usr/share/locale/* \
        /var/cache/debconf/*-old

ADD . /datamon
WORKDIR /datamon

USER developer
RUN mkdir /home/developer/go
ENV GOPATH /home/developer/go
RUN go build -o /home/developer/datamon.out ./cmd/datamon

USER root
WORKDIR /
RUN cp /home/developer/datamon.out /usr/bin/datamon
RUN chown -R developer:developers /usr/bin/datamon
USER developer

ENTRYPOINT [ "datamon" ]

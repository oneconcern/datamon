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


ARG version_import_path
ARG version
ARG commit
ARG dirty

ENV VERSION_IMPORT_PATH ${version_import_path}
ENV VERSION ${version}
ENV GIT_COMMIT ${commit}
ENV GIT_DIRTY ${dirty}

RUN LDFLAGS='-s -w -linkmode external -extldflags "-static"' && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}Version=${VERSION}'" && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}BuildDate=$(date -u -R)'" && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}GitCommit=${GIT_COMMIT}'" && \
  LDFLAGS="$LDFLAGS -X '${VERSION_IMPORT_PATH}GitState=${GIT_DIRTY}'" && \
  go build -o /stage/usr/bin/datamon --ldflags "$LDFLAGS" ./cmd/datamon
RUN upx /stage/usr/bin/datamon
RUN md5sum /stage/usr/bin/datamon

RUN go build -o /stage/usr/bin/migrate --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/backup2blobs
RUN upx /stage/usr/bin/migrate
RUN md5sum /stage/usr/bin/migrate

RUN go build -o /stage/usr/bin/datamon_metrics --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/metrics
RUN upx /stage/usr/bin/datamon_metrics
RUN md5sum /stage/usr/bin/datamon_metrics

####
# Build the dist image
FROM ubuntu:latest

RUN apt-get update && \
  apt-get install -y --no-install-recommends \
    git \
    zsh \
    less \
    watch \
    curl \
    tmux \
    bc \
    vim \
    mc \
    htop &&\
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


### BEGIN tini

# omitting gpg verification during development/demo
# RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -

ENV TINI_VERSION v0.18.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static-amd64 /tmp/tini-static-amd64
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static-amd64.asc /tmp/tini-static-amd64.asc

# omitting gpg verification during development/demo
# RUN for key in \
#       595E85A6B1B4779EA4DAAEC70B588DFF0527A9B7 \
#     ; do \
#       gpg --keyserver hkp://pgp.mit.edu:80 --recv-keys "$key" || \
#       gpg --keyserver hkp://ipv4.pool.sks-keyservers.net --recv-keys "$key" || \
#       gpg --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys "$key" ; \
#     done
# RUN gpg --verify /tmp/tini-static-amd64.asc

RUN install -m 0755 /tmp/tini-static-amd64 /bin/tini

### END tini

RUN mkdir /datamon
COPY --from=base /datamon /datamon

COPY --from=base /stage /
ENV ZONEINFO /zoneinfo.zip

ADD ./hack/fuse-demo/datamon.yaml /root/.datamon/datamon.yaml
ADD ./hack/datamover/datamover.sh /usr/bin/datamover
ADD ./hack/datamover/datamover_metrics.sh /usr/bin/datamover_metrics
ADD ./hack/datamover/backup.sh /usr/bin/backup

RUN chmod a+x /usr/bin/datamover
RUN chmod a+x /usr/bin/datamover_metrics
RUN chmod a+x /usr/bin/backup

# USER root
# USER developer

RUN useradd -u 1020 -ms /bin/bash developer
RUN groupadd -g 2000 developers
RUN usermod -g developers developer
RUN chown -R developer:developers /usr/bin/datamon

USER developer
RUN touch ~/.zshrc

RUN cp /usr/bin/datamover /home/developer/datamover.sh && \
  chmod +x /home/developer/datamover.sh
RUN cp /usr/bin/datamover_metrics /home/developer/datamover_metrics.sh && \
  chmod +x /home/developer/datamover_metrics.sh
RUN cp /usr/bin/backup /home/developer/backup.sh && \
  chmod +x /home/developer/backup.sh

ENTRYPOINT [ "datamon" ]


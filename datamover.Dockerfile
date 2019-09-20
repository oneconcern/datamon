FROM datamon-binaries as base

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


FROM datamon-binaries as base

# dist-alike during development/debug
RUN cp /stage/usr/bin/datamon /usr/bin/datamon
ADD ./hack/fuse-demo/datamon.yaml /root/.datamon/datamon.yaml

# Build the dist image
FROM ubuntu:18.10
RUN apt-get update && apt-get install -y --no-install-recommends \
    fuse \
    sudo \
    vim \
    zsh \
    &&\
  apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN echo "allow_root" >> /etc/fuse.conf

COPY --from=base /stage /
ENV ZONEINFO /zoneinfo.zip

ADD ./hack/fuse-demo/datamon.yaml /root/.datamon/datamon.yaml

ADD hack/fuse-demo/wrap_datamon.sh .
ADD hack/fuse-demo/wrap_application.sh .

# USER root
RUN chmod a+x wrap_datamon.sh
# USER developer

RUN useradd -u 1020 -ms /bin/bash developer
RUN groupadd -g 2000 developers
RUN usermod -g developers developer
RUN chown -R developer:developers /usr/bin/datamon

RUN mkdir -p /etc/sudoers.d &&\
  echo "developer ALL = (ALL) NOPASSWD: ALL" > /etc/sudoers.d/developer &&\
  chmod 0400 /etc/sudoers.d/developer

USER developer
ENTRYPOINT [ "datamon" ]

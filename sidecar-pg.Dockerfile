FROM datamon-binaries as base

# dist-alike during development/debug
RUN cp /stage/usr/bin/datamon /usr/bin/datamon
ADD ./hack/fuse-demo/datamon.yaml /root/.datamon/datamon.yaml

# Build the dist image
# ??? do not need fuse and sudo in case of bundle download instead of mount
# ??? do not need in case of bundle download instead of mount
FROM postgres:12.0
RUN apt-get update --quiet && apt-get install -y --quiet --no-install-recommends fuse sudo vim zsh curl ca-certificates gnupg netcat netcat-openbsd &&\
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* &&\
    echo "allow_root" >> /etc/fuse.conf &&\
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -

## BEGIN tini

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

## END tini

COPY --from=base /stage /
ENV ZONEINFO /zoneinfo.zip

ADD hack/fuse-demo/datamon.yaml /root/.datamon/datamon.yaml
ADD hack/fuse-demo/wrap_datamon_pg.sh .
ADD hack/fuse-demo/wrap_application.sh .

# ??? do not need in case of bundle download instead of mount
#   or if not using docker-desktop kubernetes distro
RUN chmod a+x wrap_datamon_pg.sh &&\
    useradd -u 1020 -ms /bin/bash developer &&\
    groupadd -g 2000 developers &&\
    usermod -g developers developer &&\
    chown -R developer:developers /usr/bin/datamon &&\
    mkdir -p /etc/sudoers.d &&\
    echo "developer ALL = (ALL) NOPASSWD: ALL" > /etc/sudoers.d/developer &&\
    chmod 0400 /etc/sudoers.d/developer

USER developer

RUN touch ~/.zshrc

ENTRYPOINT [ "datamon" ]

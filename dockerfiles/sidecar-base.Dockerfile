# A base image to build the fuse sidecar (debian based).
#
# This image is updated every week by CI
FROM debian

# NOTE: an equivalent of tini is now available natively with docker, but cannot be used by kubernetes
ENV TINI_VERSION v0.18.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static-amd64 /tmp/tini-static-amd64
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static-amd64.asc /tmp/tini-static-amd64.asc
RUN install -m 0755 /tmp/tini-static-amd64 /bin/tini

RUN apt-get update &&\
    apt-get install -y --quiet --no-install-recommends \
        ca-certificates tzdata git lsb-release curl zsh zip gnupg fuse sudo &&\
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

WORKDIR /usr/share/zoneinfo
ENV ZONEINFO /zoneinfo.zip
RUN zip -qr -0 ${ZONEINFO} .

RUN useradd -u 1020 -ms /bin/bash developer &&\
    groupadd -g 2000 developers &&\
    usermod -g developers developer &&\
    chown -R developer:developers /bin/tini &&\
    echo "allow_root" >> /etc/fuse.conf &&\
    mkdir -p /etc/sudoers.d && \
    echo "developer ALL = (ALL) NOPASSWD: ALL" > /etc/sudoers.d/developer &&\
    chmod 0400 /etc/sudoers.d/developer

WORKDIR /

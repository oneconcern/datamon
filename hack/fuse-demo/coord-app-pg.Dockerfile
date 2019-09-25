FROM debian

RUN apt-get update &&\
    curl -sL https://deb.nodesource.com/setup_10.x  | bash &&\
    apt-get install -y \
        curl \
        postgresql \
        ca-certificates \
        gnupg \
        zsh \
        vim \
        &&\
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

RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -

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

RUN useradd -u 1020 -ms /bin/bash developer
RUN groupadd -g 2000 developers
RUN usermod -g developers developer
RUN chown -R developer:developers /bin/tini

ADD hack/fuse-demo/mock_application_pg.sh .
RUN chmod a+x mock_application_pg.sh

USER developer

RUN touch ~/.zshrc

ENTRYPOINT [ "/tmp/coord/.scripts/wrap_application.sh"]
CMD [ "./mock_application.sh"]

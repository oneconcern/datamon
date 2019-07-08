FROM gcr.io/onec-co/datamon-fuse-sidecar:latest

USER root

RUN apt-get update &&\
    curl -sL https://deb.nodesource.com/setup_10.x  | bash &&\
    apt-get install -y \
        zsh \
        golang-go \
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

# allow pprof `list` command
ADD . /datamon

USER developer

RUN touch ~/.zshrc

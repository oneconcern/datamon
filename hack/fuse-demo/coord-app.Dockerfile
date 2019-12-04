FROM debian

RUN apt-get update &&\
    curl -sL https://deb.nodesource.com/setup_10.x  | bash &&\
    apt-get install -y \
        curl \
        ca-certificates \
        zsh \
        vim \
        gnupg &&\
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

RUN useradd -u 1020 -ms /bin/bash developer
RUN groupadd -g 2000 developers
RUN usermod -g developers developer

ADD hack/fuse-demo/mock_application.sh .
RUN chmod a+x mock_application.sh

USER developer

RUN touch ~/.zshrc
RUN touch ~/.zshenv
RUN touch ~/.zprofile

ENTRYPOINT [ "/tmp/coord/.scripts/wrap_application.sh"]
CMD [ "./mock_application.sh"]

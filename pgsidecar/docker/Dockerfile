FROM postgres:11 as base

ARG github_user
ARG github_token

ENV GITHUB_USER ${github_user}
ENV GITHUB_TOKEN ${github_token}

RUN apt-get update \
  && apt-get install -y wget \
  && apt-get install -y postgresql-11-postgis-scripts

RUN set -e && \
  wget https://github.com/oneconcern/datamon/releases/download/v0.7/datamon.linux.tgz &&\
  tar -zxvf datamon.linux.tgz &&\
  mv datamon /usr/local/bin/

RUN apt-get autoremove -yqq &&\
  apt-get clean -y &&\
  apt-get autoclean -yqq &&\
  rm -rf  /tmp/* /var/tmp/* /var/lib/apt/lists/* /usr/share/doc/* /usr/share/locale/* /var/cache/debconf/*-old

COPY initDB.sh /docker-entrypoint-initdb.d/
COPY teardownDB.sh /

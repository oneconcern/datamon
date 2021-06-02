# base image to build the postgres sidecar
#
# This image is updated every week by CI
ARG VERSION=20200307
FROM gcr.io/onec-co/datamon-sidecar-base:${VERSION}

ENV SUDO=
RUN curl -sSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | ${SUDO} apt-key add -
RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ `lsb_release -cs`-pgdg main" |${SUDO} tee  /etc/apt/sources.list.d/pgdg.list
RUN apt-get update &&\
    apt-get install -y --quiet --no-install-recommends  postgresql-12 postgresql-client-12 &&\
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

RUN mkdir -p /var/run/postgresql && \
    chown -R developer:developers /var/run/postgresql && \
    chmod -R 775 /var/run/postgresql

ENV PATH /usr/lib/postgresql/12/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

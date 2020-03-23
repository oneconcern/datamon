# A postgres-enabled sidecar to pull and push postgres instances
# on storage managed by datamon.
ARG VERSION=20200307
FROM reg.onec.co/datamon-binaries as base
ARG VERSION
FROM gcr.io/onec-co/datamon-pgsidecar-base:$VERSION

COPY --from=base --chown=developer:developers /stage /

ENV ZONEINFO /zoneinfo.zip

ADD --chown=developer:developers hack/fuse-demo/datamon.yaml /home/developer/.datamon2/datamon.yaml
WORKDIR /usr/local/bin
ADD hack/fuse-demo/wrap_datamon_pg.sh .
RUN chmod a+x wrap_datamon_pg.sh

# TODO(fred): this is done in base
RUN mkdir -p /var/run/postgresql && \
    chown -R developer:developers /var/run/postgresql && \
    chmod -R 775 /var/run/postgresql

USER developer
ENV PATH /usr/lib/postgresql/12/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
RUN mkdir -p /tmp/db0 && \
    initdb -D /tmp/db0 && \
    (cd /tmp/db0 && find . -type d | tar cf ~/pgdirs.tar --no-recursion --files-from -) && \
    rm -rf /tmp/db0
ENTRYPOINT [ "wrap_datamon_pg.sh" ]

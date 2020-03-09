# datamover tool
ARG VERSION=20200307
FROM reg.onec.co/datamon-binaries as base
ARG VERSION
FROM gcr.io/onec-co/datamon-sidecar-base:$VERSION

COPY --from=base --chown=developer:developers /stage/usr/bin/datamon /usr/bin/datamon

ENV ZONEINFO /zoneinfo.zip

ADD ./hack/fuse-demo/datamon.yaml /root/.datamon/datamon.yaml
ADD ./hack/datamover/datamover.sh /usr/bin/datamover
ADD ./hack/datamover/datamover_metrics.sh /usr/bin/datamover_metrics
ADD ./hack/datamover/backup.sh /usr/bin/backup

RUN chmod a+x /usr/bin/datamover /usr/bin/datamover_metrics /usr/bin/backup

USER developer
RUN touch ~/.zshrc &&\
    for script in /usr/bin/datamover /usr/bin/datamover_metrics /usr/bin/backup ; do cp --preserve=mode "$script" /home/developer;done

ENTRYPOINT [ "datamon" ]


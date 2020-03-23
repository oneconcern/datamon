# A sidecar to pull and push bundles on storage managed by datamon.
ARG VERSION=20200307
FROM reg.onec.co/datamon-binaries as base
ARG VERSION
FROM gcr.io/onec-co/datamon-sidecar-base:$VERSION

COPY --from=base --chown=developer:developers /stage /

ADD --chown=developer:developers hack/fuse-demo/datamon.yaml /home/developer/.datamon2/datamon.yaml
WORKDIR /usr/local/bin
ADD hack/fuse-demo/wrap_datamon.sh .
RUN chmod a+x wrap_datamon.sh

USER developer

RUN touch ~/.zshrc
ENV PATH /usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

ENTRYPOINT [ "wrap_datamon.sh" ]

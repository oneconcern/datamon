FROM gcr.io/onec-co/datamon-fuse-sidecar:latest

ADD hack/fuse-demo/wrap_datamon.sh .
ADD hack/fuse-demo/wrap_application.sh .
ADD hack/fuse-demo/fuse-params.yaml /fuse-params.yaml

USER root
RUN chmod a+x wrap_datamon.sh
RUN chmod a+x wrap_application.sh
USER developer

# RUN mkdir /home/developer/.datamon
# ADD hack/fuse-demo/datamon.yaml /home/developer/.datamon/datamon.yaml

ENTRYPOINT ["./wrap_datamon.sh"]


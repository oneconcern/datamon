FROM gcr.io/onec-co/datamon-fuse-sidecar:latest

ADD hack/fuse-demo/wrap_datamon.sh .
ADD hack/fuse-demo/wrap_application.sh .

USER root
RUN chmod a+x wrap_datamon.sh
RUN chmod a+x wrap_application.sh

RUN apt-get update && apt-get install -y --no-install-recommends \
    vim \
    zsh \
    &&\
  apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

USER developer

RUN touch ~/.zshrc

# RUN mkdir /home/developer/.datamon
# ADD hack/fuse-demo/datamon.yaml /home/developer/.datamon/datamon.yaml

ENTRYPOINT ["./wrap_datamon.sh"]
CMD ["-c", "/tmp/coord", "-d", "bundle upload --path /tmp/upload --message \"result of container coordination demo\" --repo ransom-datamon-test-repo --label coordemo", "-d", "bundle mount --repo ransom-datamon-test-repo --label testlabel --mount /tmp/mount --stream"]

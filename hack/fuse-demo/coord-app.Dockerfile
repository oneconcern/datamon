ARG VERSION=20200307
FROM gcr.io/onec-co/datamon-sidecar-base:$VERSION

WORKDIR /apps
ADD hack/fuse-demo/mock_application.sh .
RUN chmod a+x mock_application.sh

USER developer
ENV PATH /apps:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
RUN touch ~/.zshrc ~/.zshenv ~/.zprofile

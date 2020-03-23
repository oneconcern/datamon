#
# A container to run a mock application consuming postgres data
#
ARG VERSION=20200307
#
# Use this base to get a complete debian with postgres 12 installed
FROM gcr.io/onec-co/datamon-pgsidecar-base:$VERSION

WORKDIR /apps
ADD hack/fuse-demo/mock_application_pg.sh .
RUN chmod a+x mock_application_pg.sh

USER developer
ENV PATH /apps:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
RUN touch ~/.zshrc

# This container may be used by applications to copy the wrap_application.sh
# script, which insulates apps from the SIGTERM signal.
#
FROM alpine
WORKDIR /.scripts
ADD hack/fuse-demo/wrap_application.sh .
RUN chmod a+x wrap_application.sh
ENV PATH /.scripts:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

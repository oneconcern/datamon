# A base container to build datamon and run CI jobs
#
# This image is updated every week on our CI
FROM cimg/go:1.14
#FROM circleci/golang
WORKDIR /tmp
USER root
RUN echo "deb http://packages.cloud.google.com/apt cloud-sdk main" | \
    tee -a /etc/apt/sources.list.d/google-cloud-sdk.list &&\
    curl -sSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add  &&\
    apt-get update -y -qq &&\
    apt-get install -y -qq git build-essential google-cloud-sdk shellcheck zsh

# install upx version that supports osx binaries (>=3.96)
ENV UPX_VERSION 3.96
ENV ARCH amd64_linux
RUN curl -sLL -O https://github.com/upx/upx/releases/download/v${UPX_VERSION}/upx-${UPX_VERSION}-${ARCH}.tar.xz &&\
    tar xf upx-${UPX_VERSION}-${ARCH}.tar.xz &&\
    install upx-${UPX_VERSION}-${ARCH}/upx /usr/bin &&\
    rm -rf upx-*
WORKDIR /go
USER circleci

# A base container to build datamon and run CI jobs
#
# This image is updated every week on our CI
FROM golang:1.17
WORKDIR /tmp
USER root
ENV SUDO=
RUN \
    ${SUDO} apt-get update -yqq &&\
    ${SUDO} apt-get install -yqq curl ca-certificates git lsb-release apt-transport-https jq &&\
    echo "deb http://packages.cloud.google.com/apt cloud-sdk main" | ${SUDO} tee -a /etc/apt/sources.list.d/google-cloud-sdk.list &&\
    curl -sSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | ${SUDO} apt-key add  &&\
    curl -sSL https://download.docker.com/linux/debian/gpg | ${SUDO} apt-key add - &&\
    echo "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | ${SUDO} tee -a /etc/apt/sources.list.d/docker.list &&\
    ${SUDO} apt-get update -y -qq &&\
    ${SUDO} apt-get install -y -qq build-essential google-cloud-sdk shellcheck zsh docker-ce docker-ce-cli containerd.io &&\
    go install gotest.tools/gotestsum@latest &&\
    go install github.com/mattn/goveralls@latest &&\
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# install upx version that supports osx binaries (>=3.96)
ENV UPX_VERSION 3.96
ENV ARCH amd64_linux
RUN curl -sLL -O https://github.com/upx/upx/releases/download/v${UPX_VERSION}/upx-${UPX_VERSION}-${ARCH}.tar.xz &&\
    tar xf upx-${UPX_VERSION}-${ARCH}.tar.xz &&\
    ${SUDO} install upx-${UPX_VERSION}-${ARCH}/upx /usr/bin &&\
    rm -rf upx-*
WORKDIR /go

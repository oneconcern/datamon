FROM circleci/golang

RUN sudo apt-get update -y
RUN sudo apt-get install lsb-release
RUN export CLOUD_SDK_REPO="cloud-sdk-$(lsb_release -c -s)" && echo $CLOUD_SDK_REPO
# RUN echo "deb http://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
RUN echo "deb http://packages.cloud.google.com/apt cloud-sdk-buster main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
RUN curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add
RUN sudo apt-get update -y
RUN sudo apt-get install -y git build-essential
RUN sudo apt-get install -y google-cloud-sdk
RUN sudo apt-get install -y shellcheck

# RUN sudo apt-get update -y
# RUN sudo apt-get install -y zsh vim

# ENV TINI_VERSION v0.18.0
# ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static-amd64 /tmp/tini-static-amd64
# ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static-amd64.asc /tmp/tini-static-amd64.asc

# USER root

# RUN install -m 0755 /tmp/tini-static-amd64 /bin/tini

ENTRYPOINT [ "/bin/tini", "--"]
CMD ["zsh"]

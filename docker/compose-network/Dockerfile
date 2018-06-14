FROM golang:1.10-stretch
LABEL maintainer="BOSCoin Developers <devteam@boscoin.io>"

ENV LC_ALL=en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV TERM=xterm
ENV DEBIAN_FRONTEND=noninteractive

WORKDIR /go/src/github.com/owlchain/sebak

RUN ( \
    apt-get update && \
    apt-get -y install openssh-client git locales vim && \
    apt-get -y autoremove; \
    apt-get clean; \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
    echo en_US.UTF-8 UTF-8 > /etc/locale.gen && locale-gen \
)

RUN go get github.com/golang/dep/cmd/dep

Add ./ /go/src/github.com/owlchain/sebak

Add ./docker/simple-3-nodes/sebak.crt /sebak.crt
Add ./docker/simple-3-nodes/sebak.key /sebak.key

COPY ./docker/compose-network/entrypoint.sh /entrypoint.sh

RUN ( \
    chmod 755 /entrypoint.sh; \
    mkdir -p /root/.ssh; \
    chmod 700 /root/.ssh; \
    echo 'StrictHostKeyChecking=no' >> ~/.ssh/config; \
    echo 'alias ls="ls --color=auto"' > ~/.bashrc; \
    echo 'alias v="ls -al"' >> ~/.bashrc; \
    echo 'alias a="ls -a"' >> ~/.bashrc; \
)

Add ./docker/compose-network/readonly /root/.ssh/id_rsa
Add ./docker/compose-network/readonly.pub /root/.ssh/id_rsa.pub

ENTRYPOINT ./entrypoint.sh

# RUN dep ensure

# vim: set filetype=Dockerfile:

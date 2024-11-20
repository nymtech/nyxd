FROM --platform=linux/amd64 ubuntu:24.04

ARG UID=1001
ARG GID=1001
ARG NYM_CLI_GIT_TAG=master
ARG WASMD_VERSION=v0.43.0
ARG RETAIN_BLOCKS

####################################
# Necessary env variables:
# CHAIN_ID
# DENOM
# STAKE_DENOM
# VALIDATOR_ENDPOINT
# PEERS
####################################

RUN echo "Acquire::http::Pipeline-Depth 0;" > /etc/apt/apt.conf.d/99custom && \
  echo "Acquire::http::No-Cache true;" >> /etc/apt/apt.conf.d/99custom && \
  echo "Acquire::BrokenProxy    true;" >> /etc/apt/apt.conf.d/99custom

RUN apt update && apt upgrade -y \
  && apt -y install ca-certificates jq curl vim wget

# Set up a nym user
RUN groupadd -g "${GID}" nym \
  && useradd --create-home --no-log-init -u "${UID}" -g "${GID}" nym

WORKDIR /home/nym

# Download nyxd, libwasmvm and nym-cli from our Github releases
RUN wget https://builds.ci.nymte.ch/${NYM_CLI_GIT_TAG}/nym-cli
RUN wget https://github.com/nymtech/nyxd/releases/download/${WASMD_VERSION}/nyxd-ubuntu-22.04.tar.gz
RUN tar -zxvf nyxd-ubuntu-22.04.tar.gz

RUN chmod u+x nyxd
RUN chmod u+x nym-cli
RUN chmod u+x libwasmvm.x86_64.so

# libwasmvm.so needs to be in this directory to be automatically recognized when running nyxd. Alternative is to update env variables.
RUN mv libwasmvm*.so /lib/x86_64-linux-gnu/

# Ubuntu 22.04 uses libssl3 instead of 1.1 that nym-cli is built with. This is a quick fix for that so that we don't need to recompile it.
RUN wget http://nz2.archive.ubuntu.com/ubuntu/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_amd64.deb
RUN ls | grep *.deb
RUN dpkg -i libssl1.1_1.1.1f-1ubuntu2_amd64.deb

RUN mkdir ".nyxd"
RUN mkdir output
RUN chown -R nym:nym ./

VOLUME /home/nym/.nyxd
VOLUME /home/nym/output

COPY start.sh .
RUN chmod +x *.sh

USER nym

ENTRYPOINT ["./start.sh"]

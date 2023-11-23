# FIXME after release  --------------------------------------------------------
FROM --platform=linux/amd64 ubuntu:22.04
ARG RELEASE_URL=https://github.com/nymtech/nyxd/releases/download/v0.43.0/nyxd-ubuntu-22.04.tar.gz
ARG ARCHIVE_NAME=nyxd-ubuntu-22.04.tar.gz

RUN apt update \
        && apt -y install ca-certificates jq curl vim wget

WORKDIR /opt

RUN wget ${RELEASE_URL}
RUN tar -zxvf ${ARCHIVE_NAME}

RUN chmod u+x nyxd
RUN chmod u+x libwasmvm.x86_64.so
RUN mv nyxd /usr/bin/nyxd
RUN mv libwasmvm*.so /lib/x86_64-linux-gnu/

COPY docker/* /opt/
RUN chmod +x /opt/*.sh

WORKDIR /opt

# rest server
EXPOSE 1317
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

CMD ["/usr/bin/nyxd", "version"]

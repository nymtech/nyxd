# --------------------------------------------------------
FROM --platform=linux/amd64 ubuntu:22.04
ARG arch=x86_64

RUN apt update \
        && apt -y install ca-certificates jq curl vim wget

WORKDIR /opt

RUN wget https://github.com/nymtech/nyxd/releases/download/v0.31.1/nyxd-ubuntu-22.04.tar.gz
RUN tar -zxvf nyxd-ubuntu-22.04.tar.gz

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

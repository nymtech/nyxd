#!/usr/bin/env bash

set -e

export LD_LIBRARY_PATH="$LD_LIBRARY_PATH":/home/nym
APP_NAME=nyxd
VALIDATOR_DATA_DIRECTORY="/home/nym/.${APP_NAME}"

# initialise the validator
./${APP_NAME} init "${CHAIN_ID}" --chain-id "${CHAIN_ID}" --default-denom ${STAKE_DENOM}

sleep 2

echo "changing params"
sed -i 's/minimum-gas-prices = "0stake"/minimum-gas-prices = "0.025'${STAKE_DENOM}',0.025'${DENOM}'"/' "${VALIDATOR_DATA_DIRECTORY}/config/app.toml"
sed -i '0,/enable = false/s//enable = true/' "${VALIDATOR_DATA_DIRECTORY}/config/app.toml"

# Network requests
sed -i 's/cors_allowed_origins = \[\]/cors_allowed_origins = \["*"\]/' "${VALIDATOR_DATA_DIRECTORY}/config/config.toml"
sed -i 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/' "${VALIDATOR_DATA_DIRECTORY}/config/config.toml"
sed -i 's/laddr = "tcp:\/\/127.0.0.1:26656"/laddr = "tcp:\/\/0.0.0.0:26656"/' "${VALIDATOR_DATA_DIRECTORY}/config/config.toml"
sed -i 's#external_address = ""#external_address = "'${RPC_FQDN}':26656"#' "${VALIDATOR_DATA_DIRECTORY}/config/config.toml"
sed -i 's/address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/' "${VALIDATOR_DATA_DIRECTORY}/config/app.toml"

# Set pruning settings
sed -i 's/pruning = "default"/pruning = "custom"/' "${HOME}/.nyxd/config/config.toml"
sed -i 's/pruning-keep-recent = "0"/pruning-keep-recent = "750000"/' "${HOME}/.nyxd/config/config.toml"
sed -i 's/pruning-interval = "0"/pruning-interval = "100"/' "${HOME}/.nyxd/config/config.toml"

echo "params changed"

# statesync so we don't catch up from the start
SNAP_RPC="${VALIDATOR_ENDPOINT}:443"
LATEST_HEIGHT=$(curl -s $SNAP_RPC/block | jq -r .result.block.header.height)
BLOCK_HEIGHT=$((LATEST_HEIGHT - 2000))
TRUST_HASH=$(curl -s "$SNAP_RPC/block?height=$BLOCK_HEIGHT" | jq -r .result.block_id.hash)

sed -i "s|^persistent_peers =.*|persistent_peers = \"$PEERS\"|" "${HOME}/.nyxd/config/config.toml"
sed -i 's/^enable = false/enable = true/' "${HOME}/.nyxd/config/config.toml"
sed -i "s|^rpc_servers =.*|rpc_servers = \"$SNAP_RPC,$SNAP_RPC\"|" "${HOME}/.nyxd/config/config.toml"
sed -i "s|^trust_height =.*|trust_height = $BLOCK_HEIGHT|" "${HOME}/.nyxd/config/config.toml"
sed -i "s|^trust_hash =.*|trust_hash = \"$TRUST_HASH\"|" "${HOME}/.nyxd/config/config.toml"

./${APP_NAME} start

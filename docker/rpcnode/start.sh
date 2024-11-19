#!/usr/bin/env bash

export LD_LIBRARY_PATH="$LD_LIBRARY_PATH":/root
PASSPHRASE=passphrase
APP_NAME=nyxd
ADDRESSES_DIRECTORY="/root/output"
export PEERS=${PEERS}
RPC_ENDPOINT="${VALIDATOR_ENDPOINT}:443"
LATEST_HEIGHT="$(curl -s ${VALIDATOR_ENDPOINT}/block | jq -r .result.block.header.height)"
GENESIS_FILE=$VALIDATOR_ENDPOINT/genesis

INTERVAL=1000
BLOCK_HEIGHT=$(expr $LATEST_HEIGHT - $INTERVAL)
TRUST_HASH="$(curl -s \"${VALIDATOR_ENDPOINT}/block?height=$BLOCK_HEIGHT\" | jq -r .result.block_id.hash)"

# initialise the validator
echo "Initialising the validator with name $NAME"
./${APP_NAME} init ${NAME} --chain-id "${CHAIN_ID}" 2>/dev/null
echo "Initialised the validator, sleeping 3 seconds."
sleep 3

cd /root/.nyxd/config
rm -f genesis.json
echo "Removed existing genesis, now curling new endpoint: ${GENESIS_FILE}"
curl "${GENESIS_FILE}" | jq '.result.genesis' >genesis.json
echo "Fetched the new genesis"

cd $HOME
echo "Validating genesis file.."
./${APP_NAME} genesis validate-genesis
echo "Genesis validated."
#  create a new node_admin account and add it to keychain
yes "${PASSPHRASE}" | ./nyxd keys add nyxd_admin 2>&1 >/dev/null | tail -n 1 >${ADDRESSES_DIRECTORY}/node_admin_mnemonic

# edit config.toml and app.toml files

# only uncomment this if all blocks to be synced need to be verified; note that setting fast_sync to false will slow down the syncing process
# sed -i 's/fast_sync = true/fast_sync = false/' $HOME/.nyxd/config/config.toml

sed -i '/\[api\]/,/^\[/ s/enable = false/enable = true/' $HOME/.nyxd/config/app.toml
sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0.025unym,0.025unyx"/' $HOME/.nyxd/config/app.toml
sed -i 's/swagger = false/swagger = true/' $HOME/.nyxd/config/app.toml
sed -i 's/cors_allowed_origins = \["\*"\]/cors_allowed_origins = \[\]/' $HOME/.nyxd/config/app.toml
sed -i 's/create_empty_blocks = false/create_empty_blocks = true/' $HOME/.nyxd/config/app.toml
sed -i.bak -e "s/^persistent_peers *=.*/persistent_peers = \"$PEERS\"/" $HOME/.nyxd/config/config.toml

if [ "$SYNC_BLOCK" == "CUSTOM" ]; then
  sed -i "s/rpc_servers = \"\"/rpc_servers = \"${RPC_ENDPOINT},${RPC_ENDPOINT}\"|" $HOME/.nyxd/config/config.toml
  sed -i "s/trust_height = 0/trust_height = ${BLOCK_HEIGHT}/" $HOME/.nyxd/config/config.toml
  sed -i "s/trust_hash = \"\"/trust_hash = \"${TRUST_HASH}\"/" $HOME/.nyxd/config/config.toml
fi

echo "Starting nyxd.."
./${APP_NAME} start &
sleep 10

sleep infinity

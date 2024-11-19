#!/usr/bin/env bash

export LD_LIBRARY_PATH="$LD_LIBRARY_PATH":/root
PASSPHRASE=passphrase
ADDRESSES_DIRECTORY="/root/output"
MAIN_NODE_ADMIN_MNEMONIC=$(head -n1 "${ADDRESSES_DIRECTORY}/main_node_admin_mnemonic")

echo "$NAME"
sleep 10

# Add main validator to the chain
./nyxd keys add main_validator --recover << EOF
${MAIN_NODE_ADMIN_MNEMONIC}
$PASSPHRASE
$PASSPHRASE
EOF

# Get the address and nvaloper address of the new validator
yes "${PASSPHRASE}" | ./nyxd keys show nyxd_admin | grep -o 'address: .*' | awk '{print $2}' >nyxd-admin-address.txt
./nyxd debug addr "$(cat nyxd-admin-address.txt)" | grep 'Bech32 Val:' | awk '{print $3}' >nvaloper-address.txt

# send unym funds to the new validator address
echo "Sending unym to the validator"
yes "${PASSPHRASE}" | ./nyxd tx bank send main_validator "$(cat nyxd-admin-address.txt)" --chain-id "${CHAIN_ID}" 2500000000u"${DENOM}" --gas auto --gas-adjustment 1.5 --gas-prices 0.025u"${DENOM}" -y
sleep 5
echo "Successful"

# send unyx funds to the new validator address
echo "Sending nyx to the validator"
yes "${PASSPHRASE}" | ./nyxd tx bank send main_validator "$(cat nyxd-admin-address.txt)" --chain-id "${CHAIN_ID}" 2500000000u"${STAKE_DENOM}" --gas auto --gas-adjustment 1.5 --gas-prices 0.025u"${DENOM}" -y
sleep 5
echo "Successful"

# Prepare JSON file for validator creation
# Prepare JSON file for validator creation
VALIDATOR_JSON=$(cat <<EOF
{
  "pubkey": "$(./nyxd tendermint show-validator)",
  "amount": "1000000u${STAKE_DENOM}",
  "moniker": "$NAME",
  "commission-rate": "0.05",
  "commission-max-rate": "0.1",
  "commission-max-change-rate": "0.05",
  "min-self-delegation": "100000"
}
EOF
)

echo "$VALIDATOR_JSON" > validator.json


# create the validator
echo "Creating the validator"
yes "${PASSPHRASE}" | ./nyxd tx staking create-validator validator.json --from=nyxd_admin --chain-id "${CHAIN_ID}" --fees=5000u"${DENOM}" --node="${VALIDATOR_ENDPOINT}:443" -y
sleep 5
echo "Validator created"

# delegate stake
echo "Delegating stake and creating validator"
yes "${PASSPHRASE}" | ./nyxd tx staking delegate "$(cat nvaloper-address.txt)" 50000000u"${STAKE_DENOM}" --from=nyxd_admin --chain-id "${CHAIN_ID}" --fees=5000u"${DENOM}" -y
sleep 5
echo "Done!"

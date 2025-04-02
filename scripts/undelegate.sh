#!/bin/bash

KEY_NAME="your_key_name"

CHAIN_ID="nyx"

NODE_RPC="https://rpc.nymtech.net:443"


FEE="5000unyx"

# Delegation amount (in unyx) per validator.
DELEGATION_AMOUNT="6300000000000"

# Array of validator operator addresses.  Order MUST match the MONIKERS array.
VALOPERS=(
    "nvaloper1jryrapf54q7gza547t490qkdgsjh5ngmc4xtm7"
    "nvaloper1aqxz5rhsu59tz3fykskqprarq2c2vxqv2jaysv"
    "nvaloper15urq2dtp9qce4fyc85m6upwm9xul3049ckp6x4"
    "nvaloper14epla9mvgwl8456l4emvvchyj6reqg8a79fayw"
    "nvaloper12uspvr6qnrn9exf083t8l3khmd5n4r746n6ydf"
    "nvaloper1mlmqlkxjttj6m4wwmf5eur35n3hf5zm2ev2vjj"
    "nvaloper1xgyczxuxspeytdpyvp3w840ckzp4env4phq67x"
    "nvaloper12xrk9wxmh7z4n5s5d8hk7zumn0hqu6z69a3e9f"
    "nvaloper1nf7p9xlqw4jzzhslndtc40hac0g9askzrfplxg"
)

# Array of validator monikers. Order MUST match the VALOPERS array.
MONIKERS=(
  "Atalma"
  "Blockfend Genesis Labs"
  "Chorus One"
  "Commodum"
  "FairStaking"
  "Greenfield"
  "Nodes.Guru"
  "Polkachu.com"
  "Polychain"
)




delegate_tokens() {
  local valoper="$1"
  local amount="$2"
  local moniker="$3"

  

  echo "Undelegating ${amount}unyx from ${moniker} (${valoper})..."

  # Prompt for confirmation before broadcasting.
  read -r -p "Proceed to undelegate ${amount}unyx from ${moniker}? (y/N) " response
  case "$response" in
    [yY][eE][sS]|[yY])
      nyxd tx staking unbond "$valoper" "$amount"unyx \
        --from "$KEY_NAME" \
        --chain-id "$CHAIN_ID" \
        --node "$NODE_RPC" \
        --fees "$FEE"

      if [ $? -ne 0 ]; then
        echo "Error: Undelegation failed for ${moniker}."
        exit 1
      else
        echo "Undelegation from ${moniker} successful. Sleeping for 3 seconds..."
        sleep 3

        #
      fi
      ;;
    *)
      echo "Undelegation from ${moniker} skipped."
      ;;
  esac
}


# Loop through the validators and delegate tokens.
for i in "${!VALOPERS[@]}"; do
  delegate_tokens "${VALOPERS[$i]}" "$DELEGATION_AMOUNT" "${MONIKERS[$i]}"
done

echo "Undelegation script completed."

exit 0
#!/bin/bash

# Configuration
NODE_HOME="$HOME/.nyxd"
VALIDATOR_NAME="validator1"
VALIDATOR_HOME="$NODE_HOME/$VALIDATOR_NAME"
CHAIN_ID="testing-1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Create validator directory if it doesn't exist
if [ ! -d "$VALIDATOR_HOME" ]; then
    echo -e "${GREEN}Creating validator directory at $VALIDATOR_HOME${NC}"
    mkdir -p "$VALIDATOR_HOME/config"
fi

# Create or use existing priv_validator_key.json
if [ ! -f "$VALIDATOR_HOME/config/priv_validator_key.json" ]; then
    echo -e "${GREEN}Generating new validator key...${NC}"
    nyxd init "$VALIDATOR_NAME" --home "$VALIDATOR_HOME" >/dev/null 2>&1
    
    # Copy only the necessary files if we have a data directory
    if [ -d "$NODE_HOME/config" ]; then
        echo -e "${GREEN}Copying chain configuration...${NC}"
        cp "$NODE_HOME/config/genesis.json" "$VALIDATOR_HOME/config/" 2>/dev/null || true
        cp "$NODE_HOME/config/client.toml" "$VALIDATOR_HOME/config/" 2>/dev/null || true
        cp "$NODE_HOME/config/app.toml" "$VALIDATOR_HOME/config/" 2>/dev/null || true
        cp "$NODE_HOME/config/config.toml" "$VALIDATOR_HOME/config/" 2>/dev/null || true
    fi
fi

# Get the validator private key
echo -e "${GREEN}Extracting validator private key...${NC}"
PRIV_KEY=$(cat "$VALIDATOR_HOME/config/priv_validator_key.json" | base64)

# Create validator key if it doesn't exist
echo -e "${GREEN}Creating validator key...${NC}"
VALIDATOR_KEY=$(nyxd keys show "$VALIDATOR_NAME" --keyring-backend test --home "$VALIDATOR_HOME" --output json 2>/dev/null || \
    nyxd keys add "$VALIDATOR_NAME" --keyring-backend test --home "$VALIDATOR_HOME" --output json)

VALIDATOR_ADDRESS=$(echo "$VALIDATOR_KEY" | jq -r .address)
VALIDATOR_VALOPER=$(nyxd keys show "$VALIDATOR_NAME" --keyring-backend test --home "$VALIDATOR_HOME" --bech val -a)

echo -e "${GREEN}Validator Address:${NC} $VALIDATOR_ADDRESS"
echo -e "${GREEN}Validator Operator Address:${NC} $VALIDATOR_VALOPER"

# Create test accounts if they don't exist
echo -e "${GREEN}Creating test accounts...${NC}"
TEST_ACCOUNT_1=$(nyxd keys add test1 --keyring-backend test --home "$VALIDATOR_HOME" --output json 2>/dev/null || nyxd keys show test1 --keyring-backend test --home "$VALIDATOR_HOME" --output json)
TEST_ACCOUNT_2=$(nyxd keys add test2 --keyring-backend test --home "$VALIDATOR_HOME" --output json 2>/dev/null || nyxd keys show test2 --keyring-backend test --home "$VALIDATOR_HOME" --output json)

TEST_ADDR_1=$(echo "$TEST_ACCOUNT_1" | jq -r .address)
TEST_ADDR_2=$(echo "$TEST_ACCOUNT_2" | jq -r .address)

echo -e "${GREEN}Test Account 1:${NC} $TEST_ADDR_1"
echo -e "${GREEN}Test Account 2:${NC} $TEST_ADDR_2"

# Run in-place-testnet command
echo -e "${GREEN}Running in-place-testnet...${NC}"
nyxd in-place-testnet "$CHAIN_ID" "$VALIDATOR_VALOPER" \
    --validator-privkey="$PRIV_KEY" \
    --home "$VALIDATOR_HOME" \
    --accounts-to-fund="$TEST_ADDR_1,$TEST_ADDR_2"

echo -e "${GREEN}Testnet conversion complete!${NC}"
echo -e "${GREEN}To start the testnet, run:${NC}"
echo -e "nyxd start --home \"$VALIDATOR_HOME\""

# Save information for future use
cat << EOF > "$VALIDATOR_HOME/testnet_info.txt"
Chain ID: $CHAIN_ID
Validator Name: $VALIDATOR_NAME
Validator Address: $VALIDATOR_ADDRESS
Validator Operator Address: $VALIDATOR_VALOPER
Test Account 1: $TEST_ADDR_1
Test Account 2: $TEST_ADDR_2
Private Key (base64): $PRIV_KEY
EOF

echo -e "${GREEN}All information has been saved to:${NC} $VALIDATOR_HOME/testnet_info.txt"
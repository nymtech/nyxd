#!/bin/sh

CHAIN_ID=localnyx
NYX_HOME=$HOME/.nyxd
CONFIG_FOLDER=$NYX_HOME/config
MONIKER=val
STATE='false'

MNEMONIC="bottom loan skill merry east cradle onion journey palm apology verb edit desert impose absurd oil bubble sweet glove shallow size build burst effort"

while getopts s flag
do
    case "${flag}" in
        s) STATE='true';;
    esac
done

install_prerequisites () {
    apt update && apt install dasel
}

edit_genesis () {
    GENESIS=$CONFIG_FOLDER/genesis.json

    # Update staking module
    dasel put -t string -f $GENESIS '.app_state.staking.params.bond_denom' -v 'unyx'
    dasel put -t string -f $GENESIS '.app_state.staking.params.unbonding_time' -v '240s'

    # Update bank module
    dasel put -t string -f $GENESIS '.app_state.bank.denom_metadata.[].description' -v 'The native staking token of Nyx'
    dasel put -t string -f $GENESIS '.app_state.bank.denom_metadata.[0].denom_units.[].denom' -v 'unyx'
    dasel put -t string -f $GENESIS '.app_state.bank.denom_metadata.[0].denom_units.[0].exponent' -v 0
    dasel put -t string -f $GENESIS '.app_state.bank.denom_metadata.[0].base' -v 'unyx'
    dasel put -t string -f $GENESIS '.app_state.bank.denom_metadata.[0].display' -v 'nyx'
    dasel put -t string -f $GENESIS '.app_state.bank.denom_metadata.[0].name' -v 'nyx'
    dasel put -t string -f $GENESIS '.app_state.bank.denom_metadata.[0].symbol' -v 'NYX'

    # Update crisis module
    dasel put -t string -f $GENESIS '.app_state.crisis.constant_fee.denom' -v 'unyx'

    # Update gov module
    dasel put -t string -f $GENESIS '.app_state.gov.voting_params.voting_period' -v '60s'
    dasel put -t string -f $GENESIS '.app_state.gov.params.min_deposit.[0].denom' -v 'unyx'

    # Update mint module
    dasel put -t string -f $GENESIS '.app_state.mint.params.mint_denom' -v "unyx"

    # Update wasm permission (Nobody or Everybody)
    dasel put -t string -f $GENESIS '.app_state.wasm.params.code_upload_access.permission' -v "Everybody"
}



#
# Account 1 - n1yf8syypl2arj0p94d7kh4v4lqqlratfzdwje9p
# asthma earth machine solid promote ritual grant myself spin enroll beef scatter reunion buyer move side online have top anger add camp aware hungry


# Account 2 - n1du0dtepk5nepu52fnhnn3fjskypxyfq8avzkna
# unfold across relief pizza genre popular gallery calm pottery elegant soft avoid dinner best travel custom water engine claim screen siren crawl town coconut

# Account 3 - n1ay66l60qfckfwhf48fll6anp469yerw0n4rwt0
# general spice foam extra wonder bounce canvas child dinner twin jacket wrong amazing scale energy couch page cloud come opera basic company hurry car


add_genesis_accounts () {
    # Add genesis accounts with test tokens
    nyxd add-genesis-account n1yf8syypl2arj0p94d7kh4v4lqqlratfzdwje9p 100000000000unyx,100000000000unym --home $NYX_HOME
    nyxd add-genesis-account n1du0dtepk5nepu52fnhnn3fjskypxyfq8avzkna 100000000000unyx,100000000000unym --home $NYX_HOME
    nyxd add-genesis-account n1ay66l60qfckfwhf48fll6anp469yerw0n4rwt0 100000000000unyx,100000000000unym --home $NYX_HOME

    echo $MNEMONIC | nyxd keys add $MONIKER --recover --keyring-backend=test --home $NYX_HOME
    nyxd gentx $MONIKER 500000000unyx --keyring-backend=test --chain-id=$CHAIN_ID --home $NYX_HOME

    nyxd collect-gentxs --home $NYX_HOME
}

edit_config () {
    # Remove seeds
    dasel put -t string -f $CONFIG_FOLDER/config.toml '.p2p.seeds' -v ''

    # Expose the rpc
    dasel put -t string -f $CONFIG_FOLDER/config.toml '.rpc.laddr' -v "tcp://0.0.0.0:26657"

    # Expose pprof for debugging
    dasel put -t string -f $CONFIG_FOLDER/config.toml '.rpc.pprof_laddr' -v "0.0.0.0:6060"
}

enable_cors () {
    # Enable cors on RPC
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "*" '.rpc.cors_allowed_origins.[]'
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "Accept-Encoding" '.rpc.cors_allowed_headers.[]'
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "DELETE" '.rpc.cors_allowed_methods.[]'
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "OPTIONS" '.rpc.cors_allowed_methods.[]'
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "PATCH" '.rpc.cors_allowed_methods.[]'
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "PUT" '.rpc.cors_allowed_methods.[]'

    # Enable unsafe cors and swagger on the api
    dasel put -t bool -f $CONFIG_FOLDER/app.toml -v "true" '.api.swagger'
    dasel put -t bool -f $CONFIG_FOLDER/app.toml -v "true" '.api.enabled-unsafe-cors'

    # Enable cors on gRPC Web
    dasel put -t bool -f $CONFIG_FOLDER/app.toml -v "true" '.grpc-web.enable-unsafe-cors'
}

if [[ ! -d $CONFIG_FOLDER ]]
then
    echo $MNEMONIC | nyxd init -o --chain-id=$CHAIN_ID --home $NYX_HOME --recover $MONIKER
    install_prerequisites
    edit_genesis
    add_genesis_accounts
    edit_config
    enable_cors
fi

nyxd start --home $NYX_HOME --x-crisis-skip-assert-invariants & 

wait 
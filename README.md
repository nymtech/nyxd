# Nyx Zone

This repository hosts `nyxd`, a Cosmos zone with wasm smart contracts enabled. In the future, `nyxd`'s smart contracts will include extensions making it easy to use Coconut credentials. 

This code was forked from the `cosmwasm/wasmd` repository, which was itself forked from `cosmos/gaiad` as a basis by the CosmWasm project. They then added `x/wasm` and cleaned up 
many gaia-specific files. However, the `nyxd` binary should function just like `gaiad` except for the
addition of the `x/wasm` module. 

**Note**: Requires [Go 1.18+](https://golang.org/dl/)

As this is essentially a no-modifications fork of `wasmd`, security issues are best handled upstream. For critical security issues & disclosure, see their [SECURITY.md](https://github.com/CosmWasm/wasmd/blob/main/SECURITY.md).


## Supported Systems

The supported systems are limited by the dlls created in [`wasmvm`](https://github.com/CosmWasm/wasmvm). In particular, **we only support MacOS and Linux**.
However, **M1 macs are not fully supported.** (Experimental support was merged with wasmd 0.24)
For linux, the default is to build for glibc, and we cross-compile with CentOS 7 to provide
backwards compatibility for `glibc 2.12+`. This includes all known supported distributions
using glibc (CentOS 7 uses 2.12, obsolete Debian Jessy uses 2.19). 

As of `0.9.0` we support `muslc` Linux systems, in particular **Alpine linux**,
which is popular in docker distributions. Note that we do **not** store the
static `muslc` build in the repo, so you must compile this yourself, and pass `-tags muslc`.
Please look at the [`Dockerfile`](./Dockerfile) for an example of how we build a static Go
binary for `muslc`. (Or just use this Dockerfile for your production setup).


## Stability

**This is beta software** It is run in some production systems, but we cannot yet provide a stability guarantee
and have not yet gone through and audit of this codebase. Note that the
[CosmWasm smart contract framework](https://github.com/CosmWasm/cosmwasm) used by `wasmd` is in a 1.0 release candidate
as of March 2022, with stability guarantee and addressing audit results.

The APIs are pretty stable, but we cannot guarantee their stability until we reach v1.0.
However, we will provide a way for you to hard-fork your way to v1.0.

Thank you to all projects who have run this code in your mainnets and testnets and
given feedback to improve stability.

## Encoding
The used cosmos-sdk version is in transition migrating from amino encoding to protobuf for state. So are we now.

We use standard cosmos-sdk encoding (amino) for all sdk Messages. However, the message body sent to all contracts, 
as well as the internal state is encoded using JSON. Cosmwasm allows arbitrary bytes with the contract itself 
responsible for decodng. For better UX, we often use `json.RawMessage` to contain these bytes, which enforces that it is
valid json, but also give a much more readable interface.  If you want to use another encoding in the contracts, that is
a relatively minor change to wasmd but would currently require a fork. Please open in issue if this is important for 
your use case.

## Quick Start

See the Nym [validator docs](https://nymtech.net/docs/stable/run-nym-nodes/nodes/validators) for setup instructions.

## Runtime flags

We provide a number of variables in `app/app.go` that are intended to be set via `-ldflags -X ...`
compile-time flags. This enables us to avoid copying a new binary directory over for each small change
to the configuration.

Available flags:
 
* `-X github.com/CosmWasm/wasmd/app.NodeDir=.corald` - set the config/data directory for the node (default `~/.wasmd`)
* `-X github.com/CosmWasm/wasmd/app.Bech32Prefix=coral` - set the bech32 prefix for all accounts (default `wasm`)
* `-X github.com/CosmWasm/wasmd/app.ProposalsEnabled=true` - enable all x/wasm governance proposals (default `false`)
* `-X github.com/CosmWasm/wasmd/app.EnableSpecificProposals=MigrateContract,UpdateAdmin,ClearAdmin` - 
    enable a subset of the x/wasm governance proposal types (overrides `ProposalsEnabled`)

Examples:

* [`wasmd`](./Makefile#L50-L55) is a generic, permissionless version using the `cosmos` bech32 prefix

## Genesis Configuration
We strongly suggest **to limit the max block gas in the genesis** and not use the default value (`-1` for infinite).
```json
  "consensus_params": {
    "block": {
      "max_gas": "SET_YOUR_MAX_VALUE",  
```

Tip: if you want to lock this down to a permisisoned network, the following script can edit the genesis file
to only allow permissioned use of code upload or instantiating. (Make sure you set `app.ProposalsEnabled=true`
in this binary):

`sed -i 's/permission": "Everybody"/permission": "Nobody"/'  .../config/genesis.json`

## Contributors

Much thanks to all who have contributed to this project, from this app, to the `cosmwasm` framework, to example contracts and documentation.
Or even testing the app and bringing up critical issues. The following have helped bring this project to life:

* Ethan Frey [ethanfrey](https://github.com/ethanfrey)
* Simon Warta [webmaster128](https://github.com/webmaster128)
* Alex Peters [alpe](https://github.com/alpe)
* Aaron Craelius [aaronc](https://github.com/aaronc)
* Sunny Aggarwal [sunnya97](https://github.com/sunnya97)
* Cory Levinson [clevinson](https://github.com/clevinson)
* Sahith Narahari [sahith-narahari](https://github.com/sahith-narahari)
* Jehan Tremback [jtremback](https://github.com/jtremback)
* Shane Vitarana [shanev](https://github.com/shanev)
* Billy Rennekamp [okwme](https://github.com/okwme)
* Westaking [westaking](https://github.com/westaking)
* Marko [marbar3778](https://github.com/marbar3778)
* JayB [kogisin](https://github.com/kogisin)
* Rick Dudley [AFDudley](https://github.com/AFDudley)
* KamiD [KamiD](https://github.com/KamiD)
* Valery Litvin [litvintech](https://github.com/litvintech)
* Leonardo Bragagnolo [bragaz](https://github.com/bragaz)

Sorry if I forgot you from this list, just contact me or add yourself in a PR :)

# Nyx Zone

This repository hosts `nyxd`, a Cosmos zone with wasm smart contracts enabled. In the future, `nyxd`'s smart contracts will include extensions making it easy to use Coconut credentials.

This code was forked from the `cosmwasm/wasmd` repository, which was itself forked from `cosmos/gaiad` as a basis by the CosmWasm project. They then added `x/wasm` and cleaned up
many gaia-specific files. However, the `nyxd` binary should function just like `gaiad` except for the addition of the `x/wasm` module.

**Note**: Requires [Go 1.20.10](https://golang.org/dl/)

**Note**: The mainnet binary in the releases is built with `Go 1.20.10` so if you plan on building from source please use that version!

⚠️ Using other versions of Go may result in non-determinism (app-hash issues)

As this is essentially a no-modifications fork of `wasmd`, security issues are best handled upstream. For critical security issues & disclosure, see the `CosmWasm/wasmd` [SECURITY.md](https://github.com/CosmWasm/wasmd/blob/main/SECURITY.md).


## Quick Start

Please refer to [validator operators guide](https://nymtech.net/operators/nodes/validator-setup.html) for setup instructions.


## Further Reading

For extended information, please refer to the [README.md](https://github.com/CosmWasm/wasmd/blob/main/README.md) of the parent `CosmWasm/wasmd` repository.
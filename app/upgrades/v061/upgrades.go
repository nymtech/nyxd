package v061

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/nymtech/nyxd/app/upgrades"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.61"

// PinnedContracts are the busiest mainnet contracts. The upgrade pins the code
// each one runs at the upgrade height, which cuts 58,000 gas from every
// transaction that calls it. Looked up by address so that a migration before
// the upgrade is picked up.
var PinnedContracts = []string{
	"n17srjznxl9dvzdkpwpw24gg668wc73val88a6m5ajg6ankwvz9wtst0cznr", // mixnet
	"n1f53zk7txj5v37s588dyxcav3wd3y6eruymlpuy6dw3nym794c9gql3txua", // directory
	"n1r7s6aksyc6pqardx88k3rkgfagwvj4z4zum9mmz2sfk3zm2mha0sd4dnun", // ecash
	"n1fjj4xngrw85dgzt336ehj72p4mc9q20ymhqsu5egtx755mf83gwsrccq7k", // geolocation
}

// PinOperator gets authz grants from the gov module for MsgPinCodes and
// MsgUnpinCodes only, so it can change the pinned set without a proposal.
// Governance stays the wasm authority and can revoke the grants with MsgRevoke.
const PinOperator = "n1txe3x04c3wjejfn4my4ef3n4t4dfhfajcwqg9t" // Nym admin multisig, 5-of-12

// Upgrade moves nyxd to wasmd v0.61.16 (cosmos-sdk v0.53.6, wasmvm v3,
// ibc-go v10.5) and hyperlane-cosmos v1.1.0. No store is added or removed and
// no module changes its consensus version. On chains that hold the mainnet
// contracts it also pins them and hands pinning to PinOperator.
var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{},
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		versionMap, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return nil, err
		}
		if err := ConfigurePinning(sdk.UnwrapSDKContext(ctx), ak, PinnedContracts, PinOperator); err != nil {
			return nil, err
		}
		return versionMap, nil
	}
}

// ConfigurePinning pins the code of each contract in contracts and grants
// operator MsgPinCodes and MsgUnpinCodes on behalf of the gov module.
// Contracts missing from the state are skipped; when none exist (any chain
// other than mainnet or a fork of it) nothing is pinned and nothing is granted.
func ConfigurePinning(ctx sdk.Context, ak *upgrades.AppKeepers, contracts []string, operator string) error {
	logger := ctx.Logger().With("upgrade", UpgradeName)
	pinner := wasmkeeper.NewGovPermissionKeeper(ak.WasmKeeper)

	pinned := 0
	for _, contract := range contracts {
		addr, err := decodeAddress(contract)
		if err != nil {
			return err
		}
		info := ak.WasmKeeper.GetContractInfo(ctx, addr)
		if info == nil {
			logger.Info("contract not found, not pinning", "contract", contract)
			continue
		}
		if err := pinner.PinCode(ctx, info.CodeID); err != nil {
			return fmt.Errorf("pin code %d of %s: %w", info.CodeID, contract, err)
		}
		logger.Info("pinned contract code", "contract", contract, "code_id", info.CodeID)
		pinned++
	}
	if pinned == 0 {
		logger.Info("no known contracts on this chain, skipping the pin operator grants")
		return nil
	}

	grantee, err := decodeAddress(operator)
	if err != nil {
		return err
	}
	gov := authtypes.NewModuleAddress(govtypes.ModuleName)
	for _, msg := range []sdk.Msg{&wasmtypes.MsgPinCodes{}, &wasmtypes.MsgUnpinCodes{}} {
		auth := authz.NewGenericAuthorization(sdk.MsgTypeURL(msg))
		if err := ak.AuthzKeeper.SaveGrant(ctx, grantee, gov, auth, nil); err != nil {
			return fmt.Errorf("grant %s to %s: %w", auth.Msg, operator, err)
		}
		logger.Info("granted pin operator", "grantee", operator, "msg", auth.Msg)
	}
	return nil
}

// decodeAddress accepts any bech32 prefix, so tests running with the SDK's
// default prefix can use the same code path as mainnet.
func decodeAddress(addr string) (sdk.AccAddress, error) {
	_, bz, err := bech32.DecodeAndConvert(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", addr, err)
	}
	return bz, nil
}

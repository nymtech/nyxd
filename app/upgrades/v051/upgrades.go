package v050

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	circuittypes "cosmossdk.io/x/circuit/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/nymtech/nyxd/app/upgrades"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.51"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: storetypes.StoreUpgrades{
		Added: []string{
			circuittypes.ModuleName,
		},
		Deleted: []string{},
	},
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
) upgradetypes.UpgradeHandler {
	// sdk 47 to sdk 50
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		logger := sdkCtx.Logger().With("upgrade", UpgradeName)

		logger.Info(fmt.Sprintf("== Starting %s Upgrade... ==", UpgradeName))
		logger.Info(fmt.Sprintf("Pre-upgrade map: %v", fromVM))
		logger.Info("== Running any configured module migrations == ")
		newVersionMap, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return nil, err
		}
		logger.Info("== Module migrations complete == ")
		logger.Info(fmt.Sprintf("Post-upgrade map: %v", newVersionMap))
		logger.Info("== Upgrade complete! == ")

		// TODO : Consider asserting invariants

		return newVersionMap, err

	}
}

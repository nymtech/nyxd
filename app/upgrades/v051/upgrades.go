package v051

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

		logger.Info(fmt.Sprintf("Starting %s upgrade", UpgradeName))

		// Run all configured migrations within the modules
		logger.Info("Running any configured module migrations")
		newVersionMap, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return nil, err
		}
		logger.Info("Module migrations complete. Printing summary : ")

		// Generate summary
		for moduleName, oldVersion := range fromVM {
			if newVersion, ok := newVersionMap[moduleName]; ok {
				if oldVersion != newVersion {
					logger.Info(fmt.Sprintf("Module %s migrated from version %d to version %d", moduleName, oldVersion, newVersion))
				}
			} else {
				logger.Info(fmt.Sprintf("Module %s was removed during the upgrade", moduleName))
			}
		}

		for moduleName, newVersion := range newVersionMap {
			if _, ok := fromVM[moduleName]; !ok {
				logger.Info(fmt.Sprintf("Module %s was added during the upgrade with version %d", moduleName, newVersion))
			}
		}

		// Check if we're good after the upgrade
		logger.Info("Asserting invariants post-upgrade")
		ak.CrisisKeeper.AssertInvariants(sdkCtx)

		logger.Info("Upgrade complete!")

		return newVersionMap, err

	}
}

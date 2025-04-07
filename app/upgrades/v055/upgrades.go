package v055

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	"github.com/nymtech/nyxd/app/upgrades"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.55"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: storetypes.StoreUpgrades{
		Added: []string{},
		Deleted: []string{
			"intertx",
			icatypes.ModuleName,
			icacontrollertypes.StoreKey,
			icahosttypes.StoreKey,
		},
	},
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
) upgradetypes.UpgradeHandler {
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

		// Generate summary
		logger.Info("==== Migrations summary start =====")
		for moduleName, oldVersion := range fromVM {
			if newVersion, ok := newVersionMap[moduleName]; ok {
				if oldVersion != newVersion {
					logger.Info(fmt.Sprintf("Module %s migrated from version %d to version %d", moduleName, oldVersion, newVersion))
				}
			} else {
				logger.Info(fmt.Sprintf("Module %s was removed during the upgrade", moduleName))
			}
		}
		logger.Info("==== Migrations summary end =====")
		// Check if we're good after the upgrade
		logger.Info(" === Asserting invariants post-upgrade === ")
		ak.CrisisKeeper.AssertInvariants(sdkCtx)

		logger.Info("Upgrade complete! 🎉")

		return newVersionMap, err
	}
}

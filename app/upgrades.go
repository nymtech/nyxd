package app

import (
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"

	"github.com/nymtech/nyxd/app/upgrades"
	"github.com/nymtech/nyxd/app/upgrades/noop"
	v061 "github.com/nymtech/nyxd/app/upgrades/v061"
)

// Upgrades list of chain upgrades
var Upgrades = []upgrades.Upgrade{v061.Upgrade}

// RegisterUpgradeHandlers registers the chain upgrade handlers
func (app *WasmApp) RegisterUpgradeHandlers(appOpts servertypes.AppOptions) {
	if len(Upgrades) == 0 {
		// always have a unique upgrade registered for the current version to test in system tests
		Upgrades = append(Upgrades, noop.NewUpgrade(app.Version()))
	}

	keepers := upgrades.AppKeepers{
		AccountKeeper:         &app.AccountKeeper,
		ConsensusParamsKeeper: &app.ConsensusParamsKeeper,
		IBCKeeper:             app.IBCKeeper,
		Codec:                 app.appCodec,
		GetStoreKey:           app.GetKey,
	}
	app.GetStoreKeys()
	// register all upgrade handlers
	for _, upgrade := range Upgrades {
		app.UpgradeKeeper.SetUpgradeHandler(
			upgrade.UpgradeName,
			upgrade.CreateUpgradeHandler(
				app.ModuleManager,
				app.configurator,
				&keepers,
			),
		)
	}

	// in-place-testnet --trigger-testnet-upgrade runs the plan on the first testnet
	// block, so its store upgrades apply on load instead of via upgrade-info.json
	if isTestnet, _ := appOpts.Get(server.KeyIsTestnet).(bool); isTestnet {
		if name, _ := appOpts.Get(server.KeyTriggerTestnetUpgrade).(string); name != "" {
			for _, upgrade := range Upgrades {
				if upgrade.UpgradeName == name {
					app.SetStoreLoader(testnetStoreLoader(&upgrade.StoreUpgrades))
					return
				}
			}
		}
	}

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	// register store loader for current upgrade
	for _, upgrade := range Upgrades {
		if upgradeInfo.Name == upgrade.UpgradeName {
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &upgrade.StoreUpgrades))
			break
		}
	}
}

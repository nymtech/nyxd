package app

import (
	"time"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctmmigrations "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint/migrations"

	"github.com/cosmos/cosmos-sdk/baseapp"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/group"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/nft"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

const UpgradeName = "v0.43.0"

func (app WasmApp) RegisterUpgradeHandlers() {
	// Set param key table for params module migration
	for _, subspace := range app.ParamsKeeper.GetSubspaces() {
		subspace := subspace

		var keyTable paramstypes.KeyTable
		switch subspace.Name() {
		case authtypes.ModuleName:
			keyTable = authtypes.ParamKeyTable() //nolint:staticcheck
		case banktypes.ModuleName:
			keyTable = banktypes.ParamKeyTable() //nolint:staticcheck
		case stakingtypes.ModuleName:
			keyTable = stakingtypes.ParamKeyTable()
		case minttypes.ModuleName:
			keyTable = minttypes.ParamKeyTable() //nolint:staticcheck
		case distrtypes.ModuleName:
			keyTable = distrtypes.ParamKeyTable() //nolint:staticcheck
		case slashingtypes.ModuleName:
			keyTable = slashingtypes.ParamKeyTable() //nolint:staticcheck
		case govtypes.ModuleName:
			keyTable = govv1.ParamKeyTable() //nolint:staticcheck
		case crisistypes.ModuleName:
			keyTable = crisistypes.ParamKeyTable() //nolint:staticcheck
			// ibc types
		case ibctransfertypes.ModuleName:
			keyTable = ibctransfertypes.ParamKeyTable()
		case icahosttypes.SubModuleName:
			keyTable = icahosttypes.ParamKeyTable()
		case icacontrollertypes.SubModuleName:
			keyTable = icacontrollertypes.ParamKeyTable()
			// wasm
		case wasmtypes.ModuleName:
			keyTable = wasmtypes.ParamKeyTable() //nolint:staticcheck
		default:
			continue
		}

		if !subspace.HasKeyTable() {
			subspace.WithKeyTable(keyTable)
		}
	}

	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeName,
		func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			ctx.Logger().Info("== Starting in-place migration steps == ")

			// IBC v4-v5 -- nothing
			// IBC v5-v6 -- no relevant upgrades as we do not use ICS27 custom auth. modules
			// IBC v6-v7 -- Prunes expired consensus states
			ctx.Logger().Info("== IBC Upgrade => Pruning Consensus States")
			_, err := ibctmmigrations.PruneExpiredConsensusStates(ctx, app.AppCodec(), app.IBCKeeper.ClientKeeper)
			if err != nil {
				return nil, err
			}
			// IBC v7-v7.1 -- allow localhost client to IBC client params
			ctx.Logger().Info("== IBC Upgrade => allow 09-localhost")
			params := app.IBCKeeper.ClientKeeper.GetParams(ctx)
			params.AllowedClients = append(params.AllowedClients, exported.Localhost)
			app.IBCKeeper.ClientKeeper.SetParams(ctx, params)

			// Migrate Tendermint consensus parameters from x/params module to a dedicated x/consensus module
			ctx.Logger().Info("== x/params migration => Migrating to module owned params")
			baseAppLegacySS := app.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())
			baseapp.MigrateParams(ctx, baseAppLegacySS, &app.ConsensusParamsKeeper)

			migrations, err := app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
			if err != nil {
				return nil, err
			}

			// Set the voting time parameter
			ctx.Logger().Info("== x/gov param update => Set new parameters and initial deposit ")
			govParams := app.GovKeeper.GetParams(ctx)

			// Burns deposit if it doesn't enter voting period
			govParams.BurnProposalDepositPrevote = true
			// Burns deposit if proposal doesn't reach quorum
			govParams.BurnVoteQuorum = false
			// Burns deposit if NWV outcome
			govParams.BurnVoteVeto = true
			// Set the voting period
			votingPeriod := time.Hour * 24 * 7 // 7 days
			govParams.VotingPeriod = &votingPeriod
			// Set an initial deposit ratio to prevent proposal spam
			govParams.MinInitialDepositRatio = sdk.NewDecWithPrec(25, 2).String()

			err = app.GovKeeper.SetParams(ctx, govParams)
			if err != nil {
				return nil, err
			}

			return migrations, nil
		},
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == UpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{
				consensustypes.ModuleName,
				crisistypes.ModuleName,
				nft.ModuleName,
				group.ModuleName,
			},
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

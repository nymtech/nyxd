package v051

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	circuittypes "cosmossdk.io/x/circuit/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/nymtech/nyxd/app/upgrades"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.54-doubledip"

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

func CompleteAccountVesting(ctx context.Context, ak *upgrades.AppKeepers, address string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger()

	// Get the account
	accountAddr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	account := ak.AccountKeeper.GetAccount(sdkCtx, accountAddr)
	if account == nil {
		return fmt.Errorf("account not found: %s", address)
	}

	// Check if it's a vesting account
	vestingAcc, ok := account.(*vestingtypes.ContinuousVestingAccount)
	if !ok {
		return fmt.Errorf("account is not a vesting account: %s", address)
	}

	// Get the original vesting coins for logging
	originalVesting := vestingAcc.GetOriginalVesting()

	// Get the base account and set it directly
	// This removes all vesting schedules and makes all tokens available
	baseAcc := vestingAcc.BaseAccount
	ak.AccountKeeper.SetAccount(sdkCtx, baseAcc)

	logger.Info("Successfully removed vesting and released all tokens",
		"address", address,
		"released_amount", originalVesting.String())

	return nil
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
		logger.Info("==== Migrations summary start =====")
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

		logger.Info("==== Migrations summary end =====")

		// Set governance module parameters
		logger.Info("==== Setting governance parameters =====")
		govParams, err := ak.GovKeeper.Params.Get(sdkCtx)

		if err != nil {
			panic("Failed to get governance parameters")
		}

		// 5 times the deposit of regular proposals for expedited proposals
		depositDenom, depositAmount := govParams.MinDeposit[0].Denom, govParams.MinDeposit[0].Amount.Mul(sdkmath.NewInt(5))
		govParams.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewInt64Coin(depositDenom, depositAmount.Int64()))
		govParams.ProposalCancelRatio = sdkmath.LegacyZeroDec().String()

		ok := ak.GovKeeper.Params.Set(ctx, govParams)

		if ok != nil {
			panic("Failed to set governance parameters")
		}

		// Complete vesting on an abandoned validator account
		if err := CompleteAccountVesting(ctx, ak, "n1kah7954n5tcdkgyc0pgfqm8afgeld8s992zj95"); err != nil {
			return nil, fmt.Errorf("failed to complete account vesting: %w", err)
		}

		// Check if we're good after the upgrade
		logger.Info(" === Asserting invariants post-upgrade === ")
		ak.CrisisKeeper.AssertInvariants(sdkCtx)

		logger.Info("Upgrade complete! 🎉")

		return newVersionMap, err
	}
}

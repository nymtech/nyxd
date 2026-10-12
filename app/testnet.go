package app

import (
	"errors"
	"fmt"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TestnetOptions describe how InitAppForTestnet turns a copy of a live chain's
// state into a single-validator testnet.
type TestnetOptions struct {
	// ValidatorPubKey is the consensus key of the node running the fork, i.e. the
	// key in its priv_validator_key.json. CometBFT's validator set is rewritten to
	// this key, so the staking state must use the same one.
	ValidatorPubKey cryptotypes.PubKey
	// OperatorAddress becomes the operator of the new validator; its account
	// self-delegates ValidatorStake.
	OperatorAddress sdk.ValAddress
	// ValidatorStake is minted and self-delegated, in the bond denom.
	ValidatorStake math.Int
	// AccountsToFund each receive FundAmount of the bond denom.
	AccountsToFund []sdk.AccAddress
	FundAmount     math.Int
	// UpgradeToTrigger, if set, schedules that upgrade for the first testnet block.
	UpgradeToTrigger string
	// VotingPeriod and ExpeditedVotingPeriod replace the gov params when non-zero.
	VotingPeriod          time.Duration
	ExpeditedVotingPeriod time.Duration
}

// InitAppForTestnet rewrites the state loaded in app so that the node holding
// opts.ValidatorPubKey is the only bonded validator. Everything else (accounts,
// contracts, hyperlane, IBC, mint and distribution parameters) is left as it was.
//
// Every existing validator is jailed and moved to Unbonded, with its bonded tokens
// moved to the not-bonded pool, so delegations, rewards and the staking pool
// balances stay consistent. The new validator is bonded through a regular
// self-delegation. The writes land in the stores' working trees and are committed
// with the first block of the testnet.
func InitAppForTestnet(app *WasmApp, opts TestnetOptions) error {
	if opts.ValidatorPubKey == nil {
		return errors.New("validator public key is required")
	}
	if opts.OperatorAddress.Empty() {
		return errors.New("operator address is required")
	}
	if opts.ValidatorStake.IsNil() || !opts.ValidatorStake.IsPositive() {
		return errors.New("validator stake must be positive")
	}
	if opts.UpgradeToTrigger != "" && !app.UpgradeKeeper.HasHandler(opts.UpgradeToTrigger) {
		return fmt.Errorf("no upgrade handler registered for %q", opts.UpgradeToTrigger)
	}

	height := app.LastBlockHeight()
	ctx := app.NewUncachedContext(false, cmtproto.Header{Height: height})

	bondDenom, err := app.StakingKeeper.BondDenom(ctx)
	if err != nil {
		return fmt.Errorf("bond denom: %w", err)
	}

	if err := retireValidators(ctx, app, bondDenom); err != nil {
		return fmt.Errorf("retire existing validators: %w", err)
	}
	if err := createTestnetValidator(ctx, app, opts, bondDenom, height); err != nil {
		return fmt.Errorf("create testnet validator: %w", err)
	}

	if !opts.FundAmount.IsNil() && opts.FundAmount.IsPositive() {
		for _, addr := range opts.AccountsToFund {
			if err := mintTo(ctx, app, addr, sdk.NewCoins(sdk.NewCoin(bondDenom, opts.FundAmount))); err != nil {
				return fmt.Errorf("fund %s: %w", addr, err)
			}
		}
	}

	if opts.VotingPeriod > 0 || opts.ExpeditedVotingPeriod > 0 {
		params, err := app.GovKeeper.Params.Get(ctx)
		if err != nil {
			return fmt.Errorf("gov params: %w", err)
		}
		if opts.VotingPeriod > 0 {
			params.VotingPeriod = &opts.VotingPeriod
		}
		if opts.ExpeditedVotingPeriod > 0 {
			params.ExpeditedVotingPeriod = &opts.ExpeditedVotingPeriod
		}
		if err := params.ValidateBasic(); err != nil {
			return fmt.Errorf("gov params: %w", err)
		}
		if err := app.GovKeeper.Params.Set(ctx, params); err != nil {
			return fmt.Errorf("gov params: %w", err)
		}
	}

	if opts.UpgradeToTrigger != "" {
		plan := upgradetypes.Plan{Name: opts.UpgradeToTrigger, Height: height + 1}
		if err := app.UpgradeKeeper.ScheduleUpgrade(ctx, plan); err != nil {
			return fmt.Errorf("schedule upgrade %q: %w", opts.UpgradeToTrigger, err)
		}
	}
	return nil
}

// retireValidators jails every validator and moves it to Unbonded, the state the
// staking module would reach on its own once the validator had fully unbonded.
func retireValidators(ctx sdk.Context, app *WasmApp, bondDenom string) error {
	sk := app.StakingKeeper
	validators, err := sk.GetAllValidators(ctx)
	if err != nil {
		return err
	}

	bondedTokens := math.ZeroInt()
	for _, val := range validators {
		valAddr, err := sk.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			return err
		}
		if val.IsBonded() {
			bondedTokens = bondedTokens.Add(val.Tokens)
		}
		if err := sk.DeleteValidatorByPowerIndex(ctx, val); err != nil {
			return err
		}
		if err := sk.DeleteLastValidatorPower(ctx, valAddr); err != nil {
			return err
		}
		val.Jailed = true
		val.Status = stakingtypes.Unbonded
		if err := sk.SetValidator(ctx, val); err != nil {
			return err
		}
	}

	// Validators that were unbonding are Unbonded now; leaving them queued would
	// make the staking EndBlocker panic on their status.
	store := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	queue := storetypes.KVStorePrefixIterator(store, stakingtypes.ValidatorQueueKey)
	var queued [][]byte
	for ; queue.Valid(); queue.Next() {
		queued = append(queued, queue.Key())
	}
	if err := queue.Close(); err != nil {
		return err
	}
	for _, key := range queued {
		store.Delete(key)
	}

	if err := sk.SetLastTotalPower(ctx, math.ZeroInt()); err != nil {
		return err
	}
	if bondedTokens.IsPositive() {
		coins := sdk.NewCoins(sdk.NewCoin(bondDenom, bondedTokens))
		if err := app.BankKeeper.SendCoinsFromModuleToModule(ctx, stakingtypes.BondedPoolName, stakingtypes.NotBondedPoolName, coins); err != nil {
			return err
		}
	}
	return nil
}

func createTestnetValidator(ctx sdk.Context, app *WasmApp, opts TestnetOptions, bondDenom string, height int64) error {
	sk := app.StakingKeeper
	consAddr := sdk.ConsAddress(opts.ValidatorPubKey.Address())

	if _, err := sk.GetValidator(ctx, opts.OperatorAddress); err == nil {
		return fmt.Errorf("operator %s is already a validator, use a fresh operator key", opts.OperatorAddress)
	} else if !errors.Is(err, stakingtypes.ErrNoValidatorFound) {
		return err
	}
	if _, err := sk.GetValidatorByConsAddr(ctx, consAddr); err == nil {
		return fmt.Errorf("consensus key %s already belongs to a validator, use a fresh priv_validator_key.json", consAddr)
	} else if !errors.Is(err, stakingtypes.ErrNoValidatorFound) {
		return err
	}

	val, err := stakingtypes.NewValidator(opts.OperatorAddress.String(), opts.ValidatorPubKey, stakingtypes.Description{Moniker: "testnet-validator"})
	if err != nil {
		return err
	}
	val.Status = stakingtypes.Bonded
	val.MinSelfDelegation = math.OneInt()
	val.Commission = stakingtypes.NewCommissionWithTime(
		math.LegacyNewDecWithPrec(5, 2), math.LegacyNewDecWithPrec(10, 2), math.LegacyNewDecWithPrec(5, 2), ctx.BlockTime(),
	)
	if err := sk.SetValidator(ctx, val); err != nil {
		return err
	}
	if err := sk.SetValidatorByConsAddr(ctx, val); err != nil {
		return err
	}
	// initialises the distribution records and registers the pubkey with slashing
	if err := sk.Hooks().AfterValidatorCreated(ctx, opts.OperatorAddress); err != nil {
		return err
	}

	operatorAcc := sdk.AccAddress(opts.OperatorAddress)
	if err := mintTo(ctx, app, operatorAcc, sdk.NewCoins(sdk.NewCoin(bondDenom, opts.ValidatorStake))); err != nil {
		return err
	}
	// moves the stake into the bonded pool, sets the power index and runs the
	// distribution delegation hooks
	if _, err := sk.Delegate(ctx, operatorAcc, opts.ValidatorStake, stakingtypes.Unbonded, val, true); err != nil {
		return err
	}
	// CometBFT was handed a placeholder power; a last power of 0 makes the first
	// EndBlock send the real one.
	if err := sk.SetLastValidatorPower(ctx, opts.OperatorAddress, 0); err != nil {
		return err
	}

	signingInfo := slashingtypes.NewValidatorSigningInfo(consAddr, height, 0, time.Unix(0, 0), false, 0)
	return app.SlashingKeeper.SetValidatorSigningInfo(ctx, consAddr, signingInfo)
}

func mintTo(ctx sdk.Context, app *WasmApp, addr sdk.AccAddress, coins sdk.Coins) error {
	if err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins); err != nil {
		return err
	}
	return app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, addr, coins)
}

// testnetStoreLoader is used when in-place-testnet triggers an upgrade: the plan
// runs on the first testnet block, so its store additions/removals must be
// applied when the stores are loaded, without an upgrade-info.json.
func testnetStoreLoader(upgrades *storetypes.StoreUpgrades) func(storetypes.CommitMultiStore) error {
	return func(ms storetypes.CommitMultiStore) error {
		return ms.LoadLatestVersionAndUpgrade(upgrades)
	}
}

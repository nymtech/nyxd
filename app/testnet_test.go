package app

import (
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	v061 "github.com/nymtech/nyxd/app/upgrades/v061"
)

func TestInitAppForTestnet(t *testing.T) {
	wasmApp, delegator := setupForkSourceApp(t)
	ctx := blockCtx(wasmApp)

	mintParamsBefore, err := wasmApp.MintKeeper.Params.Get(ctx)
	require.NoError(t, err)
	versionMapBefore, err := wasmApp.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	supplyBefore := wasmApp.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom).Amount

	consKey := ed25519.GenPrivKey().PubKey()
	operator := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
	funded := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	stake := math.NewInt(5_000_000_000_000)
	opts := TestnetOptions{
		ValidatorPubKey:       consKey,
		OperatorAddress:       operator,
		ValidatorStake:        stake,
		AccountsToFund:        []sdk.AccAddress{funded},
		FundAmount:            math.NewInt(1_000_000),
		UpgradeToTrigger:      v061.UpgradeName,
		VotingPeriod:          time.Minute,
		ExpeditedVotingPeriod: 30 * time.Second,
	}
	forkHeight := wasmApp.LastBlockHeight()
	require.NoError(t, InitAppForTestnet(wasmApp, opts))

	// first testnet block: the triggered upgrade runs and CometBFT gets the
	// new validator's real power
	res := nextBlock(t, wasmApp)
	require.Len(t, res.ValidatorUpdates, 1)
	cmtKey, err := cryptocodec.ToCmtProtoPublicKey(consKey)
	require.NoError(t, err)
	require.Equal(t, cmtKey, res.ValidatorUpdates[0].PubKey)
	require.Equal(t, sdk.TokensToConsensusPower(stake, sdk.DefaultPowerReduction), res.ValidatorUpdates[0].Power)
	nextBlock(t, wasmApp)
	nextBlock(t, wasmApp)

	ctx = blockCtx(wasmApp)
	doneHeight, err := wasmApp.UpgradeKeeper.GetDoneHeight(ctx, v061.UpgradeName)
	require.NoError(t, err)
	require.Equal(t, forkHeight+1, doneHeight)
	versionMapAfter, err := wasmApp.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, versionMapBefore, versionMapAfter, "the upgrade must not init or migrate any module")

	// only the new validator is bonded; the old ones are jailed and unbonded
	sk := wasmApp.StakingKeeper
	validators, err := sk.GetAllValidators(ctx)
	require.NoError(t, err)
	require.Len(t, validators, 4)
	for _, val := range validators {
		if val.GetOperator() == operator.String() {
			require.True(t, val.IsBonded())
			require.False(t, val.IsJailed())
			require.Equal(t, stake, val.Tokens)
			continue
		}
		require.True(t, val.IsJailed(), val.GetOperator())
		require.True(t, val.IsUnbonded(), val.GetOperator())
	}
	lastPower, err := sk.GetLastTotalPower(ctx)
	require.NoError(t, err)
	require.Equal(t, sdk.TokensToConsensusPower(stake, sdk.DefaultPowerReduction), lastPower.Int64())

	requireStakingConsistent(t, wasmApp, ctx)

	// supply grew by exactly what was minted, and only bond-denom balances moved
	supplyAfter := wasmApp.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom).Amount
	require.True(t, supplyAfter.GTE(supplyBefore.Add(stake).Add(opts.FundAmount)))
	require.Equal(t, opts.FundAmount, wasmApp.BankKeeper.GetBalance(ctx, funded, sdk.DefaultBondDenom).Amount)

	// delegators of retired validators keep working accounts
	cacheCtx, _ := ctx.CacheContext()
	_, err = wasmApp.DistrKeeper.WithdrawDelegationRewards(cacheCtx, delegator, sdk.ValAddress(validators[0].GetOperator()))
	if err != nil {
		// validators[0] might be one the delegator never delegated to; find theirs
		dels, derr := sk.GetDelegatorDelegations(ctx, delegator, 10)
		require.NoError(t, derr)
		require.NotEmpty(t, dels)
		valAddr, verr := sdk.ValAddressFromBech32(dels[0].ValidatorAddress)
		require.NoError(t, verr)
		_, err = wasmApp.DistrKeeper.WithdrawDelegationRewards(cacheCtx, delegator, valAddr)
	}
	require.NoError(t, err)

	// the validator's operator can unbond part of its stake on the testnet
	_, _, err = sk.Undelegate(cacheCtx, sdk.AccAddress(operator), operator, math.LegacyNewDec(1_000_000))
	require.NoError(t, err)

	mintParamsAfter, err := wasmApp.MintKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, mintParamsBefore, mintParamsAfter, "mint params must survive the fork")
	govParams, err := wasmApp.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, time.Minute, *govParams.VotingPeriod)
	require.Equal(t, 30*time.Second, *govParams.ExpeditedVotingPeriod)
}

func TestInitAppForTestnetRejectsUnknownUpgrade(t *testing.T) {
	wasmApp, _ := setupForkSourceApp(t)
	err := InitAppForTestnet(wasmApp, TestnetOptions{
		ValidatorPubKey:  ed25519.GenPrivKey().PubKey(),
		OperatorAddress:  sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()),
		ValidatorStake:   math.NewInt(1_000_000),
		UpgradeToTrigger: "v9.99",
	})
	require.ErrorContains(t, err, `no upgrade handler registered for "v9.99"`)
}

// setupForkSourceApp builds a chain with three validators, one of them jailed
// and unbonding, plus a delegator with a bonded and an unbonding delegation.
func setupForkSourceApp(t *testing.T) (*WasmApp, sdk.AccAddress) {
	t.Helper()
	var vals []*cmttypes.Validator
	for i := 0; i < 3; i++ {
		pk, err := cryptocodec.ToCmtPubKeyInterface(ed25519.GenPrivKey().PubKey())
		require.NoError(t, err)
		vals = append(vals, cmttypes.NewValidator(pk, 1))
	}
	valSet := cmttypes.NewValidatorSet(vals)

	accKeys := []*secp256k1.PrivKey{secp256k1.GenPrivKey(), secp256k1.GenPrivKey()}
	var genAccs []authtypes.GenesisAccount
	var balances []banktypes.Balance
	for _, k := range accKeys {
		acc := authtypes.NewBaseAccount(k.PubKey().Address().Bytes(), k.PubKey(), 0, 0)
		genAccs = append(genAccs, acc)
		balances = append(balances, banktypes.Balance{
			Address: acc.GetAddress().String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100_000_000_000))),
		})
	}
	wasmApp := SetupWithGenesisValSet(t, valSet, genAccs, "testing", nil, balances...)
	_, err := wasmApp.Commit()
	require.NoError(t, err)

	// jail one validator so the next EndBlock starts unbonding it
	ctx := blockCtx(wasmApp)
	require.NoError(t, wasmApp.StakingKeeper.Jail(ctx, sdk.ConsAddress(vals[2].Address)))

	delegator := sdk.AccAddress(accKeys[1].PubKey().Address())
	target := sdk.ValAddress(vals[0].Address)
	val, err := wasmApp.StakingKeeper.GetValidator(ctx, target)
	require.NoError(t, err)
	shares, err := wasmApp.StakingKeeper.Delegate(ctx, delegator, math.NewInt(10_000_000), stakingtypes.Unbonded, val, true)
	require.NoError(t, err)
	_, _, err = wasmApp.StakingKeeper.Undelegate(ctx, delegator, target, shares.QuoInt64(3))
	require.NoError(t, err)

	nextBlock(t, wasmApp)
	nextBlock(t, wasmApp)

	ctx = blockCtx(wasmApp)
	unbonding := 0
	validators, err := wasmApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	for _, v := range validators {
		if v.IsUnbonding() {
			unbonding++
		}
	}
	require.Equal(t, 1, unbonding, "fixture needs one unbonding validator")
	return wasmApp, delegator
}

func blockCtx(wasmApp *WasmApp) sdk.Context {
	return wasmApp.NewUncachedContext(false, cmtproto.Header{
		Height: wasmApp.LastBlockHeight(),
		Time:   blockTime(wasmApp.LastBlockHeight()),
	})
}

func blockTime(height int64) time.Time {
	return time.Unix(1_700_000_000, 0).Add(time.Duration(height) * 5 * time.Second).UTC()
}

func nextBlock(t *testing.T, wasmApp *WasmApp) *abci.ResponseFinalizeBlock {
	t.Helper()
	height := wasmApp.LastBlockHeight() + 1
	res, err := wasmApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: blockTime(height)})
	require.NoError(t, err)
	_, err = wasmApp.Commit()
	require.NoError(t, err)
	return res
}

// requireStakingConsistent checks the invariants the removed x/crisis module used
// to assert for staking pools and delegator shares.
func requireStakingConsistent(t *testing.T, wasmApp *WasmApp, ctx sdk.Context) {
	t.Helper()
	sk := wasmApp.StakingKeeper
	bondDenom, err := sk.BondDenom(ctx)
	require.NoError(t, err)

	bonded, notBonded := math.ZeroInt(), math.ZeroInt()
	validators, err := sk.GetAllValidators(ctx)
	require.NoError(t, err)
	for _, val := range validators {
		if val.IsBonded() {
			bonded = bonded.Add(val.Tokens)
		} else {
			notBonded = notBonded.Add(val.Tokens)
		}

		valAddr, err := sdk.ValAddressFromBech32(val.GetOperator())
		require.NoError(t, err)
		dels, err := sk.GetValidatorDelegations(ctx, valAddr)
		require.NoError(t, err)
		shares := math.LegacyZeroDec()
		for _, d := range dels {
			shares = shares.Add(d.Shares)
		}
		require.Equal(t, val.DelegatorShares, shares, "delegator shares of %s", val.GetOperator())
	}
	ubds, err := sk.GetAllUnbondingDelegations(ctx, nil)
	if err != nil || ubds == nil {
		require.NoError(t, sk.IterateUnbondingDelegations(ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) bool {
			ubds = append(ubds, ubd)
			return false
		}))
	}
	for _, ubd := range ubds {
		for _, e := range ubd.Entries {
			notBonded = notBonded.Add(e.Balance)
		}
	}

	bondedPool := wasmApp.AccountKeeper.GetModuleAddress(stakingtypes.BondedPoolName)
	notBondedPool := wasmApp.AccountKeeper.GetModuleAddress(stakingtypes.NotBondedPoolName)
	require.Equal(t, bonded, wasmApp.BankKeeper.GetBalance(ctx, bondedPool, bondDenom).Amount, "bonded pool")
	require.Equal(t, notBonded, wasmApp.BankKeeper.GetBalance(ctx, notBondedPool, bondDenom).Amount, "not-bonded pool")

	// total supply equals the sum of all balances
	sum := math.ZeroInt()
	wasmApp.BankKeeper.IterateAllBalances(ctx, func(_ sdk.AccAddress, c sdk.Coin) bool {
		if c.Denom == bondDenom {
			sum = sum.Add(c.Amount)
		}
		return false
	})
	require.Equal(t, wasmApp.BankKeeper.GetSupply(ctx, bondDenom).Amount, sum, "total supply")

	// distribution can still account for every validator
	for _, val := range validators {
		valAddr, err := sdk.ValAddressFromBech32(val.GetOperator())
		require.NoError(t, err)
		outstanding, err := wasmApp.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, valAddr)
		require.NoError(t, err)
		require.False(t, outstanding.IsAnyNegative())
	}
}

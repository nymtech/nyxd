package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	cmtcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/nymtech/nyxd/app"
)

const (
	flagAccountsToFund        = "accounts-to-fund"
	flagFundAmount            = "fund-amount"
	flagValidatorStake        = "validator-stake"
	flagVotingPeriod          = "voting-period"
	flagExpeditedVotingPeriod = "expedited-voting-period"
)

func NewInPlaceTestnetCmd(addStartFlags servertypes.ModuleInitFlags) *cobra.Command {
	cmd := server.InPlaceTestnetCreator(newTestnetApp)
	addStartFlags(cmd)
	cmd.Short = "Turn a copy of a chain's data dir into a single-validator testnet and start it"
	cmd.Long = `Turns the state in --home into a testnet with one validator and starts the node.

The node's own consensus key (config/priv_validator_key.json) becomes the only
bonded validator, operated by [newOperatorAddress]. Every existing validator is
jailed and unbonded; accounts, contracts, hyperlane, IBC and module parameters are
kept as they are.

Run it on a copy of the data dir of a node that was stopped cleanly, and use a
fresh priv_validator_key.json. The first block can take a while to commit. Wait
for it before stopping the node; afterwards restart it with the normal start
command. Running in-place-testnet twice on the same data dir is not supported.

--trigger-testnet-upgrade runs the named upgrade handler on the first block, which
is how to rehearse an upgrade against real state. It is required when the binary
is one upgrade ahead of the copied state and that state has applied an upgrade
before (mainnet: v0.60): the binary only registers the newest handler, and the
upgrade module refuses to start when the last applied plan has no handler.`

	cmd.Example = `nyxd in-place-testnet nyx-fork-1 nvaloper1... --home ./fork --skip-confirmation \
  --accounts-to-fund="n1aaa...,n1bbb..." --voting-period=60s --expedited-voting-period=30s \
  --trigger-testnet-upgrade=v0.61`

	cmd.Flags().String(flagAccountsToFund, "", "Comma-separated account addresses to fund")
	cmd.Flags().String(flagFundAmount, "1000000000000", "Amount of the bond denom minted to each account in --accounts-to-fund")
	cmd.Flags().String(flagValidatorStake, "100000000000000", "Amount of the bond denom minted and self-delegated to the new validator")
	cmd.Flags().Duration(flagVotingPeriod, 0, "Replace the gov voting period (0 keeps it)")
	cmd.Flags().Duration(flagExpeditedVotingPeriod, 0, "Replace the gov expedited voting period (0 keeps it)")
	return cmd
}

// newTestnetApp builds the app as `start` would and then rewrites its state into
// a testnet, see app.InitAppForTestnet.
func newTestnetApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	wasmApp, ok := newApp(logger, db, traceStore, appOpts).(*app.WasmApp)
	if !ok {
		panic("app created from newApp is not of type *app.WasmApp")
	}

	opts, err := testnetOptions(appOpts)
	if err != nil {
		panic(fmt.Errorf("in-place-testnet: %w", err))
	}
	if err := app.InitAppForTestnet(wasmApp, opts); err != nil {
		panic(fmt.Errorf("in-place-testnet: %w", err))
	}
	return wasmApp
}

func testnetOptions(appOpts servertypes.AppOptions) (app.TestnetOptions, error) {
	var opts app.TestnetOptions

	// set by the SDK from the node's priv_validator_key.json, the key it puts in
	// CometBFT's validator set
	cmtPubKey, ok := appOpts.Get(server.KeyUserPubKey).(cmtcrypto.PubKey)
	if !ok {
		return opts, fmt.Errorf("missing %s", server.KeyUserPubKey)
	}
	pubKey, err := cryptocodec.FromCmtPubKeyInterface(cmtPubKey)
	if err != nil {
		return opts, fmt.Errorf("validator pubkey: %w", err)
	}
	opts.ValidatorPubKey = pubKey

	opts.OperatorAddress, err = sdk.ValAddressFromBech32(cast.ToString(appOpts.Get(server.KeyNewOpAddr)))
	if err != nil {
		return opts, fmt.Errorf("operator address: %w", err)
	}
	opts.UpgradeToTrigger = cast.ToString(appOpts.Get(server.KeyTriggerTestnetUpgrade))

	if opts.ValidatorStake, err = parseAmount(appOpts, flagValidatorStake); err != nil {
		return opts, err
	}
	if opts.FundAmount, err = parseAmount(appOpts, flagFundAmount); err != nil {
		return opts, err
	}
	for _, account := range strings.Split(cast.ToString(appOpts.Get(flagAccountsToFund)), ",") {
		account = strings.TrimSpace(account)
		if account == "" {
			continue
		}
		addr, err := sdk.AccAddressFromBech32(account)
		if err != nil {
			return opts, fmt.Errorf("%s: %w", flagAccountsToFund, err)
		}
		opts.AccountsToFund = append(opts.AccountsToFund, addr)
	}

	if opts.VotingPeriod, err = parseDuration(appOpts, flagVotingPeriod); err != nil {
		return opts, err
	}
	if opts.ExpeditedVotingPeriod, err = parseDuration(appOpts, flagExpeditedVotingPeriod); err != nil {
		return opts, err
	}
	return opts, nil
}

func parseAmount(appOpts servertypes.AppOptions, flag string) (math.Int, error) {
	amount, ok := math.NewIntFromString(cast.ToString(appOpts.Get(flag)))
	if !ok || amount.IsNegative() {
		return math.Int{}, fmt.Errorf("%s: invalid amount %q", flag, appOpts.Get(flag))
	}
	return amount, nil
}

func parseDuration(appOpts servertypes.AppOptions, flag string) (time.Duration, error) {
	d, err := cast.ToDurationE(appOpts.Get(flag))
	if err != nil || d < 0 {
		return 0, fmt.Errorf("%s: invalid duration %q", flag, appOpts.Get(flag))
	}
	return d, nil
}

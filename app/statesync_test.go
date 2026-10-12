package app

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// TestStateSyncIntoFreshNode drives a snapshot from a running app into a freshly
// constructed app through the same ABCI calls CometBFT's statesync reactor makes.
// A fresh node must not write to any store before the snapshot is applied, or the
// IAVL importer rejects that store with "tree must be empty".
func TestStateSyncIntoFreshNode(t *testing.T) {
	srcApp := setupSnapshotSourceApp(t)

	_, err := srcApp.Commit()
	require.NoError(t, err)
	height := uint64(srcApp.LastBlockHeight())

	snapshot, err := srcApp.SnapshotManager().Create(height)
	require.NoError(t, err)
	abciSnapshot, err := snapshot.ToABCI()
	require.NoError(t, err)

	// a fresh node, as started by `nyxd start` with an empty data dir
	destApp := SetupWithEmptyStore(t)

	offer, err := destApp.OfferSnapshot(&abci.RequestOfferSnapshot{
		Snapshot: &abciSnapshot,
		AppHash:  srcApp.LastCommitID().Hash,
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseOfferSnapshot_ACCEPT, offer.Result)

	for i := uint32(0); i < snapshot.Chunks; i++ {
		chunk, err := srcApp.SnapshotManager().LoadChunk(snapshot.Height, snapshot.Format, i)
		require.NoError(t, err)
		res, err := destApp.ApplySnapshotChunk(&abci.RequestApplySnapshotChunk{Index: i, Chunk: chunk})
		require.NoError(t, err)
		require.Equal(t, abci.ResponseApplySnapshotChunk_ACCEPT, res.Result, "chunk %d of %d", i, snapshot.Chunks)
	}

	require.Equal(t, srcApp.LastBlockHeight(), destApp.LastBlockHeight())
	require.Equal(t, srcApp.LastCommitID(), destApp.LastCommitID())
}

func setupSnapshotSourceApp(t *testing.T) *WasmApp {
	t.Helper()
	privKey := ed25519.GenPrivKey()
	pubKey, err := cryptocodec.ToCmtPubKeyInterface(privKey.PubKey())
	require.NoError(t, err)

	acc := authtypes.NewBaseAccount(privKey.PubKey().Address().Bytes(), privKey.PubKey(), 0, 0)
	amount, ok := sdkmath.NewIntFromString("10000000000000000000")
	require.True(t, ok)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amount)),
	}
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{cmttypes.NewValidator(pubKey, 1)})
	return SetupWithGenesisValSet(t, valSet, []authtypes.GenesisAccount{acc}, "testing", nil, balance)
}

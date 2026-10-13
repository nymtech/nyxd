package v061_test

import (
	"os"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/nymtech/nyxd/app"
	"github.com/nymtech/nyxd/app/upgrades"
	v061 "github.com/nymtech/nyxd/app/upgrades/v061"
)

func TestConfigurePinning(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.NewUncachedContext(false, cmtproto.Header{Height: wasmApp.LastBlockHeight() + 1, Time: time.Now().UTC(), ChainID: "testing"})
	keepers := &upgrades.AppKeepers{WasmKeeper: &wasmApp.WasmKeeper, AuthzKeeper: &wasmApp.AuthzKeeper}

	creator := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	code, err := os.ReadFile("../../../tests/integration/testdata/hackatom.wasm")
	require.NoError(t, err)
	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(&wasmApp.WasmKeeper)
	codeID, _, err := contractKeeper.Create(ctx, creator, code, nil)
	require.NoError(t, err)
	initMsg := []byte(`{"verifier":"` + creator.String() + `","beneficiary":"` + creator.String() + `"}`)
	contract, _, err := contractKeeper.Instantiate(ctx, codeID, creator, nil, initMsg, "hackatom", nil)
	require.NoError(t, err)
	// a second stored code that nothing references must stay unpinned
	otherCodeID, _, err := contractKeeper.Create(ctx, creator, code, nil)
	require.NoError(t, err)

	operator := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	missing := sdk.AccAddress(make([]byte, 32)).String()
	require.NoError(t, v061.ConfigurePinning(ctx, keepers, []string{contract.String(), missing}, operator.String()))

	require.True(t, wasmApp.WasmKeeper.IsPinnedCode(ctx, codeID))
	require.False(t, wasmApp.WasmKeeper.IsPinnedCode(ctx, otherCodeID))

	gov := authtypes.NewModuleAddress(govtypes.ModuleName)
	for _, msg := range []sdk.Msg{&wasmtypes.MsgPinCodes{}, &wasmtypes.MsgUnpinCodes{}} {
		auth, exp := wasmApp.AuthzKeeper.GetAuthorization(ctx, operator, gov, sdk.MsgTypeURL(msg))
		require.NotNil(t, auth, sdk.MsgTypeURL(msg))
		require.Nil(t, exp, "grants do not expire; governance revokes them")
	}
	// nothing beyond pinning
	for _, msg := range []sdk.Msg{&wasmtypes.MsgUpdateParams{}, &wasmtypes.MsgMigrateContract{}, &wasmtypes.MsgSudoContract{}} {
		auth, _ := wasmApp.AuthzKeeper.GetAuthorization(ctx, operator, gov, sdk.MsgTypeURL(msg))
		require.Nil(t, auth, sdk.MsgTypeURL(msg))
	}
}

func TestConfigurePinningSkipsOtherChains(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.NewUncachedContext(false, cmtproto.Header{Height: wasmApp.LastBlockHeight() + 1, Time: time.Now().UTC(), ChainID: "testing"})
	keepers := &upgrades.AppKeepers{WasmKeeper: &wasmApp.WasmKeeper, AuthzKeeper: &wasmApp.AuthzKeeper}

	// the real mainnet addresses do not exist in a fresh test chain
	require.NoError(t, v061.ConfigurePinning(ctx, keepers, v061.PinnedContracts, v061.PinOperator))

	grantee := sdk.AccAddress(mustDecode(t, v061.PinOperator))
	grants, err := wasmApp.AuthzKeeper.GetAuthorizations(ctx, grantee, authtypes.NewModuleAddress(govtypes.ModuleName))
	require.NoError(t, err)
	require.Empty(t, grants)
}

func TestMainnetAddresses(t *testing.T) {
	for _, c := range v061.PinnedContracts {
		hrp, bz, err := bech32.DecodeAndConvert(c)
		require.NoError(t, err, c)
		require.Equal(t, "n", hrp)
		require.Len(t, bz, 32, "contract address %s", c)
	}
	hrp, bz, err := bech32.DecodeAndConvert(v061.PinOperator)
	require.NoError(t, err)
	require.Equal(t, "n", hrp)
	require.Len(t, bz, 20)
}

func mustDecode(t *testing.T, addr string) []byte {
	t.Helper()
	_, bz, err := bech32.DecodeAndConvert(addr)
	require.NoError(t, err)
	return bz
}

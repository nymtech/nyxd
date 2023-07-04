package main

import (
	"os"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	
	"github.com/CosmWasm/wasmd/app"
)

func main() {
	overrideWasmVariables()
	rootCmd, _ := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}

// overrideWasmVariables overrides the wasm variables to:
//   - allow for larger wasm files
func overrideWasmVariables() {
	// Override Wasm size limitation from WASMD.
	wasmtypes.MaxWasmSize = 3 * 1024 * 1024
	wasmtypes.MaxProposalWasmSize = wasmtypes.MaxWasmSize
}

package app

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	ibctransferkeeper "github.com/cosmos/ibc-go/v3/modules/apps/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v3/modules/core/keeper"
	"github.com/cosmos/wasmd/app/params"
	"github.com/cosmos/wasmd/x/wasm"
	stakingkeeper "github.com/iqlusioninc/liquidity-staking-module/x/staking/keeper"
)

type TestSupport struct {
	t   testing.TB
	app *WasmApp
}

func NewTestSupport(t testing.TB, app *WasmApp) *TestSupport {
	return &TestSupport{t: t, app: app}
}

func (s TestSupport) IBCKeeper() *ibckeeper.Keeper {
	return s.app.IBCKeeper
}

func (s TestSupport) WasmKeeper() wasm.Keeper {
	return s.app.WasmKeeper
}

func (s TestSupport) AppCodec() codec.Codec {
	return s.app.appCodec
}
func (s TestSupport) ScopedWasmIBCKeeper() capabilitykeeper.ScopedKeeper {
	return s.app.scopedWasmKeeper
}

func (s TestSupport) ScopeIBCKeeper() capabilitykeeper.ScopedKeeper {
	return s.app.ScopedIBCKeeper
}

func (s TestSupport) ScopedTransferKeeper() capabilitykeeper.ScopedKeeper {
	return s.app.ScopedTransferKeeper
}

func (s TestSupport) StakingKeeper() stakingkeeper.Keeper {
	return s.app.StakingKeeper
}

func (s TestSupport) BankKeeper() bankkeeper.Keeper {
	return s.app.BankKeeper
}

func (s TestSupport) TransferKeeper() ibctransferkeeper.Keeper {
	return s.app.TransferKeeper
}

func (s TestSupport) GetBaseApp() *baseapp.BaseApp {
	return s.app.BaseApp
}

func (s TestSupport) GetTxConfig() client.TxConfig {
	return params.MakeEncodingConfig().TxConfig
}

// We are not going to add ICA-related methods in cosmos/wasmd since we are not trying
// to add IA support to wasmd.

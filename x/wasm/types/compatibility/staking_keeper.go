package compatibility

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	stakingtypessdk "github.com/cosmos/cosmos-sdk/x/staking/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	ibckeeper "github.com/cosmos/ibc-go/v3/modules/core/keeper"
	stakingkeeper "github.com/iqlusioninc/liquidity-staking-module/x/staking/keeper"
	stakingtypes "github.com/iqlusioninc/liquidity-staking-module/x/staking/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

type IBCCompatibleStakingKeeper struct {
	stakingkeeper.Keeper
}

func NewIBCCompatibleStakingKeeper(keeper stakingkeeper.Keeper) clienttypes.StakingKeeper {
	return &IBCCompatibleStakingKeeper{
		Keeper: keeper,
	}
}

func (k *IBCCompatibleStakingKeeper) GetHistoricalInfo(ctx sdk.Context, height int64) (stakingtypessdk.HistoricalInfo, bool) {
	historicalInfo, ok := k.Keeper.GetHistoricalInfo(ctx, height)
	if !ok {
		return stakingtypessdk.HistoricalInfo{}, false
	}

	sdkHistoricalInfo := stakingtypessdk.HistoricalInfo{
		Header: historicalInfo.GetHeader(),
		Valset: CastValidatorsLiquidToSDK(historicalInfo.GetValset()),
	}

	return sdkHistoricalInfo, ok
}

func CastValidatorsLiquidToSDK(validators []stakingtypes.Validator) (ret []stakingtypessdk.Validator) {
	// Proto3 returns the Zero Value even if a field isn't set. It seems like
	// there is no need to check every embedded value for nils.
	for _, val := range validators {
		sdkValidator := stakingtypessdk.Validator{
			OperatorAddress: val.OperatorAddress,
			ConsensusPubkey: val.ConsensusPubkey,
			Jailed:          val.Jailed,
			Status:          val.GetStatus(),
			Tokens:          val.GetTokens(),
			DelegatorShares: val.GetDelegatorShares(),
			Description: stakingtypessdk.Description{
				Moniker:         val.Description.GetMoniker(),
				Identity:        val.Description.GetIdentity(),
				Website:         val.Description.GetWebsite(),
				SecurityContact: val.Description.GetSecurityContact(),
				Details:         val.Description.GetDetails(),
			},
			UnbondingHeight: val.UnbondingHeight,
			UnbondingTime:   val.UnbondingTime,
			Commission: stakingtypessdk.Commission{
				CommissionRates: stakingtypessdk.CommissionRates{
					Rate:          val.Commission.CommissionRates.Rate,
					MaxRate:       val.Commission.CommissionRates.MaxRate,
					MaxChangeRate: val.Commission.CommissionRates.MaxChangeRate,
				},
				UpdateTime: val.Commission.UpdateTime,
			},
			MinSelfDelegation: val.MinSelfDelegation,
		}
		ret = append(ret, sdkValidator)
	}

	return
}

type IBCTestingApp interface {
	abci.Application

	GetBaseApp() *baseapp.BaseApp
	GetStakingKeeper() stakingkeeper.Keeper
	GetIBCKeeper() *ibckeeper.Keeper
	GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper
	GetTxConfig() client.TxConfig

	AppCodec() codec.Codec

	LastCommitID() storetypes.CommitID
	LastBlockHeight() int64
}

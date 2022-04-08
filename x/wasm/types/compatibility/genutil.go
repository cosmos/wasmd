package compatibility

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	cfg "github.com/tendermint/tendermint/config"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankexported "github.com/cosmos/cosmos-sdk/x/bank/exported"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/iqlusioninc/liquidity-staking-module/x/staking/types"
)

// ValidateAccountInGenesis checks that the provided account has a sufficient
// balance in the set of genesis accounts.
func ValidateAccountInGenesis(
	appGenesisState map[string]json.RawMessage, genBalIterator types.GenesisBalancesIterator,
	addr sdk.Address, coins sdk.Coins, cdc codec.JSONCodec,
) error {

	var stakingData stakingtypes.GenesisState
	cdc.MustUnmarshalJSON(appGenesisState[stakingtypes.ModuleName], &stakingData)
	bondDenom := stakingData.Params.BondDenom

	var err error

	accountIsInGenesis := false

	genBalIterator.IterateGenesisBalances(cdc, appGenesisState,
		func(bal bankexported.GenesisBalance) (stop bool) {
			accAddress := bal.GetAddress()
			accCoins := bal.GetCoins()

			// ensure that account is in genesis
			if accAddress.Equals(addr) {
				// ensure account contains enough funds of default bond denom
				if coins.AmountOf(bondDenom).GT(accCoins.AmountOf(bondDenom)) {
					err = fmt.Errorf(
						"account %s has a balance in genesis, but it only has %v%s available to stake, not %v%s",
						addr, accCoins.AmountOf(bondDenom), bondDenom, coins.AmountOf(bondDenom), bondDenom,
					)

					return true
				}

				accountIsInGenesis = true
				return true
			}

			return false
		},
	)

	if err != nil {
		return err
	}

	if !accountIsInGenesis {
		return fmt.Errorf("account %s does not have a balance in the genesis state", addr)
	}

	return nil
}

// GenAppStateFromConfig gets the genesis app state from the config
func GenAppStateFromConfig(cdc codec.JSONCodec, txEncodingConfig client.TxEncodingConfig,
	config *cfg.Config, initCfg genutiltypes.InitConfig, genDoc tmtypes.GenesisDoc, genBalIterator types.GenesisBalancesIterator,
) (appState json.RawMessage, err error) {

	// process genesis transactions, else create default genesis.json
	appGenTxs, persistentPeers, err := CollectTxs(
		cdc, txEncodingConfig.TxJSONDecoder(), config.Moniker, initCfg.GenTxsDir, genDoc, genBalIterator,
	)
	if err != nil {
		return appState, err
	}

	config.P2P.PersistentPeers = persistentPeers
	cfg.WriteConfigFile(config.RootDir, config)

	// if there are no gen txs to be processed, return the default empty state
	if len(appGenTxs) == 0 {
		return appState, errors.New("there must be at least one genesis tx")
	}

	// create the app state
	appGenesisState, err := genutiltypes.GenesisStateFromGenDoc(genDoc)
	if err != nil {
		return appState, err
	}

	appGenesisState, err = genutil.SetGenTxsInAppGenesisState(cdc, txEncodingConfig.TxJSONEncoder(), appGenesisState, appGenTxs)
	if err != nil {
		return appState, err
	}

	appState, err = json.MarshalIndent(appGenesisState, "", "  ")
	if err != nil {
		return appState, err
	}

	genDoc.AppState = appState
	err = genutil.ExportGenesisFile(&genDoc, config.GenesisFile())

	return appState, err
}

// CollectTxs processes and validates application's genesis Txs and returns
// the list of appGenTxs, and persistent peers required to generate genesis.json.
func CollectTxs(cdc codec.JSONCodec, txJSONDecoder sdk.TxDecoder, moniker, genTxsDir string,
	genDoc tmtypes.GenesisDoc, genBalIterator types.GenesisBalancesIterator,
) (appGenTxs []sdk.Tx, persistentPeers string, err error) {
	// prepare a map of all balances in genesis state to then validate
	// against the validators addresses
	var appState map[string]json.RawMessage
	if err := json.Unmarshal(genDoc.AppState, &appState); err != nil {
		return appGenTxs, persistentPeers, err
	}

	var fos []os.DirEntry
	fos, err = os.ReadDir(genTxsDir)
	if err != nil {
		return appGenTxs, persistentPeers, err
	}

	balancesMap := make(map[string]bankexported.GenesisBalance)

	genBalIterator.IterateGenesisBalances(
		cdc, appState,
		func(balance bankexported.GenesisBalance) (stop bool) {
			balancesMap[balance.GetAddress().String()] = balance
			return false
		},
	)

	// addresses and IPs (and port) validator server info
	var addressesIPs []string

	for _, fo := range fos {
		if fo.IsDir() {
			continue
		}
		if !strings.HasSuffix(fo.Name(), ".json") {
			continue
		}

		// get the genTx
		jsonRawTx, err := os.ReadFile(filepath.Join(genTxsDir, fo.Name()))
		if err != nil {
			return appGenTxs, persistentPeers, err
		}

		var genTx sdk.Tx
		if genTx, err = txJSONDecoder(jsonRawTx); err != nil {
			return appGenTxs, persistentPeers, err
		}

		appGenTxs = append(appGenTxs, genTx)

		// the memo flag is used to store
		// the ip and node-id, for example this may be:
		// "528fd3df22b31f4969b05652bfe8f0fe921321d5@192.168.2.37:26656"

		memoTx, ok := genTx.(sdk.TxWithMemo)
		if !ok {
			return appGenTxs, persistentPeers, fmt.Errorf("expected TxWithMemo, got %T", genTx)
		}
		nodeAddrIP := memoTx.GetMemo()
		if len(nodeAddrIP) == 0 {
			return appGenTxs, persistentPeers, fmt.Errorf("failed to find node's address and IP in %s", fo.Name())
		}

		// genesis transactions must be single-message
		msgs := genTx.GetMsgs()

		// TODO abstract out staking message validation back to staking
		msg := msgs[0].(*stakingtypes.MsgCreateValidator)

		// validate delegator and validator addresses and funds against the accounts in the state
		delAddr := msg.DelegatorAddress
		valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
		if err != nil {
			return appGenTxs, persistentPeers, err
		}

		delBal, delOk := balancesMap[delAddr]
		if !delOk {
			_, file, no, ok := runtime.Caller(1)
			if ok {
				fmt.Printf("CollectTxs-1, called from %s#%d\n", file, no)
			}

			return appGenTxs, persistentPeers, fmt.Errorf("account %s balance not in genesis state: %+v", delAddr, balancesMap)
		}

		_, valOk := balancesMap[sdk.AccAddress(valAddr).String()]
		if !valOk {
			_, file, no, ok := runtime.Caller(1)
			if ok {
				fmt.Printf("CollectTxs-2, called from %s#%d - %s\n", file, no, sdk.AccAddress(msg.ValidatorAddress).String())
			}
			return appGenTxs, persistentPeers, fmt.Errorf("account %s balance not in genesis state: %+v", valAddr, balancesMap)
		}

		if delBal.GetCoins().AmountOf(msg.Value.Denom).LT(msg.Value.Amount) {
			return appGenTxs, persistentPeers, fmt.Errorf(
				"insufficient fund for delegation %v: %v < %v",
				delBal.GetAddress().String(), delBal.GetCoins().AmountOf(msg.Value.Denom), msg.Value.Amount,
			)
		}

		// exclude itself from persistent peers
		if msg.Description.Moniker != moniker {
			addressesIPs = append(addressesIPs, nodeAddrIP)
		}
	}

	sort.Strings(addressesIPs)
	persistentPeers = strings.Join(addressesIPs, ",")

	return appGenTxs, persistentPeers, nil
}

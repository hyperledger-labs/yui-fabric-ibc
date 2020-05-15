package compat

import (
	"os"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

func MakeContext(stub shim.ChaincodeStubInterface, keys map[string]*sdk.KVStoreKey) sdk.Context {
	cms := store.NewCommitMultiStore(NewDB(stub))
	for _, key := range keys {
		cms.MountStoreWithDB(key, sdk.StoreTypeIAVL, nil)
	}
	if err := cms.LoadLatestVersion(); err != nil {
		panic(err)
	}
	return sdk.NewContext(cms, abci.Header{}, false, log.NewTMLogger(os.Stdout))
}

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode/app"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode/commitment"
	"github.com/hyperledger-labs/yui-fabric-ibc/simapp"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmdb "github.com/tendermint/tm-db"
)

func main() {
	cc := chaincode.NewIBCChaincode(
		"fabricibc",
		tmlog.NewTMLogger(os.Stdout),
		commitment.NewDefaultSequenceManager(),
		newApp,
		simapp.DefaultAnteHandler,
		chaincode.DefaultDBProvider,
		chaincode.DefaultMultiEventHandler(),
	)
	chaincode, err := contractapi.NewChaincode(cc)

	if err != nil {
		fmt.Printf("Error create IBC chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting IBC chaincode: %s", err.Error())
		return
	}
}

func newApp(appName string, logger tmlog.Logger, db tmdb.DB, traceStore io.Writer, seqMgr commitment.SequenceManager, blockProvider app.BlockProvider, anteHandlerProvider app.AnteHandlerProvider) (app.Application, error) {
	return simapp.NewIBCApp(
		appName,
		logger,
		db,
		traceStore,
		simapp.MakeEncodingConfig(),
		seqMgr,
		blockProvider,
		anteHandlerProvider,
	)
}

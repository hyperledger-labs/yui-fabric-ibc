package main

import (
	"fmt"
	"io"
	"os"

	"github.com/datachainlab/fabric-ibc/app"
	"github.com/datachainlab/fabric-ibc/chaincode"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/datachainlab/fabric-ibc/example"
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
		chaincode.DefaultDBProvider,
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

func newApp(appName string, logger tmlog.Logger, db tmdb.DB, traceStore io.Writer, seqMgr commitment.SequenceManager, blockProvider app.BlockProvider) (app.Application, error) {
	return example.NewIBCApp(
		appName,
		logger,
		db,
		traceStore,
		example.MakeEncodingConfig(),
		seqMgr,
		blockProvider,
	)
}

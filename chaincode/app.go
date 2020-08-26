package chaincode

import (
	"io"

	"github.com/datachainlab/fabric-ibc/app"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/datachainlab/fabric-ibc/x/compat"
	client "github.com/datachainlab/fabric-ibc/x/ibc/02-client"
	fabric "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

type AppRunner struct {
	logger     log.Logger
	traceStore io.Writer
	dbProvider DBProvider
	seqMgr     *commitment.SequenceManager
}

func NewAppRunner(logger log.Logger, dbProvider DBProvider, seqMgr *commitment.SequenceManager) AppRunner {
	return AppRunner{
		logger:     logger,
		dbProvider: dbProvider,
		seqMgr:     seqMgr,
	}
}

func (r AppRunner) Init(stub shim.ChaincodeStubInterface, appStateBytes []byte) error {
	return r.RunFunc(stub, func(app app.Application) error {
		return app.InitChain(appStateBytes)
	})
}

func (r AppRunner) RunFunc(stub shim.ChaincodeStubInterface, f func(app.Application) error) error {
	db := r.dbProvider(stub)
	app, err := app.NewIBCApp(r.logger, db, r.traceStore, r.getSelfConsensusStateProvider(stub), r.getBlockProvider(stub))
	if err != nil {
		return err
	}
	if err := f(app); err != nil {
		return err
	}
	return nil
}

func (r AppRunner) RunMsg(stub shim.ChaincodeStubInterface, txBytes []byte) ([]abci.Event, error) {
	db := r.dbProvider(stub)
	app, err := app.NewIBCApp(r.logger, db, r.traceStore, r.getSelfConsensusStateProvider(stub), r.getBlockProvider(stub))
	if err != nil {
		return nil, err
	}
	res, err := app.RunTx(stub, txBytes)
	if err != nil {
		return nil, err
	}
	return res.Events, nil
}

func (r AppRunner) getSelfConsensusStateProvider(stub shim.ChaincodeStubInterface) app.SelfConsensusStateKeeperProvider {
	return func() client.SelfConsensusStateKeeper {
		return fabric.NewConsensusStateKeeper(stub, r.seqMgr)
	}
}

type block struct {
	height    int64
	timestamp int64
}

func (bk block) Height() int64 {
	return bk.height
}

func (bk block) Timestamp() int64 {
	return bk.timestamp
}

func (r AppRunner) getBlockProvider(stub shim.ChaincodeStubInterface) app.BlockProvider {
	return func() app.Block {
		seq, err := r.seqMgr.GetCurrentSequence(stub)
		if err != nil {
			panic(err)
		}
		return block{height: int64(seq.Value), timestamp: seq.Timestamp}
	}
}

type DBProvider func(shim.ChaincodeStubInterface) dbm.DB

func DefaultDBProvider(stub shim.ChaincodeStubInterface) dbm.DB {
	return compat.NewDB(stub)
}

package chaincode

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hyperledger-labs/yui-fabric-ibc/app"
	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
	"github.com/hyperledger-labs/yui-fabric-ibc/x/compat"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

type AppProvider func(appName string, logger log.Logger, db dbm.DB, traceStore io.Writer, seqMgr commitment.SequenceManager, blockProvider app.BlockProvider, anteHandlerProvider app.AnteHandlerProvider) (app.Application, error)

type AppRunner struct {
	appName             string
	logger              log.Logger
	traceStore          io.Writer
	appProvider         AppProvider
	anteHandlerProvider app.AnteHandlerProvider
	dbProvider          DBProvider
	seqMgr              commitment.SequenceManager
}

func NewAppRunner(
	appName string,
	logger log.Logger,
	appProvider AppProvider,
	anteHandlerProvider app.AnteHandlerProvider,
	dbProvider DBProvider,
	seqMgr commitment.SequenceManager,
) AppRunner {
	return AppRunner{
		appName:             appName,
		logger:              logger,
		appProvider:         appProvider,
		anteHandlerProvider: anteHandlerProvider,
		dbProvider:          dbProvider,
		seqMgr:              seqMgr,
	}
}

func (r AppRunner) Init(stub shim.ChaincodeStubInterface, appStateBytes []byte) error {
	return r.RunFunc(stub, func(app app.Application) error {
		return app.InitChain(appStateBytes)
	})
}

func (r AppRunner) RunFunc(stub shim.ChaincodeStubInterface, f func(app.Application) error) error {
	db := r.dbProvider(stub)
	app, err := r.appProvider(r.appName, r.logger, db, r.traceStore, r.seqMgr, r.GetBlockProvider(stub), r.anteHandlerProvider)
	if err != nil {
		return err
	}
	if err := f(app); err != nil {
		return err
	}
	return nil
}

func (r AppRunner) RunTx(stub shim.ChaincodeStubInterface, txBytes []byte) (*app.ResponseTx, []abci.Event, error) {
	db := r.dbProvider(stub)
	app, err := r.appProvider(r.appName, r.logger, db, r.traceStore, r.seqMgr, r.GetBlockProvider(stub), r.anteHandlerProvider)
	if err != nil {
		return nil, nil, err
	}
	res, err := app.RunTx(stub, txBytes)
	if err != nil {
		return nil, nil, err
	}
	return makeResponseTx(*res), res.Events, nil
}

func (r AppRunner) Query(stub shim.ChaincodeStubInterface, req app.RequestQuery) (*app.ResponseQuery, error) {
	db := r.dbProvider(stub)
	a, err := r.appProvider(r.appName, r.logger, db, r.traceStore, r.seqMgr, r.GetBlockProvider(stub), r.anteHandlerProvider)
	if err != nil {
		return nil, err
	}
	data, err := DecodeString(req.Data)
	if err != nil {
		return nil, err
	}
	res := a.Query(abci.RequestQuery{Data: data, Path: req.Path})
	if res.IsErr() {
		return nil, fmt.Errorf("failed to query '%v': %v", req.Path, res.Log)
	}
	return &app.ResponseQuery{Key: string(res.Key), Value: EncodeToString(res.Value)}, nil
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

func (r AppRunner) GetBlockProvider(stub shim.ChaincodeStubInterface) app.BlockProvider {
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

func makeResponseTx(res sdk.Result) *app.ResponseTx {
	bz, err := json.Marshal(res.Events)
	if err != nil {
		panic(err)
	}
	return &app.ResponseTx{
		Data:   string(res.Data),
		Events: string(bz),
		Log:    res.Log,
	}
}

func EncodeToString(bz []byte) string {
	return base64.StdEncoding.EncodeToString(bz)
}

func DecodeString(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

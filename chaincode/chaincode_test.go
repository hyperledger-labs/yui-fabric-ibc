package chaincode_test

import (
	"io"
	"testing"
	"time"

	"github.com/datachainlab/fabric-ibc/app"
	"github.com/datachainlab/fabric-ibc/chaincode"
	"github.com/datachainlab/fabric-ibc/example"
	"github.com/datachainlab/fabric-ibc/x/compat"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/require"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmdb "github.com/tendermint/tm-db"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestResponseSerializer(t *testing.T) {
	require := require.New(t)

	cc := chaincode.NewIBCChaincode(newApp, chaincode.DefaultDBProvider)
	chaincode, err := contractapi.NewChaincode(cc)
	require.NoError(err)

	stub0 := compat.MakeFakeStub()

	// Initialize chaincode
	res := chaincode.Init(stub0)
	require.EqualValues(200, res.Status, res.String())

	now := time.Now()
	stub0.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: now.Unix()}, nil)
	stub0.GetFunctionAndParametersReturns("InitChaincode", []string{"{}"})
	res = chaincode.Invoke(stub0)
	require.EqualValues(200, res.Status, res.String())

	// UpdateSequence
	now = now.Add(10 * time.Second)
	stub0.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: now.Unix()}, nil)
	stub0.GetFunctionAndParametersReturns("UpdateSequence", nil)
	res = chaincode.Invoke(stub0)
	require.EqualValues(200, res.Status, res.String())

	// EndorseSequenceCommitment
	now = now.Add(10 * time.Second)
	stub0.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: now.Unix()}, nil)
	stub0.GetFunctionAndParametersReturns("EndorseSequenceCommitment", []string{"2"})
	res = chaincode.Invoke(stub0)
	require.EqualValues(200, res.Status, res.String())

	// Query
	// FIXME we should use grpc query instead of legacy query
	// bz := channeltypes.SubModuleCdc.MustMarshalJSON(channeltypes.QueryAllChannelsParams{Limit: 100, Page: 1})
	// req := app.RequestQuery{Data: string(bz), Path: "/custom/ibc/channel/channels"}
	// jbz, err := json.Marshal(req)
	// require.NoError(err)
	// now = now.Add(10 * time.Second)
	// stub0.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: now.Unix()}, nil)
	// stub0.GetFunctionAndParametersReturns("Query", []string{string(jbz)})
	// res = chaincode.Invoke(stub0)
	// require.EqualValues(200, res.Status, res.String())
}

func newApp(logger tmlog.Logger, db tmdb.DB, traceStore io.Writer, blockProvider app.BlockProvider) (app.Application, error) {
	return example.NewIBCApp(
		logger,
		db,
		traceStore,
		example.MakeEncodingConfig(),
		blockProvider,
	)
}

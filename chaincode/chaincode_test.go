package chaincode_test

import (
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger-labs/yui-fabric-ibc/app"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode"
	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
	"github.com/hyperledger-labs/yui-fabric-ibc/example"
	testsstub "github.com/hyperledger-labs/yui-fabric-ibc/tests/stub"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/require"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmdb "github.com/tendermint/tm-db"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestResponseSerializer(t *testing.T) {
	require := require.New(t)

	cc := chaincode.NewIBCChaincode(
		"fabricibc",
		tmlog.NewTMLogger(os.Stdout),
		commitment.NewDefaultSequenceManager(),
		newApp,
		example.DefaultAnteHandler,
		chaincode.DefaultDBProvider,
		chaincode.DefaultMultiEventHandler(),
	)
	chaincode, err := contractapi.NewChaincode(cc)
	require.NoError(err)

	stub0 := testsstub.MakeFakeStub()

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
	data, err := proto.Marshal(&channeltypes.QueryChannelsRequest{})
	require.NoError(err)
	req := app.RequestQuery{Path: "/ibc.core.channel.v1.Query/Channels", Data: string(data)}
	jbz, err := json.Marshal(req)
	require.NoError(err)
	now = now.Add(10 * time.Second)
	stub0.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: now.Unix()}, nil)
	stub0.GetFunctionAndParametersReturns("Query", []string{string(jbz)})
	res = chaincode.Invoke(stub0)
	require.EqualValues(int32(200), res.Status, res.String())
}

func newApp(appName string, logger tmlog.Logger, db tmdb.DB, traceStore io.Writer, seqMgr commitment.SequenceManager, blockProvider app.BlockProvider, anteHandlerProvider app.AnteHandlerProvider) (app.Application, error) {
	return example.NewIBCApp(
		appName,
		logger,
		db,
		traceStore,
		example.MakeEncodingConfig(),
		seqMgr,
		blockProvider,
		anteHandlerProvider,
	)
}

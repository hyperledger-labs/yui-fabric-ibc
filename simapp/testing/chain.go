package testing

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	csim "github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	ibctransfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/modules/core"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/modules/core/keeper"
	"github.com/cosmos/ibc-go/modules/core/types"
	ibctypes "github.com/cosmos/ibc-go/modules/core/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
	"github.com/cosmos/ibc-go/testing/mock"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/common"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/common/policydsl"
	fabricmock "github.com/hyperledger/fabric/core/chaincode/lifecycle/mock"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode/app"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode/commitment"
	testsstub "github.com/hyperledger-labs/yui-fabric-ibc/chaincode/tests/stub"
	fabricauthante "github.com/hyperledger-labs/yui-fabric-ibc/chaincode/x/auth/ante"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode/x/compat"
	fabrictypes "github.com/hyperledger-labs/yui-fabric-ibc/light-client/types"
	"github.com/hyperledger-labs/yui-fabric-ibc/simapp"
)

const (
	// client types
	Tendermint  = exported.Tendermint
	SoloMachine = exported.Solomachine
	Fabric      = fabrictypes.Fabric

	// Default params constants used to create a TM client
	TrustingPeriod     time.Duration = time.Hour * 24 * 7 * 2
	UnbondingPeriod    time.Duration = time.Hour * 24 * 7 * 3
	MaxClockDrift      time.Duration = time.Second * 10
	DefaultDelayPeriod uint64        = 0

	ChannelTransferVersion = ibctransfertypes.Version

	ConnectionIDPrefix = "conn"
	ChannelIDPrefix    = "chan"

	TransferPort = ibctransfertypes.ModuleName
	MockPort     = mock.ModuleName
)

var (
	DefaultOpenInitVersion *connectiontypes.Version
	ConnectionVersion      = connectiontypes.ExportedVersionsToProto(connectiontypes.GetCompatibleVersions())[0]

	// Default params variables used to create a TM client
	DefaultTrustLevel ibctmtypes.Fraction = ibctmtypes.DefaultTrustLevel
	TestHash                              = tmhash.Sum([]byte("TESTING HASH"))
	TestCoin                              = sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))

	UpgradePath = []string{"upgrade", "upgradedIBCState"}
)

type TestChainI interface {
	Type() string
	GetApp() interface{}
	GetChainID() string
	GetSenderAccount() authtypes.AccountI
	GetLastHeader() *ibctmtypes.Header
	NextBlock()
	GetContext() sdk.Context

	AddTestConnection(clientID, counterpartyClientID, nextChannelVersion string) *TestConnection
	AddTestChannel(conn *TestConnection, portID string) TestChannel
	ConstructNextTestConnection(clientID, counterpartyClientID, nextChannelVersion string) *TestConnection

	GetChannel(testChannel TestChannel) channeltypes.Channel

	ConnectionOpenInit(
		counterparty TestChainI,
		connection, counterpartyConnection *TestConnection,
	) error
	ConnectionOpenTry(
		counterparty TestChainI,
		connection, counterpartyConnection *TestConnection,
	) error
	ConnectionOpenAck(
		counterparty TestChainI,
		connection, counterpartyConnection *TestConnection,
	) error
	ConnectionOpenConfirm(
		counterparty TestChainI,
		connection, counterpartyConnection *TestConnection,
	) error

	ChanOpenInit(
		ch, counterparty TestChannel,
		order channeltypes.Order,
		connectionID string,
	) error
	ChanOpenTry(
		counterparty TestChainI,
		ch, counterpartyCh TestChannel,
		order channeltypes.Order,
		connectionID string,
	) error
	ChanOpenAck(
		counterparty TestChainI,
		ch, counterpartyCh TestChannel,
	) error
	ChanOpenConfirm(
		counterparty TestChainI,
		ch, counterpartyCh TestChannel,
	) error
	ChanCloseInit(
		counterparty TestChainI,
		channel TestChannel,
	) error

	CreatePortCapability(portID string)

	QueryClientStateProof(clientID string) (exported.ClientState, []byte)
	QueryProof(key []byte) ([]byte, clienttypes.Height)
	QueryConsensusStateProof(clientID string) ([]byte, clienttypes.Height)

	NewClientID(clientType string) string
	GetPrefix() commitmenttypes.MerklePrefix

	SendMsgs(msgs ...sdk.Msg) (*sdk.Result, error)
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

// TestChain is a testing struct that wraps a simapp with the last TM Header, the current ABCI
// header and the validators of the TestChain. It also contains a field called ChainID. This
// is the clientID that *other* chains use to refer to this TestChain. The SenderAccount
// is used for delivering transactions through the application state.
// NOTE: the actual application uses an empty chain-id for ease of testing.
type TestChain struct {
	t *testing.T

	App  *simapp.IBCApp
	CC   *chaincode.IBCChaincode
	Stub *fabricmock.ChaincodeStub

	ChainID       string
	LastHeader    *ibctmtypes.Header // header for last block height committed
	CurrentHeader tmproto.Header     // header for current block height
	QueryServer   types.QueryServer
	TxConfig      client.TxConfig
	Codec         codec.BinaryCodec

	Vals    *tmtypes.ValidatorSet
	Signers []tmtypes.PrivValidator

	SenderAccount authtypes.AccountI

	// IBC specific helpers
	ClientIDs   []string          // ClientID's used on this chain
	Connections []*TestConnection // track connectionID's created for this chain

	NextChannelVersion string

	// Fabric
	fabChannelID   string
	fabChaincodeID fabrictypes.ChaincodeID
	endorser       msp.SigningIdentity
	mspConfig      msppb.MSPConfig
	seqMgr         commitment.SequenceManager

	currentTime time.Time
	txSignMode  TxSignMode
}

var _ TestChainI = (*TestChain)(nil)

type TxSignMode uint8

const (
	TxSignModeStdTx TxSignMode = iota + 1
	TxSignModeFabricTx
)

func NewTestFabricChain(t *testing.T, chainID string, mspID string, txSignMode TxSignMode) *TestChain {
	// generate validator private/public key
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)

	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})
	signers := []tmtypes.PrivValidator{privVal}

	logger := tmlog.NewTMLogger(os.Stdout)
	seqMgr := commitment.NewSequenceManager(
		commitment.CommitmentConfig{
			MinTimeInterval:  0,
			MaxTimestampDiff: 30 * time.Second,
		},
		commitmenttypes.NewMerklePrefix([]byte(host.StoreKey)),
	)

	var anteHandlerProvider app.AnteHandlerProvider
	switch txSignMode {
	case TxSignModeStdTx:
		anteHandlerProvider = simapp.DefaultAnteHandler
	case TxSignModeFabricTx:
		anteHandlerProvider = newAnteHandler
	default:
		panic(fmt.Sprintf("unknown txSignMode %v", txSignMode))
	}

	cc := chaincode.NewIBCChaincode(chainID, logger, seqMgr, newApp, anteHandlerProvider, chaincode.DefaultDBProvider, chaincode.DefaultMultiEventHandler())
	runner := cc.GetAppRunner()
	stub := testsstub.MakeFakeStub()
	app, err := newApp(
		chainID,
		logger,
		compat.NewDB(stub),
		nil,
		seqMgr,
		runner.GetBlockProvider(stub),
		anteHandlerProvider,
	)
	require.NoError(t, err)

	genesisState := makeGenesisState()

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	genAccs := []authtypes.GenesisAccount{acc}
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100000000000000))),
	}
	balances := []banktypes.Balance{balance}

	// bondAmt := sdk.NewInt(1000000)
	// totalSupply := sdk.NewCoins()
	// add genesis acc tokens and delegated tokens to total supply
	// totalSupply = totalSupply.Add(balance.Coins.Add(sdk.NewCoin(sdk.DefaultBondDenom, bondAmt))...)
	totalSupply := balance.Coins
	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{})

	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	ibcGenesisState := ibctypes.DefaultGenesisState()
	ibcGenesisState.ClientGenesis.Params.AllowedClients = append(ibcGenesisState.ClientGenesis.Params.AllowedClients, fabrictypes.Fabric)
	genesisState[ibc.AppModule{}.Name()] = app.AppCodec().MustMarshalJSON(ibcGenesisState)

	// create current header and call begin block
	header := tmproto.Header{
		ChainID: chainID,
		Height:  1,
		Time:    globalStartTime,
	}

	// Fabric configuration

	conf, err := fabrictypes.DefaultConfig()
	require.NoError(t, err)
	// setup the MSP manager so that we can sign/verify
	mconf, bconf, err := fabrictypes.GetLocalMspConfigForTest(conf.MSPsDir, mspID)
	require.NoError(t, err)
	lcMSP, err := fabrictypes.SetupLocalMspForTest(mconf, bconf)
	require.NoError(t, err)
	endorser, err := lcMSP.GetDefaultSigningIdentity()
	require.NoError(t, err)

	// TODO fix to clarify this method(or name?)
	err = fabrictypes.GetVerifyingConfig(mconf)
	require.NoError(t, err)

	// create an account to send transactions from
	chain := &TestChain{
		t:                  t,
		ChainID:            chainID,
		App:                app.(*simapp.IBCApp),
		CC:                 cc,
		Stub:               stub,
		CurrentHeader:      header,
		QueryServer:        app.GetIBCKeeper(),
		TxConfig:           csim.MakeTestEncodingConfig().TxConfig,
		Codec:              app.AppCodec(),
		Vals:               valSet,
		Signers:            signers,
		ClientIDs:          make([]string, 0),
		Connections:        make([]*TestConnection, 0),
		NextChannelVersion: ChannelTransferVersion,

		fabChaincodeID: fabrictypes.ChaincodeID{
			Name:    "dummyCC",
			Version: "dummyVer",
		},
		fabChannelID: "dummyChannel",
		mspConfig:    *mconf,
		endorser:     endorser,
		currentTime:  globalStartTime,
		seqMgr:       seqMgr,
		txSignMode:   txSignMode,
	}

	switch txSignMode {
	case TxSignModeStdTx:
		chain.SenderAccount = acc
	case TxSignModeFabricTx:
		chain.SenderAccount = NewAccount(acc, getTestId())
	default:
		panic(fmt.Sprintf("unknown txSignMode %v", txSignMode))
	}

	stub.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: globalStartTime.Unix()}, nil)
	var tctx contractapi.TransactionContext
	tctx.SetStub(stub)

	seqMgr.SetClock(func() time.Time {
		return chain.currentTime
	})

	// Init chaincode
	jsonBytes, err := json.Marshal(genesisState)
	require.NoError(t, err)
	require.NoError(t, chain.CC.InitChaincode(&tctx, string(jsonBytes)))

	// cap := chain.App.IBCKeeper.PortKeeper.BindPort(chain.GetContext(), MockPort)
	// pp.Println(chain.App.ScopedIBCMockKeeper)
	// err = chain.App.ScopedIBCMockKeeper.ClaimCapability(chain.GetContext(), cap, host.PortPath(MockPort))
	// require.NoError(t, err)

	chain.NextBlock()

	return chain
}

func newAnteHandler(
	ibcKeeper ibckeeper.Keeper,
	sigGasConsumer ante.SignatureVerificationGasConsumer,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		fabricauthante.NewFabricIDVerificationDecorator(),
	)
}

func makeGenesisState() csim.GenesisState {
	return simapp.NewDefaultGenesisState()
}

// Type implements TestChainI.Type
func (chain TestChain) Type() string {
	return Fabric
}

func (chain *TestChain) GetContext() sdk.Context {
	ctx, _ := chain.App.MakeCacheContext(chain.CurrentHeader)
	return ctx
}

func (chain *TestChain) GetFabricContext() *contractapi.TransactionContext {
	ctx := contractapi.TransactionContext{}
	ctx.SetStub(chain.Stub)
	return &ctx
}

func (chain TestChain) GetApp() interface{} {
	return chain.App
}

// GetChainID implements TestChainI.GetChainID
func (chain TestChain) GetChainID() string {
	return chain.ChainID
}

func (chain TestChain) GetSenderAccount() authtypes.AccountI {
	return chain.SenderAccount
}

func (chain TestChain) GetLastHeader() *ibctmtypes.Header {
	return chain.LastHeader
}

func (chain *TestChain) NextBlock() {}

// QueryProof performs an abci query with the given key and returns the proto encoded merkle proof
// for the query and the height at which the proof will succeed on a tendermint verifier.
func (chain *TestChain) QueryProof(key []byte) ([]byte, clienttypes.Height) {
	bz, err := queryEndorseCommitment(chain.GetFabricContext(), chain, key)
	require.NoError(chain.t, err)

	version := clienttypes.ParseChainID(chain.ChainID)

	// set height correctly
	return bz, clienttypes.NewHeight(version, 1)
}

// QueryClientStateProof performs and abci query for a client state
// stored with a given clientID and returns the ClientState along with the proof
func (chain *TestChain) QueryClientStateProof(clientID string) (exported.ClientState, []byte) {
	// retrieve client state to provide proof for
	clientState, found := chain.App.IBCKeeper.ClientKeeper.GetClientState(chain.GetContext(), clientID)
	require.True(chain.t, found)

	clientKey := host.FullClientKey(clientID, host.ClientStateKey())
	proofClient, _ := chain.QueryProof(clientKey)

	return clientState, proofClient
}

// QueryConsensusStateProof performs an abci query for a consensus state
// stored on the given clientID. The proof and consensusHeight are returned.
func (chain *TestChain) QueryConsensusStateProof(clientID string) ([]byte, clienttypes.Height) {
	clientState := chain.GetClientState(clientID)

	consensusHeight := clientState.GetLatestHeight().(clienttypes.Height)
	consensusKey := host.FullClientKey(clientID, host.ConsensusStateKey(consensusHeight))
	proofConsensus, _ := chain.QueryProof(consensusKey)

	return proofConsensus, consensusHeight
}

// AddTestConnection appends a new TestConnection which contains references
// to the connection id, client id and counterparty client id.
func (chain *TestChain) AddTestConnection(clientID, counterpartyClientID, nextChannelVersion string) *TestConnection {
	conn := chain.ConstructNextTestConnection(clientID, counterpartyClientID, nextChannelVersion)

	chain.Connections = append(chain.Connections, conn)
	return conn
}

// AddTestChannel appends a new TestChannel which contains references to the port and channel ID
// used for channel creation and interaction. See 'NextTestChannel' for channel ID naming format.
func (chain *TestChain) AddTestChannel(conn *TestConnection, portID string) TestChannel {
	channel := chain.NextTestChannel(conn, portID)
	conn.Channels = append(conn.Channels, channel)
	return channel
}

// NextTestChannel returns the next test channel to be created on this connection, but does not
// add it to the list of created channels. This function is expected to be used when the caller
// has not created the associated channel in app state, but would still like to refer to the
// non-existent channel usually to test for its non-existence.
//
// channel ID format: <connectionid>-chan<channel-index>
//
// The port is passed in by the caller.
func (chain *TestChain) NextTestChannel(conn *TestConnection, portID string) TestChannel {
	nextChanSeq := chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(chain.GetContext())
	channelID := channeltypes.FormatChannelIdentifier(nextChanSeq)
	return TestChannel{
		PortID:               portID,
		ID:                   channelID,
		ClientID:             conn.ClientID,
		CounterpartyClientID: conn.CounterpartyClientID,
		Version:              conn.NextChannelVersion,
	}
}

// ConstructNextTestConnection constructs the next test connection to be
// created given a clientID and counterparty clientID. The connection id
// format: <chainID>-conn<index>
func (chain *TestChain) ConstructNextTestConnection(clientID, counterpartyClientID, nextChannelVersion string) *TestConnection {
	connectionID := connectiontypes.FormatConnectionIdentifier(uint64(len(chain.Connections)))
	return &TestConnection{
		ID:                   connectionID,
		ClientID:             clientID,
		NextChannelVersion:   nextChannelVersion,
		CounterpartyClientID: counterpartyClientID,
	}
}

func (chain *TestChain) CreateClient(counterparty TestChainI, clientID string) error {
	if counterparty.Type() != Fabric {
		panic("not implemented error")
	}

	// construct MsgCreateClient using counterparty
	msg := chain.ConstructMsgCreateClient(counterparty, clientID, Fabric)
	return chain.sendMsgs(msg)
}

// TODO add tests for other headers
// UpdateClient updates the sequence and timestamp
func (chain *TestChain) UpdateClient(counterparty TestChainI, clientID string) error {
	if counterparty.Type() != Fabric {
		panic("not implemented error")
	}

	header, err := chain.ConstructUpdateClientHeader(counterparty, clientID)
	require.NoError(chain.t, err)

	msg, err := clienttypes.NewMsgUpdateClient(
		clientID, header,
		chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(chain.t, err)

	return chain.sendMsgs(msg)
}

func (*TestChain) ConstructUpdateClientHeader(counterparty TestChainI, clientID string) (exported.Header, error) {
	cp := counterparty.(*TestChain)

	var tctx contractapi.TransactionContext
	tctx.SetStub(cp.Stub)
	_, err := cp.CC.UpdateSequence(&tctx)
	if err != nil {
		return nil, err
	}

	seq, err := cp.seqMgr.GetCurrentSequence(cp.Stub)
	if err != nil {
		return nil, err
	}
	e, err := cp.CC.EndorseSequenceCommitment(&tctx)
	if err != nil {
		return nil, err
	}
	proof, err := endorseCommitment(&tctx, cp, e)
	if err != nil {
		return nil, err
	}
	h := fabrictypes.NewChaincodeHeader(seq.Value, seq.Timestamp, *proof)
	return fabrictypes.NewHeader(&h, nil, nil), nil
}

// ConnectionOpenInit will construct and execute a MsgConnectionOpenInit.
func (chain *TestChain) ConnectionOpenInit(
	counterparty TestChainI,
	connection, counterpartyConnection *TestConnection,
) error {
	msg := connectiontypes.NewMsgConnectionOpenInit(
		connection.ClientID, connection.CounterpartyClientID,
		counterparty.GetPrefix(), DefaultOpenInitVersion, DefaultDelayPeriod,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

// ConnectionOpenTry will construct and execute a MsgConnectionOpenTry.
func (chain *TestChain) ConnectionOpenTry(
	counterparty TestChainI,
	connection, counterpartyConnection *TestConnection,
) error {
	counterpartyClient, proofClient := counterparty.QueryClientStateProof(counterpartyConnection.ClientID)

	connectionKey := host.ConnectionKey(counterpartyConnection.ID)
	proofInit, proofHeight := counterparty.QueryProof(connectionKey)

	proofConsensus, consensusHeight := counterparty.QueryConsensusStateProof(counterpartyConnection.ClientID)

	msg := connectiontypes.NewMsgConnectionOpenTry(
		"", connection.ClientID, // does not support handshake continuation
		counterpartyConnection.ID, counterpartyConnection.ClientID,
		counterpartyClient, counterparty.GetPrefix(), []*connectiontypes.Version{ConnectionVersion}, DefaultDelayPeriod,
		proofInit, proofClient, proofConsensus,
		proofHeight, consensusHeight,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

// ConnectionOpenAck will construct and execute a MsgConnectionOpenAck.
func (chain *TestChain) ConnectionOpenAck(
	counterparty TestChainI,
	connection, counterpartyConnection *TestConnection,
) error {
	counterpartyClient, proofClient := counterparty.QueryClientStateProof(counterpartyConnection.ClientID)

	connectionKey := host.ConnectionKey(counterpartyConnection.ID)
	proofTry, proofHeight := counterparty.QueryProof(connectionKey)

	proofConsensus, consensusHeight := counterparty.QueryConsensusStateProof(counterpartyConnection.ClientID)

	msg := connectiontypes.NewMsgConnectionOpenAck(
		connection.ID, counterpartyConnection.ID, counterpartyClient, // testing doesn't use flexible selection
		proofTry, proofClient, proofConsensus,
		proofHeight, consensusHeight,
		ConnectionVersion,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

// ConnectionOpenConfirm will construct and execute a MsgConnectionOpenConfirm.
func (chain *TestChain) ConnectionOpenConfirm(
	counterparty TestChainI,
	connection, counterpartyConnection *TestConnection,
) error {
	connectionKey := host.ConnectionKey(counterpartyConnection.ID)
	proof, height := counterparty.QueryProof(connectionKey)

	msg := connectiontypes.NewMsgConnectionOpenConfirm(
		connection.ID,
		proof, height,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

// CreatePortCapability binds and claims a capability for the given portID if it does not
// already exist. This function will fail testing on any resulting error.
// NOTE: only creation of a capbility for a transfer or mock port is supported
// Other applications must bind to the port in InitGenesis or modify this code.
func (chain *TestChain) CreatePortCapability(portID string) {
	// check if the portId is already binded, if not bind it
	_, ok := chain.App.ScopedIBCKeeper.GetCapability(chain.GetContext(), host.PortPath(portID))
	if !ok {
		// create capability using the IBC capability keeper
		cap, err := chain.App.ScopedIBCKeeper.NewCapability(chain.GetContext(), host.PortPath(portID))
		require.NoError(chain.t, err)

		switch portID {
		case MockPort:
			// claim capability using the mock capability keeper
			err = chain.App.ScopedIBCMockKeeper.ClaimCapability(chain.GetContext(), cap, host.PortPath(portID))
			require.NoError(chain.t, err)
		case TransferPort:
			// claim capability using the transfer capability keeper
			err = chain.App.ScopedTransferKeeper.ClaimCapability(chain.GetContext(), cap, host.PortPath(portID))
			require.NoError(chain.t, err)
		default:
			panic(fmt.Sprintf("unsupported ibc testing package port ID %s", portID))
		}
	}

	// FIXME how do we implement commit?
	// chain.App.Commit()

	chain.NextBlock()
}

// GetPortCapability returns the port capability for the given portID. The capability must
// exist, otherwise testing will fail.
func (chain *TestChain) GetPortCapability(portID string) *capabilitytypes.Capability {
	cap, ok := chain.App.ScopedIBCKeeper.GetCapability(chain.GetContext(), host.PortPath(portID))
	require.True(chain.t, ok)

	return cap
}

// CreateChannelCapability binds and claims a capability for the given portID and channelID
// if it does not already exist. This function will fail testing on any resulting error.
func (chain *TestChain) CreateChannelCapability(portID, channelID string) {
	capName := host.ChannelCapabilityPath(portID, channelID)
	// check if the portId is already binded, if not bind it
	_, ok := chain.App.ScopedIBCKeeper.GetCapability(chain.GetContext(), capName)
	if !ok {
		cap, err := chain.App.ScopedIBCKeeper.NewCapability(chain.GetContext(), capName)
		require.NoError(chain.t, err)
		err = chain.App.ScopedTransferKeeper.ClaimCapability(chain.GetContext(), cap, capName)
		require.NoError(chain.t, err)
	}

	// FIXME how do we implement commit?
	// chain.App.Commit()

	chain.NextBlock()
}

// GetChannelCapability returns the channel capability for the given portID and channelID.
// The capability must exist, otherwise testing will fail.
func (chain *TestChain) GetChannelCapability(portID, channelID string) *capabilitytypes.Capability {
	cap, ok := chain.App.ScopedIBCKeeper.GetCapability(chain.GetContext(), host.ChannelCapabilityPath(portID, channelID))
	require.True(chain.t, ok)

	return cap
}

// ChanOpenInit will construct and execute a MsgChannelOpenInit.
func (chain *TestChain) ChanOpenInit(
	ch, counterparty TestChannel,
	order channeltypes.Order,
	connectionID string,
) error {
	msg := channeltypes.NewMsgChannelOpenInit(
		ch.PortID,
		ch.Version, order, []string{connectionID},
		counterparty.PortID,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

// ChanOpenTry will construct and execute a MsgChannelOpenTry.
func (chain *TestChain) ChanOpenTry(
	counterparty TestChainI,
	ch, counterpartyCh TestChannel,
	order channeltypes.Order,
	connectionID string,
) error {
	proof, height := counterparty.QueryProof(host.ChannelKey(counterpartyCh.PortID, counterpartyCh.ID))

	msg := channeltypes.NewMsgChannelOpenTry(
		ch.PortID, "", // does not support handshake continuation
		ch.Version, order, []string{connectionID},
		counterpartyCh.PortID, counterpartyCh.ID, counterpartyCh.Version,
		proof, height,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

// ChanOpenAck will construct and execute a MsgChannelOpenAck.
func (chain *TestChain) ChanOpenAck(
	counterparty TestChainI,
	ch, counterpartyCh TestChannel,
) error {
	proof, height := counterparty.QueryProof(host.ChannelKey(counterpartyCh.PortID, counterpartyCh.ID))

	msg := channeltypes.NewMsgChannelOpenAck(
		ch.PortID, ch.ID,
		counterpartyCh.ID, counterpartyCh.Version, // testing doesn't use flexible selection
		proof, height,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

// ChanOpenConfirm will construct and execute a MsgChannelOpenConfirm.
func (chain *TestChain) ChanOpenConfirm(
	counterparty TestChainI,
	ch, counterpartyCh TestChannel,
) error {
	proof, height := counterparty.QueryProof(host.ChannelKey(counterpartyCh.PortID, counterpartyCh.ID))

	msg := channeltypes.NewMsgChannelOpenConfirm(
		ch.PortID, ch.ID,
		proof, height,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

// ChanCloseInit will construct and execute a MsgChannelCloseInit.
//
// NOTE: does not work with ibc-transfer module
func (chain *TestChain) ChanCloseInit(
	counterparty TestChainI,
	channel TestChannel,
) error {
	msg := channeltypes.NewMsgChannelCloseInit(
		channel.PortID, channel.ID,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

// GetPacketData returns a ibc-transfer marshalled packet to be used for
// callback testing.
func (chain *TestChain) GetPacketData(counterparty TestChainI) []byte {
	packet := ibctransfertypes.FungibleTokenPacketData{
		Denom:    TestCoin.Denom,
		Amount:   TestCoin.Amount.Uint64(),
		Sender:   chain.SenderAccount.GetAddress().String(),
		Receiver: counterparty.GetSenderAccount().GetAddress().String(),
	}

	return packet.GetBytes()
}

// SendPacket simulates sending a packet through the channel keeper. No message needs to be
// passed since this call is made from a module.
func (chain *TestChain) SendPacket(
	packet exported.PacketI,
) error {
	ctx, writer := chain.GetContext().CacheContext()
	channelCap := chain.GetChannelCapability(packet.GetSourcePort(), packet.GetSourceChannel())

	// no need to send message, acting as a module
	err := chain.App.IBCKeeper.ChannelKeeper.SendPacket(ctx, channelCap, packet)
	if err != nil {
		return err
	}

	// commit changes
	writer()
	chain.NextBlock()

	return nil
}

// WriteAcknowledgement simulates writing an acknowledgement to the chain.
func (chain *TestChain) WriteAcknowledgement(
	packet exported.PacketI,
) error {
	channelCap := chain.GetChannelCapability(packet.GetDestPort(), packet.GetDestChannel())

	// no need to send message, acting as a handler
	err := chain.App.IBCKeeper.ChannelKeeper.WriteAcknowledgement(chain.GetContext(), channelCap, packet, TestHash)
	if err != nil {
		return err
	}

	// commit changes
	// FIXME how do we implement commit?
	// chain.App.Commit()
	chain.NextBlock()

	return nil
}

// GetPrefix returns the prefix for used by a chain in connection creation
func (chain *TestChain) GetPrefix() commitmenttypes.MerklePrefix {
	return commitmenttypes.NewMerklePrefix(chain.App.IBCKeeper.ConnectionKeeper.GetCommitmentPrefix().Bytes())
}

// sendMsgs delivers a transaction through the application without returning the result.
func (chain *TestChain) sendMsgs(msgs ...sdk.Msg) error {
	_, err := chain.SendMsgs(msgs...)
	return err
}

// SendMsgs delivers a transaction through the application. It updates the senders sequence
// number and updates the TestChain's headers. It returns the result and error if one
// occurred.
func (chain *TestChain) SendMsgs(msgs ...sdk.Msg) (*sdk.Result, error) {
	switch chain.txSignMode {
	case TxSignModeStdTx:
		return chain.sendMsgsWithStdTx(msgs...)
	case TxSignModeFabricTx:
		return chain.sendMsgsWithFabricTx(msgs...)
	default:
		return nil, fmt.Errorf("unknown txSignMode '%v'", chain.txSignMode)
	}
}

func (chain *TestChain) sendMsgsWithStdTx(msgs ...sdk.Msg) (*sdk.Result, error) {
	marshaler := codec.NewProtoCodec(chain.App.InterfaceRegistry())
	cfg := authtx.NewTxConfig(marshaler, []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})
	txBuilder := cfg.NewTxBuilder()
	require.NoError(chain.t, txBuilder.SetMsgs(msgs...))
	sig := signing.SignatureV2{
		PubKey: chain.SenderAccount.GetPubKey(),
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode_SIGN_MODE_DIRECT,
		},
		Sequence: chain.SenderAccount.GetSequence(),
	}
	require.NoError(chain.t, txBuilder.SetSignatures(sig))
	tx := txBuilder.GetTx()
	bz, err := cfg.TxJSONEncoder()(tx)
	require.NoError(chain.t, err)

	res, err := chain.CC.HandleTx(chain.GetFabricContext(), string(bz))
	require.NoError(chain.t, err)
	var events []abci.Event
	require.NoError(chain.t, json.Unmarshal([]byte(res.Events), &events))

	// increment sequence for successful transaction execution
	chain.SenderAccount.SetSequence(chain.SenderAccount.GetSequence() + 1)
	return &sdk.Result{Data: []byte(res.Data), Log: res.Log, Events: events}, nil
}

func (chain *TestChain) sendMsgsWithFabricTx(msgs ...sdk.Msg) (*sdk.Result, error) {
	marshaler := codec.NewProtoCodec(chain.App.InterfaceRegistry())
	cfg := authtx.NewTxConfig(marshaler, []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})
	txBuilder := cfg.NewTxBuilder()
	require.NoError(chain.t, txBuilder.SetMsgs(msgs...))
	sig := signing.SignatureV2{
		PubKey: chain.SenderAccount.GetPubKey(),
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode_SIGN_MODE_DIRECT,
		},
		Sequence: chain.SenderAccount.GetSequence(),
	}
	require.NoError(chain.t, txBuilder.SetSignatures(sig))
	tx := txBuilder.GetTx()
	bz, err := cfg.TxJSONEncoder()(tx)
	require.NoError(chain.t, err)

	chain.Stub.GetCreatorStub = func() ([]byte, error) {
		return proto.Marshal(getTestId())
	}

	res, err := chain.CC.HandleTx(chain.GetFabricContext(), string(bz))
	require.NoError(chain.t, err)
	var events []abci.Event
	require.NoError(chain.t, json.Unmarshal([]byte(res.Events), &events))
	return &sdk.Result{Data: []byte(res.Data), Log: res.Log, Events: events}, nil
}

// GetClientState retrieves the client state for the provided clientID. The client is
// expected to exist otherwise testing will fail.
func (chain *TestChain) GetClientState(clientID string) exported.ClientState {
	clientState, found := chain.App.IBCKeeper.ClientKeeper.GetClientState(chain.GetContext(), clientID)
	require.True(chain.t, found)

	return clientState
}

// GetConsensusState retrieves the consensus state for the provided clientID and height.
// It will return a success boolean depending on if consensus state exists or not.
func (chain *TestChain) GetConsensusState(clientID string, height exported.Height) (exported.ConsensusState, bool) {
	return chain.App.IBCKeeper.ClientKeeper.GetClientConsensusState(chain.GetContext(), clientID, height)
}

// GetConnection retrieves an IBC Connection for the provided TestConnection. The
// connection is expected to exist otherwise testing will fail.
func (chain *TestChain) GetConnection(testConnection *TestConnection) connectiontypes.ConnectionEnd {
	connection, found := chain.App.IBCKeeper.ConnectionKeeper.GetConnection(chain.GetContext(), testConnection.ID)
	require.True(chain.t, found)

	return connection
}

// GetChannel retrieves an IBC Channel for the provided TestChannel. The channel
// is expected to exist otherwise testing will fail.
func (chain *TestChain) GetChannel(testChannel TestChannel) channeltypes.Channel {
	channel, found := chain.App.IBCKeeper.ChannelKeeper.GetChannel(chain.GetContext(), testChannel.PortID, testChannel.ID)
	require.True(chain.t, found)

	return channel
}

// GetAcknowledgement retrieves an acknowledgement for the provided packet. If the
// acknowledgement does not exist then testing will fail.
func (chain *TestChain) GetAcknowledgement(packet exported.PacketI) []byte {
	ack, found := chain.App.IBCKeeper.ChannelKeeper.GetPacketAcknowledgement(chain.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	require.True(chain.t, found)

	return ack
}

// NewClientID appends a new clientID string in the format:
// ClientFor<counterparty-chain-id><index>
func (chain *TestChain) NewClientID(clientType string) string {
	clientID := fmt.Sprintf("%s-%s", clientType, strconv.Itoa(len(chain.ClientIDs)))
	chain.ClientIDs = append(chain.ClientIDs, clientID)
	return clientID
}

// ConstructMsgCreateClient constructs a message to create a new client state (tendermint or solomachine).
// NOTE: a solo machine client will be created with an empty diversifier.
func (chain *TestChain) ConstructMsgCreateClient(counterparty TestChainI, clientID string, clientType string) *clienttypes.MsgCreateClient {
	var (
		clientState    exported.ClientState
		consensusState exported.ConsensusState
	)

	switch clientType {
	case Tendermint:
		height := counterparty.GetLastHeader().GetHeight().(clienttypes.Height)
		clientState = ibctmtypes.NewClientState(
			counterparty.GetChainID(), DefaultTrustLevel, TrustingPeriod, UnbondingPeriod, MaxClockDrift,
			height, commitmenttypes.GetSDKSpecs(), UpgradePath, false, false,
		)
		consensusState = counterparty.GetLastHeader().ConsensusState()
	case SoloMachine:
		solo := ibctesting.NewSolomachine(chain.t, chain.Codec, clientID, "", 1)
		clientState = solo.ClientState()
		consensusState = solo.ConsensusState()
	case Fabric:
		clientState = chain.NewFabricClientState(counterparty, clientID)
		consensusState = chain.NewFabricConsensusState(counterparty)
	default:
		chain.t.Fatalf("unsupported client state type %s", clientType)
	}

	msg, err := clienttypes.NewMsgCreateClient(
		clientState, consensusState, chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(chain.t, err)
	return msg
}

// ExpireClient fast forwards the chain's block time by the provided amount of time which will
// expire any clients with a trusting period less than or equal to this amount of time.
func (chain *TestChain) ExpireClient(amount time.Duration) {
	chain.CurrentHeader.Time = chain.CurrentHeader.Time.Add(amount)
}

func (chain *TestChain) NewFabricConsensusState(counterparty TestChainI) *fabrictypes.ConsensusState {
	return &fabrictypes.ConsensusState{Timestamp: counterparty.GetApp().(*simapp.IBCApp).BlockProvider()().Timestamp()}
}

func (chain *TestChain) NewFabricClientState(counterparty TestChainI, clientID string) *fabrictypes.ClientState {
	mspID := "SampleOrgMSP"
	var pcBytes []byte = makePolicy([]string{mspID})

	block := counterparty.GetApp().(*simapp.IBCApp).BlockProvider()()

	mcBytes, err := proto.Marshal(&chain.mspConfig)
	require.NoError(chain.t, err)
	mhs := fabrictypes.NewMSPHeaders([]fabrictypes.MSPHeader{
		fabrictypes.NewMSPHeader(fabrictypes.MSPHeaderTypeCreate, mspID, mcBytes, pcBytes, &fabrictypes.MessageProof{}),
	})
	mspInfos, err := createMSPInitialClientState(mhs.Headers)
	require.NoError(chain.t, err)
	return &fabrictypes.ClientState{
		Id:                  clientID,
		LastChaincodeHeader: fabrictypes.NewChaincodeHeader(uint64(block.Height()), block.Timestamp(), fabrictypes.CommitmentProof{}),
		LastChaincodeInfo: fabrictypes.NewChaincodeInfo(
			chain.fabChannelID,
			chain.fabChaincodeID,
			pcBytes, pcBytes,
			&fabrictypes.MessageProof{},
		),
		LastMspInfos: *mspInfos,
	}
}

func createMSPInitialClientState(headers []fabrictypes.MSPHeader) (*fabrictypes.MSPInfos, error) {
	var infos fabrictypes.MSPInfos
	for _, mh := range headers {
		if mh.Type != fabrictypes.MSPHeaderTypeCreate {
			return nil, fmt.Errorf("unexpected fabric type: %v", mh.Type)
		}
		infos.Infos = append(infos.Infos, fabrictypes.MSPInfo{
			MSPID:   mh.MSPID,
			Config:  mh.Config,
			Policy:  mh.Policy,
			Freezed: false,
		})
	}
	return &infos, nil
}

func makePolicy(mspids []string) []byte {
	return protoutil.MarshalOrPanic(&common.ApplicationPolicy{
		Type: &common.ApplicationPolicy_SignaturePolicy{
			SignaturePolicy: policydsl.SignedByNOutOfGivenRole(int32(len(mspids)/2+1), msppb.MSPRole_MEMBER, mspids),
		},
	})
}

// WARNING !!! This certificate must be only used in test !!!
// This certificate is a copy of
// https://github.com/hyperledger/fabric/blob/665ace61890f79cb09060af0807e344c0d8f19d6/common/crypto/testdata/cert.pem
func getTestId() *msppb.SerializedIdentity {
	csr := `-----BEGIN CERTIFICATE-----
MIICCDCCAa6gAwIBAgIRANLH5Ue5a6tHuzCQtap1BP8wCgYIKoZIzj0EAwIwZzEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xEzARBgNVBAoTCmhybC5pYm0uaWwxFjAUBgNVBAMTDWNhLmhybC5p
Ym0uaWwwHhcNMTcwODE5MTIxOTQ4WhcNMjcwODE3MTIxOTQ4WjBVMQswCQYDVQQG
EwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZyYW5jaXNj
bzEZMBcGA1UEAwwQVXNlcjFAaHJsLmlibS5pbDBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABE7fF65KsF0nxNgIBFVA2x/QU0LuAyuTsRaSWc/ycQAuLQfCti5bYp4W
WaQUc5sBaKAmVbFQTm9RhmOhtIz7PL6jTTBLMA4GA1UdDwEB/wQEAwIHgDAMBgNV
HRMBAf8EAjAAMCsGA1UdIwQkMCKAIMjiBsyFZlbO6pRxo7VgoqKhl78Ujd9sdWUk
epB05fodMAoGCCqGSM49BAMCA0gAMEUCIQCiOzbaApF46NVobwh3wqHf8ID1zxja
j23HPXR3FjjFZgIgXLujyDGETptNrELaytjG+dxO3Kzq/SM07K2zPUg4368=
-----END CERTIFICATE-----`
	return &msppb.SerializedIdentity{
		IdBytes: []byte(csr),
	}
}

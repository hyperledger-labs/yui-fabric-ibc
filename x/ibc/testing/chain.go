package testing

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	connectiontypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/23-commitment/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/types"
	solomachinetypes "github.com/cosmos/cosmos-sdk/x/ibc/light-clients/06-solomachine/types"
	ibctmtypes "github.com/cosmos/cosmos-sdk/x/ibc/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/cosmos-sdk/x/ibc/testing"
	"github.com/cosmos/cosmos-sdk/x/ibc/testing/mock"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/datachainlab/fabric-ibc/app"
	"github.com/datachainlab/fabric-ibc/chaincode"
	"github.com/datachainlab/fabric-ibc/example"
	"github.com/datachainlab/fabric-ibc/x/compat"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/light-clients/xx-fabric/types"
)

const (
	// client types
	Tendermint  = ibctmtypes.Tendermint
	SoloMachine = solomachinetypes.SoloMachine
	Fabric      = fabrictypes.Fabric

	// Default params constants used to create a TM client
	TrustingPeriod  time.Duration = time.Hour * 24 * 7 * 2
	UnbondingPeriod time.Duration = time.Hour * 24 * 7 * 3
	MaxClockDrift   time.Duration = time.Second * 10

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

	UpgradePath = fmt.Sprintf("%s/%s", "upgrade", "upgradedClient")
)

type TestChainI interface {
	Type() string
	GetApp() interface{}
	GetChainID() string
	GetSenderAccount() authtypes.AccountI
	GetLastHeader() *ibctmtypes.Header
	NextBlock()
	GetContext() sdk.Context

	AddTestConnection(clientID, counterpartyClientID string) *ibctesting.TestConnection
	ConstructNextTestConnection(clientID, counterpartyClientID string) *ibctesting.TestConnection

	GetChannel(testChannel ibctesting.TestChannel) channeltypes.Channel

	ConnectionOpenInit(
		counterparty TestChainI,
		connection, counterpartyConnection *ibctesting.TestConnection,
	) error
	ConnectionOpenTry(
		counterparty TestChainI,
		connection, counterpartyConnection *ibctesting.TestConnection,
	) error
	ConnectionOpenAck(
		counterparty TestChainI,
		connection, counterpartyConnection *ibctesting.TestConnection,
	) error
	ConnectionOpenConfirm(
		counterparty TestChainI,
		connection, counterpartyConnection *ibctesting.TestConnection,
	) error

	ChanOpenInit(
		ch, counterparty ibctesting.TestChannel,
		order channeltypes.Order,
		connectionID string,
	) error
	ChanOpenTry(
		counterparty TestChainI,
		ch, counterpartyCh ibctesting.TestChannel,
		order channeltypes.Order,
		connectionID string,
	) error
	ChanOpenAck(
		counterparty TestChainI,
		ch, counterpartyCh ibctesting.TestChannel,
	) error
	ChanOpenConfirm(
		counterparty TestChainI,
		ch, counterpartyCh ibctesting.TestChannel,
	) error
	ChanCloseInit(
		counterparty TestChainI,
		channel ibctesting.TestChannel,
	) error

	CreatePortCapability(portID string)

	QueryClientStateProof(clientID string) (exported.ClientState, []byte)
	QueryProof(key []byte) ([]byte, clienttypes.Height)
	QueryConsensusStateProof(clientID string) ([]byte, clienttypes.Height)

	NewClientID(counterpartyChainID string) string
	GetPrefix() commitmenttypes.MerklePrefix

	sendMsgs(msgs ...sdk.Msg) error
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

// TestChain is a testing struct that wraps a simapp with the last TM Header, the current ABCI
// header and the validators of the TestChain. It also contains a field called ChainID. This
// is the clientID that *other* chains use to refer to this TestChain. The SenderAccount
// is used for delivering transactions through the application state.
// NOTE: the actual application uses an empty chain-id for ease of testing.
type TestChain struct {
	t *testing.T

	App  *example.IBCApp
	CC   *chaincode.IBCChaincode
	Stub shim.ChaincodeStubInterface

	ChainID       string
	LastHeader    *ibctmtypes.Header // header for last block height committed
	CurrentHeader tmproto.Header     // header for current block height
	QueryServer   types.QueryServer
	TxConfig      client.TxConfig
	Codec         codec.BinaryMarshaler

	Vals    *tmtypes.ValidatorSet
	Signers []tmtypes.PrivValidator

	senderPrivKey crypto.PrivKey
	SenderAccount authtypes.AccountI

	// IBC specific helpers
	ClientIDs   []string                     // ClientID's used on this chain
	Connections []*ibctesting.TestConnection // track connectionID's created for this chain

	NextChannelVersion string
}

var _ TestChainI = (*TestChain)(nil)

func NewTestFabricChain(t *testing.T, chainID string) *TestChain {
	// generate validator private/public key
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)

	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey.(cryptotypes.IntoTmPubKey).AsTmPubKey(), 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})
	signers := []tmtypes.PrivValidator{privVal}

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100000000000000))),
	}
	_ = balance
	cc := chaincode.NewIBCChaincode(newApp, chaincode.DefaultDBProvider)
	runner := cc.GetAppRunner()
	stub := compat.MakeFakeStub()
	app, err := newApp(
		tmlog.NewTMLogger(os.Stdout),
		compat.NewDB(stub),
		nil,
		runner.GetBlockProvider(stub),
	)
	require.NoError(t, err)

	// create current header and call begin block
	header := tmproto.Header{
		ChainID: chainID,
		Height:  1,
		Time:    globalStartTime,
	}

	txConfig := simapp.MakeTestEncodingConfig().TxConfig

	// create an account to send transactions from
	chain := &TestChain{
		t:                  t,
		ChainID:            chainID,
		App:                app.(*example.IBCApp),
		CC:                 cc,
		Stub:               stub,
		CurrentHeader:      header,
		QueryServer:        app.GetIBCKeeper(),
		TxConfig:           txConfig,
		Codec:              app.AppCodec(),
		Vals:               valSet,
		Signers:            signers,
		senderPrivKey:      senderPrivKey,
		SenderAccount:      acc,
		ClientIDs:          make([]string, 0),
		Connections:        make([]*ibctesting.TestConnection, 0),
		NextChannelVersion: ChannelTransferVersion,
	}

	stub.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: time.Now().Unix()}, nil)
	var tctx contractapi.TransactionContext
	tctx.SetStub(stub)

	// Init chaincode
	require.NoError(t, chain.CC.InitChaincode(&tctx, "{}"))

	// cap := chain.App.IBCKeeper.PortKeeper.BindPort(chain.GetContext(), MockPort)
	// pp.Println(chain.App.ScopedIBCMockKeeper)
	// err = chain.App.ScopedIBCMockKeeper.ClaimCapability(chain.GetContext(), cap, host.PortPath(MockPort))
	// require.NoError(t, err)

	chain.NextBlock()

	return chain
}

// Type implements TestChainI.Type
func (chain TestChain) Type() string {
	return Fabric
}

func (chain *TestChain) GetContext() sdk.Context {
	ctx, _ := chain.App.MakeCacheContext(chain.CurrentHeader)
	return ctx
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
	res := chain.App.Query(abci.RequestQuery{
		Path: fmt.Sprintf("store/%s/key", host.StoreKey),
		// Height: chain.App.LastBlockHeight() - 1,
		Data:  key,
		Prove: true,
	})

	merkleProof := commitmenttypes.MerkleProof{
		Proof: res.ProofOps,
	}

	proof, err := chain.App.AppCodec().MarshalBinaryBare(&merkleProof)
	require.NoError(chain.t, err)

	version := clienttypes.ParseChainID(chain.ChainID)

	// proof height + 1 is returned as the proof created corresponds to the height the proof
	// was created in the IAVL tree. Tendermint and subsequently the clients that rely on it
	// have heights 1 above the IAVL tree. Thus we return proof height + 1
	return proof, clienttypes.NewHeight(version, uint64(res.Height)+1)
}

// QueryClientStateProof performs and abci query for a client state
// stored with a given clientID and returns the ClientState along with the proof
func (chain *TestChain) QueryClientStateProof(clientID string) (exported.ClientState, []byte) {
	// retrieve client state to provide proof for
	clientState, found := chain.App.IBCKeeper.ClientKeeper.GetClientState(chain.GetContext(), clientID)
	require.True(chain.t, found)

	clientKey := host.FullKeyClientPath(clientID, host.KeyClientState())
	proofClient, _ := chain.QueryProof(clientKey)

	return clientState, proofClient
}

// QueryConsensusStateProof performs an abci query for a consensus state
// stored on the given clientID. The proof and consensusHeight are returned.
func (chain *TestChain) QueryConsensusStateProof(clientID string) ([]byte, clienttypes.Height) {
	clientState := chain.GetClientState(clientID)

	consensusHeight := clientState.GetLatestHeight().(clienttypes.Height)
	consensusKey := host.FullKeyClientPath(clientID, host.KeyConsensusState(consensusHeight))
	proofConsensus, _ := chain.QueryProof(consensusKey)

	return proofConsensus, consensusHeight
}

// AddTestConnection appends a new TestConnection which contains references
// to the connection id, client id and counterparty client id.
func (chain *TestChain) AddTestConnection(clientID, counterpartyClientID string) *ibctesting.TestConnection {
	conn := chain.ConstructNextTestConnection(clientID, counterpartyClientID)

	chain.Connections = append(chain.Connections, conn)
	return conn
}

// ConstructNextTestConnection constructs the next test connection to be
// created given a clientID and counterparty clientID. The connection id
// format: <chainID>-conn<index>
func (chain *TestChain) ConstructNextTestConnection(clientID, counterpartyClientID string) *ibctesting.TestConnection {
	connectionID := fmt.Sprintf("%s-%s%d", chain.ChainID, ConnectionIDPrefix, len(chain.Connections))
	return &ibctesting.TestConnection{
		ID:                   connectionID,
		ClientID:             clientID,
		NextChannelVersion:   chain.NextChannelVersion,
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

func (chain *TestChain) UpdateClient(counterparty TestChainI, clientID string) error {
	if counterparty.Type() != Fabric {
		panic("not implemented error")
	}

	header, err := chain.ConstructUpdateClientHeader(counterparty, clientID)
	require.NoError(chain.t, err)

	msg, err := clienttypes.NewMsgUpdateClient(
		clientID, header,
		chain.SenderAccount.GetAddress(),
	)
	require.NoError(chain.t, err)

	return chain.sendMsgs(msg)
}

func (chain *TestChain) ConstructUpdateClientHeader(counterparty TestChainI, clientID string) (*ibctmtypes.Header, error) {
	// TODO implements
	panic("not implemented error")
}

// ConnectionOpenInit will construct and execute a MsgConnectionOpenInit.
func (chain *TestChain) ConnectionOpenInit(
	counterparty TestChainI,
	connection, counterpartyConnection *ibctesting.TestConnection,
) error {
	msg := connectiontypes.NewMsgConnectionOpenInit(
		connection.ID, connection.ClientID,
		counterpartyConnection.ID, connection.CounterpartyClientID,
		counterparty.GetPrefix(), DefaultOpenInitVersion,
		chain.SenderAccount.GetAddress(),
	)
	return chain.sendMsgs(msg)
}

// ConnectionOpenTry will construct and execute a MsgConnectionOpenTry.
func (chain *TestChain) ConnectionOpenTry(
	counterparty TestChainI,
	connection, counterpartyConnection *ibctesting.TestConnection,
) error {
	counterpartyClient, proofClient := counterparty.QueryClientStateProof(counterpartyConnection.ClientID)

	connectionKey := host.KeyConnection(counterpartyConnection.ID)
	proofInit, proofHeight := counterparty.QueryProof(connectionKey)

	proofConsensus, consensusHeight := counterparty.QueryConsensusStateProof(counterpartyConnection.ClientID)

	msg := connectiontypes.NewMsgConnectionOpenTry(
		connection.ID, connection.ID, connection.ClientID, // testing doesn't use flexible selection
		counterpartyConnection.ID, counterpartyConnection.ClientID,
		counterpartyClient, counterparty.GetPrefix(), []*connectiontypes.Version{ConnectionVersion},
		proofInit, proofClient, proofConsensus,
		proofHeight, consensusHeight,
		chain.SenderAccount.GetAddress(),
	)
	return chain.sendMsgs(msg)
}

// ConnectionOpenAck will construct and execute a MsgConnectionOpenAck.
func (chain *TestChain) ConnectionOpenAck(
	counterparty TestChainI,
	connection, counterpartyConnection *ibctesting.TestConnection,
) error {
	counterpartyClient, proofClient := counterparty.QueryClientStateProof(counterpartyConnection.ClientID)

	connectionKey := host.KeyConnection(counterpartyConnection.ID)
	proofTry, proofHeight := counterparty.QueryProof(connectionKey)

	proofConsensus, consensusHeight := counterparty.QueryConsensusStateProof(counterpartyConnection.ClientID)

	msg := connectiontypes.NewMsgConnectionOpenAck(
		connection.ID, counterpartyConnection.ID, counterpartyClient, // testing doesn't use flexible selection
		proofTry, proofClient, proofConsensus,
		proofHeight, consensusHeight,
		ConnectionVersion,
		chain.SenderAccount.GetAddress(),
	)
	return chain.sendMsgs(msg)
}

// ConnectionOpenConfirm will construct and execute a MsgConnectionOpenConfirm.
func (chain *TestChain) ConnectionOpenConfirm(
	counterparty TestChainI,
	connection, counterpartyConnection *ibctesting.TestConnection,
) error {
	connectionKey := host.KeyConnection(counterpartyConnection.ID)
	proof, height := counterparty.QueryProof(connectionKey)

	msg := connectiontypes.NewMsgConnectionOpenConfirm(
		connection.ID,
		proof, height,
		chain.SenderAccount.GetAddress(),
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
	ch, counterparty ibctesting.TestChannel,
	order channeltypes.Order,
	connectionID string,
) error {
	msg := channeltypes.NewMsgChannelOpenInit(
		ch.PortID, ch.ID,
		ch.Version, order, []string{connectionID},
		counterparty.PortID, counterparty.ID,
		chain.SenderAccount.GetAddress(),
	)
	return chain.sendMsgs(msg)
}

// ChanOpenTry will construct and execute a MsgChannelOpenTry.
func (chain *TestChain) ChanOpenTry(
	counterparty TestChainI,
	ch, counterpartyCh ibctesting.TestChannel,
	order channeltypes.Order,
	connectionID string,
) error {
	proof, height := counterparty.QueryProof(host.KeyChannel(counterpartyCh.PortID, counterpartyCh.ID))

	msg := channeltypes.NewMsgChannelOpenTry(
		ch.PortID, ch.ID, ch.ID, // testing doesn't use flexible selection
		ch.Version, order, []string{connectionID},
		counterpartyCh.PortID, counterpartyCh.ID,
		counterpartyCh.Version,
		proof, height,
		chain.SenderAccount.GetAddress(),
	)
	return chain.sendMsgs(msg)
}

// ChanOpenAck will construct and execute a MsgChannelOpenAck.
func (chain *TestChain) ChanOpenAck(
	counterparty TestChainI,
	ch, counterpartyCh ibctesting.TestChannel,
) error {
	proof, height := counterparty.QueryProof(host.KeyChannel(counterpartyCh.PortID, counterpartyCh.ID))

	msg := channeltypes.NewMsgChannelOpenAck(
		ch.PortID, ch.ID,
		counterpartyCh.ID, counterpartyCh.Version, // testing doesn't use flexible selection
		proof, height,
		chain.SenderAccount.GetAddress(),
	)
	return chain.sendMsgs(msg)
}

// ChanOpenConfirm will construct and execute a MsgChannelOpenConfirm.
func (chain *TestChain) ChanOpenConfirm(
	counterparty TestChainI,
	ch, counterpartyCh ibctesting.TestChannel,
) error {
	proof, height := counterparty.QueryProof(host.KeyChannel(counterpartyCh.PortID, counterpartyCh.ID))

	msg := channeltypes.NewMsgChannelOpenConfirm(
		ch.PortID, ch.ID,
		proof, height,
		chain.SenderAccount.GetAddress(),
	)
	return chain.sendMsgs(msg)
}

// ChanCloseInit will construct and execute a MsgChannelCloseInit.
//
// NOTE: does not work with ibc-transfer module
func (chain *TestChain) ChanCloseInit(
	counterparty TestChainI,
	channel ibctesting.TestChannel,
) error {
	msg := channeltypes.NewMsgChannelCloseInit(
		channel.PortID, channel.ID,
		chain.SenderAccount.GetAddress(),
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
	// TODO use fabric-ibc auth tx instead of default

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

	_, err = chain.CC.GetAppRunner().RunMsg(chain.Stub, bz)
	if err != nil {
		return nil, err
	}

	// increment sequence for successful transaction execution
	chain.SenderAccount.SetSequence(chain.SenderAccount.GetSequence() + 1)
	return nil, nil
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
func (chain *TestChain) GetConnection(testConnection *ibctesting.TestConnection) connectiontypes.ConnectionEnd {
	connection, found := chain.App.IBCKeeper.ConnectionKeeper.GetConnection(chain.GetContext(), testConnection.ID)
	require.True(chain.t, found)

	return connection
}

// GetChannel retrieves an IBC Channel for the provided TestChannel. The channel
// is expected to exist otherwise testing will fail.
func (chain *TestChain) GetChannel(testChannel ibctesting.TestChannel) channeltypes.Channel {
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
func (chain *TestChain) NewClientID(counterpartyChainID string) string {
	clientID := "client" + strconv.Itoa(len(chain.ClientIDs)) + "For" + counterpartyChainID
	chain.ClientIDs = append(chain.ClientIDs, clientID)
	return clientID
}

// GetFirstTestConnection returns the first test connection for a given clientID.
// The connection may or may not exist in the chain state.
func (chain *TestChain) GetFirstTestConnection(clientID, counterpartyClientID string) *ibctesting.TestConnection {
	if len(chain.Connections) > 0 {
		return chain.Connections[0]
	}

	return chain.ConstructNextTestConnection(clientID, counterpartyClientID)
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
			height, counterparty.GetApp().(simapp.SimApp).GetConsensusParams(counterparty.GetContext()), commitmenttypes.GetSDKSpecs(),
			UpgradePath, false, false,
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
		clientID, clientState, consensusState, chain.SenderAccount.GetAddress(),
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
	return &fabrictypes.ConsensusState{Timestamp: counterparty.GetApp().(*example.IBCApp).BlockProvider()().Timestamp()}
}

func (chain *TestChain) NewFabricClientState(counterparty TestChainI, clientID string) *fabrictypes.ClientState {
	block := counterparty.GetApp().(*example.IBCApp).BlockProvider()()
	return &fabrictypes.ClientState{
		Id:                  clientID,
		LastChaincodeHeader: fabrictypes.NewChaincodeHeader(uint64(block.Height()), block.Timestamp(), fabrictypes.CommitmentProof{}),
	}
}

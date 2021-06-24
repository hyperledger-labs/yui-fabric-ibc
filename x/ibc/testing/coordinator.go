package testing

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/testing/simapp"
	"github.com/datachainlab/fabric-ibc/example"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ChainIDPrefix   = "testchain"
	globalStartTime = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	timeIncrement   = time.Second * 5
)

type Coordinator struct {
	t *testing.T

	Chains map[string]TestChainI
}

func NewCoordinator(t *testing.T, n int, mspID string, txSignMode TxSignMode) *Coordinator {
	chains := make(map[string]TestChainI)

	for i := 0; i < n; i++ {
		chainID := GetChainID(i)
		chains[chainID] = NewTestFabricChain(t, chainID, mspID, txSignMode)
	}
	return &Coordinator{
		t:      t,
		Chains: chains,
	}
}

// GetChainID returns the chainID used for the provided index.
func GetChainID(index int) string {
	return ChainIDPrefix + strconv.Itoa(index)
}

// Setup constructs a TM client, connection, and channel on both chains provided. It will
// fail if any error occurs. The clientID's, TestConnections, and TestChannels are returned
// for both chains. The channels created are connected to the ibc-transfer application.
func (coord *Coordinator) Setup(
	chainA, chainB TestChainI, order channeltypes.Order,
) (string, string, *TestConnection, *TestConnection, TestChannel, TestChannel) {
	clientA, clientB, connA, connB := coord.SetupClientConnections(chainA, chainB, Fabric)

	// channels can also be referenced through the returned connections
	channelA, channelB := coord.CreateMockChannels(chainA, chainB, connA, connB, order)

	return clientA, clientB, connA, connB, channelA, channelB
}

// SetupClientConnections is a helper function to create clients and the appropriate
// connections on both the source and counterparty chain. It assumes the caller does not
// anticipate any errors.
func (coord *Coordinator) SetupClientConnections(
	chainA, chainB TestChainI,
	clientType string,
) (string, string, *TestConnection, *TestConnection) {

	clientA, clientB := coord.SetupClients(chainA, chainB, clientType)

	connA, connB := coord.CreateConnection(chainA, chainB, clientA, clientB, ChannelTransferVersion)

	return clientA, clientB, connA, connB
}

// SetupClients is a helper function to create clients on both chains. It assumes the
// caller does not anticipate any errors.
func (coord *Coordinator) SetupClients(
	chainA, chainB TestChainI,
	clientType string,
) (string, string) {

	clientA, err := coord.CreateClient(chainA, chainB, clientType)
	require.NoError(coord.t, err)

	clientB, err := coord.CreateClient(chainB, chainA, clientType)
	require.NoError(coord.t, err)

	return clientA, clientB
}

// CreateClient creates a counterparty client on the source chain and returns the clientID.
func (coord *Coordinator) CreateClient(
	source, counterparty TestChainI,
	clientType string,
) (clientID string, err error) {
	coord.CommitBlock(source, counterparty)

	clientID = source.NewClientID(clientType)

	switch clientType {
	// case Tendermint:
	// 	err = source.(*TestChain).CreateTMClient(counterparty, clientID)
	// case SoloMachine:
	// err = source.(*TestSolomachineChain).CreateClient(counterparty, clientID)
	case Fabric:
		err = source.(*TestChain).CreateClient(counterparty, clientID)
	default:
		err = fmt.Errorf("client type %s is not supported", clientType)
	}

	if err != nil {
		return "", err
	}

	coord.IncrementTime()

	return clientID, nil
}

// UpdateClient updates a counterparty client on the source chain.
func (coord *Coordinator) UpdateClient(
	source, counterparty TestChainI,
	clientID string,
	clientType string,
) (err error) {
	coord.CommitBlock(source, counterparty)

	switch clientType {
	// case Tendermint:
	// 	err = source.UpdateTMClient(counterparty, clientID)
	// case SoloMachine:
	// err = source.(*TestSolomachineChain).UpdateClient(counterparty, clientID)
	case Fabric:
		err = source.(*TestChain).UpdateClient(counterparty, clientID)
	default:
		err = fmt.Errorf("client type %s is not supported", clientType)
	}

	if err != nil {
		return err
	}

	coord.IncrementTime()

	return nil
}

// CreateConnection constructs and executes connection handshake messages in order to create
// OPEN channels on chainA and chainB. The connection information of for chainA and chainB
// are returned within a TestConnection struct. The function expects the connections to be
// successfully opened otherwise testing will fail.
func (coord *Coordinator) CreateConnection(
	chainA, chainB TestChainI,
	clientA, clientB string,
	nextChannelVersion string,
) (*TestConnection, *TestConnection) {

	connA, connB, err := coord.ConnOpenInit(chainA, chainB, clientA, clientB, nextChannelVersion)
	require.NoError(coord.t, err)

	err = coord.ConnOpenTry(chainB, chainA, connB, connA)
	require.NoError(coord.t, err)

	err = coord.ConnOpenAck(chainA, chainB, connA, connB)
	require.NoError(coord.t, err)

	err = coord.ConnOpenConfirm(chainB, chainA, connB, connA)
	require.NoError(coord.t, err)

	return connA, connB
}

// CreateTransferChannels constructs and executes channel handshake messages to create OPEN
// ibc-transfer channels on chainA and chainB. The function expects the channels to be
// successfully opened otherwise testing will fail.
func (coord *Coordinator) CreateTransferChannels(
	chainA, chainB TestChainI,
	connA, connB *TestConnection,
	order channeltypes.Order,
) (TestChannel, TestChannel) {
	return coord.CreateChannel(chainA, chainB, connA, connB, TransferPort, TransferPort, order)
}

// CreateChannel constructs and executes channel handshake messages in order to create
// OPEN channels on chainA and chainB. The function expects the channels to be successfully
// opened otherwise testing will fail.
func (coord *Coordinator) CreateChannel(
	chainA, chainB TestChainI,
	connA, connB *TestConnection,
	sourcePortID, counterpartyPortID string,
	order channeltypes.Order,
) (TestChannel, TestChannel) {
	channelA, channelB, err := coord.ChanOpenInit(chainA, chainB, connA, connB, sourcePortID, counterpartyPortID, order)
	require.NoError(coord.t, err)

	err = coord.ChanOpenTry(chainB, chainA, channelB, channelA, connB, order)
	require.NoError(coord.t, err)

	err = coord.ChanOpenAck(chainA, chainB, channelA, channelB)
	require.NoError(coord.t, err)

	err = coord.ChanOpenConfirm(chainB, chainA, channelB, channelA)
	require.NoError(coord.t, err)

	return channelA, channelB
}

// CreateMockChannels constructs and executes channel handshake messages to create OPEN
// channels that use a mock application module that returns nil on all callbacks. This
// function is expects the channels to be successfully opened otherwise testing will
// fail.
func (coord *Coordinator) CreateMockChannels(
	chainA, chainB TestChainI,
	connA, connB *TestConnection,
	order channeltypes.Order,
) (TestChannel, TestChannel) {
	return coord.CreateChannel(chainA, chainB, connA, connB, MockPort, MockPort, order)
}

// ConnOpenInit initializes a connection on the source chain with the state INIT
// using the OpenInit handshake call.
//
// NOTE: The counterparty testing connection will be created even if it is not created in the
// application state.
func (coord *Coordinator) ConnOpenInit(
	source, counterparty TestChainI,
	clientID, counterpartyClientID string, nextChannelVersion string,
) (*TestConnection, *TestConnection, error) {
	sourceConnection := source.AddTestConnection(clientID, counterpartyClientID, nextChannelVersion)
	counterpartyConnection := counterparty.AddTestConnection(counterpartyClientID, clientID, nextChannelVersion)

	// initialize connection on source
	if err := source.ConnectionOpenInit(counterparty, sourceConnection, counterpartyConnection); err != nil {
		return sourceConnection, counterpartyConnection, err
	}
	coord.IncrementTime()

	// update source client on counterparty connection
	if err := coord.UpdateClient(
		counterparty, source,
		counterpartyClientID, counterparty.Type(),
	); err != nil {
		return sourceConnection, counterpartyConnection, err
	}

	return sourceConnection, counterpartyConnection, nil
}

// ConnOpenTry initializes a connection on the source chain with the state TRYOPEN
// using the OpenTry handshake call.
func (coord *Coordinator) ConnOpenTry(
	source, counterparty TestChainI,
	sourceConnection, counterpartyConnection *TestConnection,
) error {
	// initialize TRYOPEN connection on source
	if err := source.ConnectionOpenTry(counterparty, sourceConnection, counterpartyConnection); err != nil {
		return err
	}
	coord.IncrementTime()

	// update source client on counterparty connection
	return coord.UpdateClient(
		counterparty, source,
		counterpartyConnection.ClientID, counterparty.Type(),
	)
}

// ConnOpenAck initializes a connection on the source chain with the state OPEN
// using the OpenAck handshake call.
func (coord *Coordinator) ConnOpenAck(
	source, counterparty TestChainI,
	sourceConnection, counterpartyConnection *TestConnection,
) error {
	// set OPEN connection on source using OpenAck
	if err := source.ConnectionOpenAck(counterparty, sourceConnection, counterpartyConnection); err != nil {
		return err
	}
	coord.IncrementTime()

	// update source client on counterparty connection
	return coord.UpdateClient(
		counterparty, source,
		counterpartyConnection.ClientID, counterparty.Type(),
	)
}

// ConnOpenConfirm initializes a connection on the source chain with the state OPEN
// using the OpenConfirm handshake call.
func (coord *Coordinator) ConnOpenConfirm(
	source, counterparty TestChainI,
	sourceConnection, counterpartyConnection *TestConnection,
) error {
	if err := source.ConnectionOpenConfirm(counterparty, sourceConnection, counterpartyConnection); err != nil {
		return err
	}
	coord.IncrementTime()

	// update source client on counterparty connection
	return coord.UpdateClient(
		counterparty, source,
		counterpartyConnection.ClientID, counterparty.Type(),
	)
}

// ChanOpenInit initializes a channel on the source chain with the state INIT
// using the OpenInit handshake call.
//
// NOTE: The counterparty testing channel will be created even if it is not created in the
// application state.
func (coord *Coordinator) ChanOpenInit(
	source, counterparty TestChainI,
	connection, counterpartyConnection *TestConnection,
	sourcePortID, counterpartyPortID string,
	order channeltypes.Order,
) (TestChannel, TestChannel, error) {
	sourceChannel := source.AddTestChannel(connection, sourcePortID)
	counterpartyChannel := counterparty.AddTestChannel(counterpartyConnection, counterpartyPortID)

	// NOTE: only creation of a capability for a transfer or mock port is supported
	// Other applications must bind to the port in InitGenesis or modify this code.
	source.CreatePortCapability(sourceChannel.PortID)
	coord.IncrementTime()

	// initialize channel on source
	if err := source.ChanOpenInit(sourceChannel, counterpartyChannel, order, connection.ID); err != nil {
		return sourceChannel, counterpartyChannel, err
	}
	coord.IncrementTime()

	// update source client on counterparty connection
	if err := coord.UpdateClient(
		counterparty, source,
		counterpartyConnection.ClientID, counterparty.Type(),
	); err != nil {
		return sourceChannel, counterpartyChannel, err
	}

	return sourceChannel, counterpartyChannel, nil
}

// ChanOpenTry initializes a channel on the source chain with the state TRYOPEN
// using the OpenTry handshake call.
func (coord *Coordinator) ChanOpenTry(
	source, counterparty TestChainI,
	sourceChannel, counterpartyChannel TestChannel,
	connection *TestConnection,
	order channeltypes.Order,
) error {
	// create a capability
	source.CreatePortCapability(sourceChannel.PortID)

	// initialize channel on source
	if err := source.ChanOpenTry(counterparty, sourceChannel, counterpartyChannel, order, connection.ID); err != nil {
		return err
	}
	coord.IncrementTime()

	// update source client on counterparty connection
	return coord.UpdateClient(
		counterparty, source,
		connection.CounterpartyClientID, counterparty.Type(),
	)
}

// ChanOpenAck initializes a channel on the source chain with the state OPEN
// using the OpenAck handshake call.
func (coord *Coordinator) ChanOpenAck(
	source, counterparty TestChainI,
	sourceChannel, counterpartyChannel TestChannel,
) error {

	if err := source.ChanOpenAck(counterparty, sourceChannel, counterpartyChannel); err != nil {
		return err
	}
	coord.IncrementTime()

	// update source client on counterparty connection
	return coord.UpdateClient(
		counterparty, source,
		sourceChannel.CounterpartyClientID, counterparty.Type(),
	)
}

// ChanOpenConfirm initializes a channel on the source chain with the state OPEN
// using the OpenConfirm handshake call.
func (coord *Coordinator) ChanOpenConfirm(
	source, counterparty TestChainI,
	sourceChannel, counterpartyChannel TestChannel,
) error {

	if err := source.ChanOpenConfirm(counterparty, sourceChannel, counterpartyChannel); err != nil {
		return err
	}
	coord.IncrementTime()

	// update source client on counterparty connection
	return coord.UpdateClient(
		counterparty, source,
		sourceChannel.CounterpartyClientID, counterparty.Type(),
	)
}

// ChanCloseInit closes a channel on the source chain resulting in the channels state
// being set to CLOSED.
//
// NOTE: does not work with ibc-transfer module
func (coord *Coordinator) ChanCloseInit(
	source, counterparty TestChainI,
	channel TestChannel,
) error {

	if err := source.ChanCloseInit(counterparty, channel); err != nil {
		return err
	}
	coord.IncrementTime()

	// update source client on counterparty connection
	return coord.UpdateClient(
		counterparty, source,
		channel.CounterpartyClientID, counterparty.Type(),
	)
}

// SetChannelClosed sets a channel state to CLOSED.
func (coord *Coordinator) SetChannelClosed(
	source, counterparty TestChainI,
	testChannel TestChannel,
) error {
	channel := source.GetChannel(testChannel)

	channel.State = channeltypes.CLOSED

	switch source.Type() {
	case Tendermint:
		source.GetApp().(*simapp.SimApp).IBCKeeper.ChannelKeeper.SetChannel(source.GetContext(), testChannel.PortID, testChannel.ID, channel)
	case Fabric:
		source.GetApp().(*example.IBCApp).IBCKeeper.ChannelKeeper.SetChannel(source.GetContext(), testChannel.PortID, testChannel.ID, channel)
	}

	coord.CommitBlock(source)

	// update source client on counterparty connection
	return coord.UpdateClient(
		counterparty, source,
		testChannel.CounterpartyClientID, counterparty.Type(),
	)
}

// RelayPacket receives a channel packet on counterparty, queries the ack
// and acknowledges the packet on source. The clients are updated as needed.
func (coord *Coordinator) RelayPacket(
	source, counterparty TestChainI,
	sourceClient, counterpartyClient string,
	packet channeltypes.Packet, ack []byte,
) error {
	if err := coord.RecvPacket(source, counterparty, sourceClient, packet); err != nil {
		return err
	}

	return coord.AcknowledgePacket(source, counterparty, counterpartyClient, packet, ack)
}

// RecvPacket receives a channel packet on the counterparty chain and updates
// the client on the source chain representing the counterparty.
func (coord *Coordinator) RecvPacket(
	source, counterparty TestChainI,
	sourceClient string,
	packet channeltypes.Packet,
) error {
	// get proof of packet commitment on source
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := source.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, counterparty.GetSenderAccount().GetAddress().String())

	// receive on counterparty and update source client
	return coord.SendMsgs(counterparty, source, sourceClient, []sdk.Msg{recvMsg})
}

// AcknowledgePacket acknowledges on the source chain the packet received on
// the counterparty chain and updates the client on the counterparty representing
// the source chain.
// TODO: add a query for the acknowledgement by events
// - https://github.com/cosmos/cosmos-sdk/issues/6509
func (coord *Coordinator) AcknowledgePacket(
	source, counterparty TestChainI,
	counterpartyClient string,
	packet channeltypes.Packet, ack []byte,
) error {
	// get proof of acknowledgement on counterparty
	packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := counterparty.QueryProof(packetKey)

	ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, source.GetSenderAccount().GetAddress().String())
	return coord.SendMsgs(source, counterparty, counterpartyClient, []sdk.Msg{ackMsg})
}

// CommitBlock commits a block on the provided indexes and then increments the global time.
//
// CONTRACT: the passed in list of indexes must not contain duplicates
func (coord *Coordinator) CommitBlock(chains ...TestChainI) {
	for _, chain := range chains {
		switch chain.Type() {
		case Tendermint:
			chain.GetApp().(*simapp.SimApp).Commit()
			chain.NextBlock()
		case SoloMachine:
		}
	}
	coord.IncrementTime()
}

// IncrementTime iterates through all the TestChain's and increments their current header time
// by 5 seconds.
//
// CONTRACT: this function must be called after every commit on any TestChain.
func (coord *Coordinator) IncrementTime() {
	for _, chain := range coord.Chains {
		switch chain := chain.(type) {
		case *TestChain: // Fabric
			chain.currentTime = chain.currentTime.Add(time.Second)
			chain.Stub.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: chain.currentTime.Unix()}, nil)
			// case *TestChain: // Tendermint
			// chain.CurrentHeader.Time = chain.CurrentHeader.Time.Add(timeIncrement)
			// chain.App.BeginBlock(abci.RequestBeginBlock{Header: chain.CurrentHeader})
		default:
			panic("not implemented error")
		}
	}
}

// GetChain returns the TestChain using the given chainID and returns an error if it does
// not exist.
func (coord *Coordinator) GetChain(chainID string) TestChainI {
	chain, found := coord.Chains[chainID]
	require.True(coord.t, found, fmt.Sprintf("%s chain does not exist", chainID))
	return chain
}

// SendMsg delivers a single provided message to the chain. The counterparty
// client is update with the new source consensus state.
func (coord *Coordinator) SendMsg(source, counterparty TestChainI, counterpartyClientID string, msg sdk.Msg) error {
	return coord.SendMsgs(source, counterparty, counterpartyClientID, []sdk.Msg{msg})
}

// SendMsgs delivers the provided messages to the chain. The counterparty
// client is updated with the new source consensus state.
func (coord *Coordinator) SendMsgs(source, counterparty TestChainI, counterpartyClientID string, msgs []sdk.Msg) error {
	if _, err := source.SendMsgs(msgs...); err != nil {
		return err
	}

	coord.IncrementTime()

	// update source client on counterparty connection
	return coord.UpdateClient(
		counterparty, source,
		counterpartyClientID, counterparty.Type(),
	)
}

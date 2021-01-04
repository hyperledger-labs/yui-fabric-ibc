package cross_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	ibctesting "github.com/cosmos/cosmos-sdk/x/ibc/testing"
	samplemodtypes "github.com/datachainlab/cross/simapp/samplemod/types"
	accounttypes "github.com/datachainlab/cross/x/core/account/types"
	authtypes "github.com/datachainlab/cross/x/core/auth/types"
	initiatortypes "github.com/datachainlab/cross/x/core/initiator/types"
	txtypes "github.com/datachainlab/cross/x/core/tx/types"
	crosstypes "github.com/datachainlab/cross/x/core/types"
	xcctypes "github.com/datachainlab/cross/x/core/xcc/types"
	crosstesting "github.com/datachainlab/cross/x/ibc/testing"
	"github.com/datachainlab/fabric-ibc/example"
	fabrictesting "github.com/datachainlab/fabric-ibc/x/ibc/testing"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"
)

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}

type AppTestSuite struct {
	suite.Suite

	coordinator *fabrictesting.Coordinator

	// testing chains used for convenience and readability
	chainA fabrictesting.TestChainI
	chainB fabrictesting.TestChainI
}

func (suite *AppTestSuite) SetupTest() {
	suite.coordinator = fabrictesting.NewCoordinator(suite.T(), 2, "SampleOrgMSP")
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
}

func (suite *AppTestSuite) TestInitiateTxSimple() {
	// setup

	clientA, clientB, connA, connB := suite.coordinator.SetupClientConnections(suite.chainA, suite.chainB, fabrictesting.Fabric)
	suite.chainB.CreatePortCapability(crosstypes.PortID)

	channelA, channelB := suite.coordinator.CreateChannel(suite.chainA, suite.chainB, connA, connB, crosstypes.PortID, crosstypes.PortID, channeltypes.UNORDERED)

	chAB := xcctypes.ChannelInfo{Port: channelA.PortID, Channel: channelA.ID}
	xccB, err := xcctypes.PackCrossChainChannel(&chAB)
	suite.Require().NoError(err)
	chBA := xcctypes.ChannelInfo{Port: channelB.PortID, Channel: channelB.ID}
	xccA, err := xcctypes.PackCrossChainChannel(&chBA)
	suite.Require().NoError(err)

	chainA := suite.chainA.(*fabrictesting.TestChain)
	chainB := suite.chainB.(*fabrictesting.TestChain)
	appA := suite.chainA.GetApp().(*example.IBCApp)

	xccSelf, err := xcctypes.PackCrossChainChannel(appA.XCCResolver.GetSelfCrossChainChannel(suite.chainA.GetContext()))
	suite.Require().NoError(err)

	var txID txtypes.TxID

	// Send a MsgInitiateTx to chainA
	{
		msg0 := initiatortypes.NewMsgInitiateTx(
			chainA.SenderAccount.GetAddress().Bytes(),
			chainA.ChainID,
			0,
			txtypes.COMMIT_PROTOCOL_SIMPLE,
			[]initiatortypes.ContractTransaction{
				{
					CrossChainChannel: xccSelf,
					Signers: []accounttypes.AccountID{
						accounttypes.AccountID(chainA.SenderAccount.GetAddress()),
					},
					CallInfo: samplemodtypes.NewContractCallRequest("counter").ContractCallInfo(chainA.App.AppCodec()),
				},
				{
					CrossChainChannel: xccB,
					Signers: []accounttypes.AccountID{
						accounttypes.AccountID(chainB.SenderAccount.GetAddress()),
					},
					CallInfo: samplemodtypes.NewContractCallRequest("counter").ContractCallInfo(chainB.App.AppCodec()),
				},
			},
			clienttypes.NewHeight(0, uint64(chainA.CurrentHeader.Height)+100),
			0,
		)
		res0, err := sendMsgs(suite.coordinator, chainA, chainB, clientB, msg0)
		suite.Require().NoError(err)
		suite.chainA.NextBlock()

		var txMsgData sdk.TxMsgData
		var initiateTxRes initiatortypes.MsgInitiateTxResponse
		suite.Require().NoError(proto.Unmarshal(res0.Data, &txMsgData))
		suite.Require().NoError(proto.Unmarshal(txMsgData.Data[0].Data, &initiateTxRes))
		suite.Require().Equal(initiatortypes.INITIATE_TX_STATUS_PENDING, initiateTxRes.Status)
		txID = initiateTxRes.TxID
	}

	// Send a MsgIBCSignTx to chainB & receive the MsgIBCSignTx to run the transaction on chainA
	var packetCall channeltypes.Packet
	{
		msg1 := authtypes.MsgIBCSignTx{
			CrossChainChannel: xccA,
			TxID:              txID,
			Signers:           []accounttypes.AccountID{chainB.SenderAccount.GetAddress().Bytes()},
			TimeoutHeight:     clienttypes.NewHeight(0, uint64(chainB.CurrentHeader.Height)+100),
			TimeoutTimestamp:  0,
		}
		res1, err := sendMsgs(suite.coordinator, suite.chainB, suite.chainA, clientA, &msg1)
		suite.Require().NoError(err)
		suite.chainB.NextBlock()

		ps, err := crosstesting.GetPacketsFromEvents(res1.GetEvents().ToABCIEvents())
		suite.Require().NoError(err)
		p := ps[0]

		res2, err := recvPacket(
			suite.coordinator, suite.chainB, suite.chainA, clientB, p,
		)
		suite.Require().NoError(err)
		suite.chainA.NextBlock()
		ps, err = crosstesting.GetPacketsFromEvents(res2.GetEvents().ToABCIEvents())
		suite.Require().NoError(err)
		packetCall = ps[0]

		ack, err := crosstesting.GetACKFromEvents(res2.GetEvents().ToABCIEvents())
		suite.Require().NoError(err)
		_, err = acknowledgePacket(
			suite.coordinator,
			suite.chainB,
			suite.chainA,
			clientA,
			p,
			ack,
		)
		suite.Require().NoError(err)
		suite.chainB.NextBlock()
	}

	// Send a PacketDataCall to chainB
	{
		suite.Require().NoError(
			suite.coordinator.UpdateClient(suite.chainB, suite.chainA, clientB, fabrictesting.Fabric),
		)
		res, err := recvPacket(
			suite.coordinator, suite.chainA, suite.chainB, clientA, packetCall,
		)
		suite.Require().NoError(err)
		suite.chainB.NextBlock()

		ack, err := crosstesting.GetACKFromEvents(res.GetEvents().ToABCIEvents())
		suite.Require().NoError(err)
		_, err = acknowledgePacket(
			suite.coordinator,
			suite.chainA,
			suite.chainB,
			clientB,
			packetCall,
			ack,
		)
		suite.Require().NoError(err)
		suite.chainB.NextBlock()
	}
}

func sendMsgs(coord *fabrictesting.Coordinator, source, counterparty fabrictesting.TestChainI, counterpartyClientID string, msgs ...sdk.Msg) (*sdk.Result, error) {
	res, err := source.SendMsgs(msgs...)
	if err != nil {
		return nil, err
	}

	coord.IncrementTime()
	err = coord.UpdateClient(
		counterparty, source,
		counterpartyClientID, counterparty.Type(),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func recvPacket(coord *fabrictesting.Coordinator, source, counterparty fabrictesting.TestChainI, sourceClient string, packet channeltypes.Packet) (*sdk.Result, error) {
	// get proof of packet commitment on source
	packetKey := host.KeyPacketCommitment(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := source.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, counterparty.(*fabrictesting.TestChain).SenderAccount.GetAddress())

	// receive on counterparty and update source client
	return sendMsgs(coord, counterparty, source, sourceClient, recvMsg)
}

func acknowledgePacket(coord *fabrictesting.Coordinator,
	source, counterparty fabrictesting.TestChainI,
	counterpartyClient string,
	packet channeltypes.Packet, ack []byte,
) (*sdk.Result, error) {
	packetKey := host.KeyPacketAcknowledgement(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := counterparty.QueryProof(packetKey)

	ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, source.(*fabrictesting.TestChain).SenderAccount.GetAddress())
	return sendMsgs(coord, source, counterparty, counterpartyClient, ackMsg)
}

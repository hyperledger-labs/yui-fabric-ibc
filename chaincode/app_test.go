package chaincode_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibctesting "github.com/cosmos/cosmos-sdk/x/ibc/testing"
	"github.com/datachainlab/fabric-ibc/app"
	fabrictesting "github.com/datachainlab/fabric-ibc/x/ibc/testing"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
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

func (suite *AppTestSuite) TestTransfer() {
	suite.coordinator = fabrictesting.NewCoordinator(suite.T(), 2, "SampleOrgMSP", fabrictesting.TxSignModeStdTx)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	clientA, clientB, connA, connB := suite.coordinator.SetupClientConnections(suite.chainA, suite.chainB, fabrictesting.Fabric)
	channelA, channelB := suite.coordinator.CreateTransferChannels(suite.chainA, suite.chainB, connA, connB, channeltypes.UNORDERED)

	chainA := suite.chainA.(*fabrictesting.TestChain)
	chainB := suite.chainB.(*fabrictesting.TestChain)

	originalBalance := chainA.App.BankKeeper.GetBalance(suite.chainA.GetContext(), chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	timeoutHeight := clienttypes.NewHeight(0, 110)

	coinToSendToB := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))

	// send from chainA to chainB
	msg := transfertypes.NewMsgTransfer(channelA.PortID, channelA.ID, coinToSendToB, chainA.SenderAccount.GetAddress(), chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0)
	err := suite.coordinator.SendMsg(suite.chainA, suite.chainB, clientB, msg)
	suite.Require().NoError(err) // message committed

	// relay send
	fungibleTokenPacket := transfertypes.NewFungibleTokenPacketData(coinToSendToB.Denom, coinToSendToB.Amount.Uint64(), chainA.SenderAccount.GetAddress().String(), chainB.SenderAccount.GetAddress().String())
	packet := channeltypes.NewPacket(fungibleTokenPacket.GetBytes(), 1, channelA.PortID, channelA.ID, channelB.PortID, channelB.ID, timeoutHeight, 0)
	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	err = suite.coordinator.RelayPacket(suite.chainA, suite.chainB, clientA, clientB, packet, ack.GetBytes())
	suite.Require().NoError(err) // relay committed

	// check that voucher exists on chain B
	voucherDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(packet.GetDestPort(), packet.GetDestChannel(), sdk.DefaultBondDenom))
	balance := chainB.App.BankKeeper.GetBalance(suite.chainB.GetContext(), chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())

	coinToSendBackToA := transfertypes.GetTransferCoin(channelB.PortID, channelB.ID, sdk.DefaultBondDenom, 100)
	suite.Require().Equal(coinToSendBackToA, balance)

	// send from chainB back to chainA
	msg = transfertypes.NewMsgTransfer(channelB.PortID, channelB.ID, coinToSendBackToA, chainB.SenderAccount.GetAddress(), chainA.SenderAccount.GetAddress().String(), timeoutHeight, 0)

	err = suite.coordinator.SendMsg(suite.chainB, suite.chainA, clientA, msg)
	suite.Require().NoError(err) // message committed

	// relay send
	// NOTE: fungible token is prefixed with the full trace in order to verify the packet commitment
	voucherDenom := voucherDenomTrace.GetPrefix() + voucherDenomTrace.BaseDenom
	fungibleTokenPacket = transfertypes.NewFungibleTokenPacketData(voucherDenom, coinToSendBackToA.Amount.Uint64(), chainB.SenderAccount.GetAddress().String(), chainA.SenderAccount.GetAddress().String())
	packet = channeltypes.NewPacket(fungibleTokenPacket.GetBytes(), 1, channelB.PortID, channelB.ID, channelA.PortID, channelA.ID, timeoutHeight, 0)
	err = suite.coordinator.RelayPacket(suite.chainB, suite.chainA, clientB, clientA, packet, ack.GetBytes())
	suite.Require().NoError(err) // relay committed

	balance = chainA.App.BankKeeper.GetBalance(suite.chainA.GetContext(), chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

	// check that the balance on chainA returned back to the original state
	suite.Require().Equal(originalBalance, balance)

	// check that module account escrow address is empty
	escrowAddress := transfertypes.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel())
	balance = chainA.App.BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom)
	suite.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()), balance)

	// check that balance on chain B is empty
	balance = chainB.App.BankKeeper.GetBalance(suite.chainB.GetContext(), chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	suite.Require().Zero(balance.Amount.Int64())
}

func (suite *AppTestSuite) TestQuery() {
	suite.coordinator = fabrictesting.NewCoordinator(suite.T(), 2, "SampleOrgMSP", fabrictesting.TxSignModeFabricTx)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	clientA, _, _, _ := suite.coordinator.SetupClientConnections(suite.chainA, suite.chainB, fabrictesting.Fabric)

	cc, err := contractapi.NewChaincode(suite.chainA.(*fabrictesting.TestChain).CC)
	suite.Require().NoError(err)
	req := &clienttypes.QueryClientStateRequest{
		ClientId: clientA,
	}
	data, err := proto.Marshal(req)
	jbz, err := json.Marshal(app.RequestQuery{
		Path: "/ibc.core.client.v1.Query/ClientState",
		Data: string(data),
	})
	stubA := suite.chainA.(*fabrictesting.TestChain).Stub
	stubA.GetFunctionAndParametersReturns("Query", []string{string(jbz)})
	res := cc.Invoke(stubA)
	var r app.ResponseQuery
	suite.Require().NoError(json.Unmarshal(res.Payload, &r))

	var cres clienttypes.QueryClientStateResponse
	bz, err := base64.StdEncoding.DecodeString(r.Value)
	suite.Require().NoError(err)
	suite.Require().NoError(cres.Unmarshal(bz))
}

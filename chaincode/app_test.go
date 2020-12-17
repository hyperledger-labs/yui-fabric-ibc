package chaincode_test

import (
	"testing"

	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibctesting "github.com/cosmos/cosmos-sdk/x/ibc/testing"
	fabrictesting "github.com/datachainlab/fabric-ibc/x/ibc/testing"
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
	suite.coordinator = fabrictesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
}

func (suite *AppTestSuite) TestTransfer() {
	clientA, clientB, connA, connB := suite.coordinator.SetupClientConnections(suite.chainA, suite.chainB, fabrictesting.Fabric)
	channelA, channelB := suite.coordinator.CreateTransferChannels(suite.chainA, suite.chainB, connA, connB, channeltypes.UNORDERED)

	_, _, _, _ = clientA, clientB, connA, connB
	_, _ = channelA, channelB
}

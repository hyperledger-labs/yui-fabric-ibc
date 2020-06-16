package chaincode

import (
	"os"

	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/datachainlab/fabric-ibc/x/ibc"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

type IBCChaincode struct {
	contractapi.Contract
	runner AppRunner
}

func (c *IBCChaincode) HandleIBCMsg(ctx contractapi.TransactionContextInterface, msgJSON string) error {
	return c.runner.RunMsg(ctx.GetStub(), msgJSON)
}

func (c *IBCChaincode) GenerateChaincodeHeader(ctx contractapi.ContractInterface) error {
	// TODO generate sequence and timestamp
	return nil
}

func (c *IBCChaincode) EndorseChaincodeHeader(ctx contractapi.ContractInterface) error {
	return nil
}

func (c *IBCChaincode) EndorsePacketCommitment(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) error {
	return c.runner.RunFunc(ctx.GetStub(), func(app *App) error {
		entry, err := commitment.MakePacketCommitmentEntry(
			app.NewContext(false, abci.Header{}),
			app.IBCKeeper.ChannelKeeper,
			commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)), // TODO use fabric prefix instead of this
			portID,
			channelID,
			sequence,
		)
		if err != nil {
			return err
		}
		// TODO also put timestamp and sequence entry?
		return ctx.GetStub().PutState(entry.Key, entry.Value)
	})
}

func NewIBCChaincode() *IBCChaincode {
	logger := log.NewTMLogger(os.Stdout)
	runner := NewAppRunner(logger, DefaultDBProvider)
	c := &IBCChaincode{
		runner: runner,
	}
	return c
}

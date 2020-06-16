package chaincode

import (
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/datachainlab/fabric-ibc/x/ibc"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

type IBCChaincode struct {
	contractapi.Contract
	sequenceMgr commitment.SequenceManager
	runner      AppRunner
}

func (c *IBCChaincode) HandleIBCMsg(ctx contractapi.TransactionContextInterface, msgJSON string) error {
	return c.runner.RunMsg(ctx.GetStub(), msgJSON)
}

func (c *IBCChaincode) UpdateSequence(ctx contractapi.TransactionContextInterface) error {
	_, err := c.sequenceMgr.UpdateSequence(ctx.GetStub())
	return err
}

func (c *IBCChaincode) MakeSequenceCommitment(ctx contractapi.TransactionContextInterface) error {
	var (
		seq *commitment.Sequence
		err error
	)

	args := ctx.GetStub().GetArgs()
	if len(args) > 0 {
		seqValue := sdk.BigEndianToUint64(args[0])
		seq, err = c.sequenceMgr.GetSequence(ctx.GetStub(), seqValue)
	} else {
		seq, err = c.sequenceMgr.GetCurrentSequence(ctx.GetStub())
	}
	if err != nil {
		return err
	}

	entry, err := commitment.MakeSequenceCommitmentEntry(seq)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(entry.Key, entry.Value)
}

func (c *IBCChaincode) MakePacketCommitment(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) error {
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

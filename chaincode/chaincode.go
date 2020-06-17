package chaincode

import (
	"errors"
	"fmt"
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

func (c *IBCChaincode) EndorseSequenceCommitment(ctx contractapi.TransactionContextInterface) error {
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

func (c *IBCChaincode) EndorsePacketCommitment(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) error {
	return c.runner.RunFunc(ctx.GetStub(), func(app *App) error {
		c := app.NewContext(false, abci.Header{})
		cmbz := app.IBCKeeper.ChannelKeeper.GetPacketCommitment(c, portID, channelID, sequence)
		if cmbz == nil {
			return errors.New("commitment not found")
		}

		entry, err := commitment.MakePacketCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)), // TODO use fabric prefix instead of this
			portID,
			channelID,
			sequence,
			cmbz,
		)
		if err != nil {
			return err
		}
		// TODO also put timestamp and sequence entry?
		return ctx.GetStub().PutState(entry.Key, entry.Value)
	})
}

func (c *IBCChaincode) EndorseConsensusStateCommitment(ctx contractapi.TransactionContextInterface, clientID string, height uint64) error {
	return c.runner.RunFunc(ctx.GetStub(), func(app *App) error {
		c := app.NewContext(false, abci.Header{})
		cs, ok := app.IBCKeeper.ClientKeeper.GetClientConsensusState(c, clientID, height)
		if !ok {
			return fmt.Errorf("consensusState not found: clientID=%v height=%v", clientID, height)
		}
		bz, err := app.cdc.Amino.MarshalBinaryBare(cs)
		if err != nil {
			return err
		}
		entry, err := commitment.MakeConsensusStateCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)), // TODO use fabric prefix instead of this
			clientID,
			height,
			bz,
		)
		if err != nil {
			return err
		}
		return ctx.GetStub().PutState(entry.Key, entry.Value)
	})
}

func NewIBCChaincode() *IBCChaincode {
	logger := log.NewTMLogger(os.Stdout)
	runner := NewAppRunner(logger, DefaultDBProvider)
	c := &IBCChaincode{
		runner:      runner,
		sequenceMgr: commitment.NewSequenceManager(commitment.DefaultConfig(), commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey))),
	}
	return c
}

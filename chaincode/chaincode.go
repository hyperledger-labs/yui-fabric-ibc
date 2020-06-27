package chaincode

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	"github.com/datachainlab/fabric-ibc/app"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/datachainlab/fabric-ibc/x/ibc"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	EventIBC = "ibc"
)

type IBCChaincode struct {
	logger log.Logger
	contractapi.Contract
	sequenceMgr commitment.SequenceManager
	runner      AppRunner
}

func (c *IBCChaincode) InitChaincode(ctx contractapi.TransactionContextInterface, appStateJSON string) error {
	if err := c.runner.Init(ctx.GetStub(), []byte(appStateJSON)); err != nil {
		return err
	}
	if _, err := c.sequenceMgr.InitSequence(ctx.GetStub()); err != nil {
		return err
	}
	return nil
}

func (c *IBCChaincode) HandleIBCMsg(ctx contractapi.TransactionContextInterface, txJSON string) error {
	events, err := c.runner.RunMsg(ctx.GetStub(), []byte(txJSON))
	if err != nil {
		return err
	}
	bz, err := json.Marshal(events)
	if err != nil {
		return err
	}
	return ctx.GetStub().SetEvent(EventIBC, bz)
}

func (c *IBCChaincode) UpdateSequence(ctx contractapi.TransactionContextInterface) (*commitment.Sequence, error) {
	return c.sequenceMgr.UpdateSequence(ctx.GetStub())
}

func (c *IBCChaincode) EndorseSequenceCommitment(ctx contractapi.TransactionContextInterface) (*commitment.Entry, error) {
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
		return nil, err
	}

	entry, err := commitment.MakeSequenceCommitmentEntry(seq)
	if err != nil {
		return nil, err
	}
	if err := ctx.GetStub().PutState(entry.Key, entry.Value); err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *IBCChaincode) EndorseConnectionState(ctx contractapi.TransactionContextInterface, connectionID string) (*commitment.Entry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app *app.IBCApp) error {
		c, writer := app.MakeContext(abci.Header{})

		connection, found := app.IBCKeeper.ConnectionKeeper.GetConnection(c, connectionID)
		if !found {
			return sdkerrors.Wrap(types.ErrConnectionNotFound, "cannot relay ACK of open attempt")
		}
		bz, err := proto.Marshal(&connection)
		if err != nil {
			return err
		}
		e, err := commitment.MakeConnectionStateCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)),
			connectionID,
			bz,
		)
		if err != nil {
			return err
		}
		entry = e
		writer()
		return ctx.GetStub().PutState(e.Key, e.Value)
	}); err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *IBCChaincode) EndorseChannelState(ctx contractapi.TransactionContextInterface, portID, channelID string) (*commitment.Entry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app *app.IBCApp) error {
		c, writer := app.MakeContext(abci.Header{})

		channel, found := app.IBCKeeper.ChannelKeeper.GetChannel(c, portID, channelID)
		if !found {
			return sdkerrors.Wrap(channeltypes.ErrChannelNotFound, channelID)
		}
		bz, err := proto.Marshal(&channel)
		if err != nil {
			return err
		}
		e, err := commitment.MakeChannelStateCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)),
			portID,
			channelID,
			bz,
		)
		if err != nil {
			return err
		}
		entry = e
		writer()
		return ctx.GetStub().PutState(e.Key, e.Value)
	}); err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *IBCChaincode) EndorsePacketCommitment(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) (*commitment.Entry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app *app.IBCApp) error {
		c, writer := app.MakeContext(abci.Header{})
		cmbz := app.IBCKeeper.ChannelKeeper.GetPacketCommitment(c, portID, channelID, sequence)
		if cmbz == nil {
			return errors.New("commitment not found")
		}

		e, err := commitment.MakePacketCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)), // TODO use fabric prefix instead of this
			portID,
			channelID,
			sequence,
			cmbz,
		)
		if err != nil {
			return err
		}
		entry = e
		writer()
		return ctx.GetStub().PutState(e.Key, e.Value)
	}); err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *IBCChaincode) EndorsePacketAcknowledgement(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) (*commitment.Entry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app *app.IBCApp) error {
		c, writer := app.MakeContext(abci.Header{})
		ackBytes, ok := app.IBCKeeper.ChannelKeeper.GetPacketAcknowledgement(c, portID, channelID, sequence)
		if !ok {
			return errors.New("acknowledgement packet not found")
		}
		e, err := commitment.MakePacketAcknowledgementEntry(
			commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)),
			portID,
			channelID,
			sequence,
			ackBytes,
		)
		if err != nil {
			return err
		}
		entry = e
		writer()
		return ctx.GetStub().PutState(e.Key, e.Value)
	}); err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *IBCChaincode) EndorsePacketAcknowledgementAbsence(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) (*commitment.Entry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app *app.IBCApp) error {
		c, writer := app.MakeContext(abci.Header{})
		_, ok := app.IBCKeeper.ChannelKeeper.GetPacketAcknowledgement(c, portID, channelID, sequence)
		if ok {
			return errors.New("acknowledgement packet found")
		}
		e, err := commitment.MakePacketAcknowledgementAbsenceEntry(
			commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)),
			portID,
			channelID,
			sequence,
		)
		if err != nil {
			return err
		}
		entry = e
		writer()
		return ctx.GetStub().PutState(e.Key, e.Value)
	}); err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *IBCChaincode) EndorseConsensusStateCommitment(ctx contractapi.TransactionContextInterface, clientID string, height uint64) (*commitment.Entry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app *app.IBCApp) error {
		c, writer := app.MakeContext(abci.Header{})
		cs, ok := app.IBCKeeper.ClientKeeper.GetClientConsensusState(c, clientID, height)
		if !ok {
			return fmt.Errorf("consensusState not found: clientID=%v height=%v", clientID, height)
		}
		bz, err := app.Codec().Amino.MarshalBinaryBare(cs)
		if err != nil {
			return err
		}
		e, err := commitment.MakeConsensusStateCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)), // TODO use fabric prefix instead of this
			clientID,
			height,
			bz,
		)
		if err != nil {
			return err
		}
		entry = e
		writer()
		return ctx.GetStub().PutState(e.Key, e.Value)
	}); err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *IBCChaincode) EndorseNextSequenceRecv(ctx contractapi.TransactionContextInterface, portID, channelID string) (*commitment.Entry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app *app.IBCApp) error {
		c, writer := app.MakeContext(abci.Header{})
		seq, found := app.IBCKeeper.ChannelKeeper.GetNextSequenceRecv(c, portID, channelID)
		if !found {
			return sdkerrors.Wrapf(
				channel.ErrSequenceReceiveNotFound,
				"port: %s, channel: %s", portID, channelID,
			)
		}
		e, err := commitment.MakeNextSequenceRecvEntry(
			commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)), // TODO use fabric prefix instead of this
			portID,
			channelID,
			seq,
		)
		if err != nil {
			return err
		}
		entry = e
		writer()
		return ctx.GetStub().PutState(e.Key, e.Value)
	}); err != nil {
		return nil, err
	}
	return entry, nil
}

func NewIBCChaincode() *IBCChaincode {
	logger := log.NewTMLogger(os.Stdout)
	sequenceMgr := commitment.NewSequenceManager(commitment.DefaultConfig(), commitmenttypes.NewMerklePrefix([]byte(ibc.StoreKey)))
	runner := NewAppRunner(logger, DefaultDBProvider, &sequenceMgr)
	c := &IBCChaincode{
		sequenceMgr: sequenceMgr,
		runner:      runner,
	}
	return c
}

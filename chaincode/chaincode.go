package chaincode

import (
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"

	"github.com/hyperledger-labs/yui-fabric-ibc/app"
	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

const (
	EventIBC = "ibc"
)

// IBCChaincode provides Cosmos-SDK based Applicatoin which implements IBC
type IBCChaincode struct {
	contractapi.Contract
	logger       log.Logger
	sequenceMgr  commitment.SequenceManager
	runner       AppRunner
	eventHandler MultiEventHandler
}

// NewIBCChaincode returns a new instance of IBCChaincode
func NewIBCChaincode(appName string, logger log.Logger, seqMgr commitment.SequenceManager, appProvider AppProvider, anteHandlerProvider app.AnteHandlerProvider, dbProvider DBProvider, eventHandler MultiEventHandler) *IBCChaincode {
	runner := NewAppRunner(appName, logger, appProvider, anteHandlerProvider, dbProvider, seqMgr)
	c := &IBCChaincode{
		logger:       logger,
		sequenceMgr:  seqMgr,
		runner:       runner,
		eventHandler: eventHandler,
	}
	return c
}

// SetEventHandler sets IBCChaincode to a given handler
func (c *IBCChaincode) SetEventHandler(handler MultiEventHandler) {
	c.eventHandler = handler
}

// InitChaincode initialize the state of the chaincode
// This must be called when the chaincode is initialized
func (c *IBCChaincode) InitChaincode(ctx contractapi.TransactionContextInterface, appStateJSON string) error {
	if err := c.runner.Init(ctx.GetStub(), []byte(appStateJSON)); err != nil {
		return err
	}
	if _, err := c.sequenceMgr.InitSequence(ctx.GetStub()); err != nil {
		return err
	}
	return nil
}

// HandleTx handles a transaction
func (c *IBCChaincode) HandleTx(ctx contractapi.TransactionContextInterface, txJSON string) (*app.ResponseTx, error) {
	res, events, err := c.runner.RunTx(ctx.GetStub(), []byte(txJSON))
	if err != nil {
		return nil, err
	}
	if err := c.eventHandler.Handle(ctx, events); err != nil {
		return nil, err
	}
	return res, ctx.GetStub().SetEvent(EventIBC, []byte(res.Events))
}

func (c *IBCChaincode) Query(ctx contractapi.TransactionContextInterface, reqJSON string) (*app.ResponseQuery, error) {
	var req app.RequestQuery
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		return nil, err
	}
	return c.runner.Query(ctx.GetStub(), req)
}

// GetSequence returns current Sequence
func (c *IBCChaincode) GetSequence(ctx contractapi.TransactionContextInterface) (*commitment.Sequence, error) {
	return c.sequenceMgr.GetCurrentSequence(ctx.GetStub())
}

// UpdateSequence updates Sequence
func (c *IBCChaincode) UpdateSequence(ctx contractapi.TransactionContextInterface) (*commitment.Sequence, error) {
	return c.sequenceMgr.UpdateSequence(ctx.GetStub())
}

func (c *IBCChaincode) EndorseSequenceCommitment(ctx contractapi.TransactionContextInterface) (*commitment.CommitmentEntry, error) {
	var (
		seq *commitment.Sequence
		err error
	)

	args := ctx.GetStub().GetArgs()
	if len(args) > 1 {
		seqValue := sdk.BigEndianToUint64(args[1])
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
	return entry.ToCommitment(), nil
}

func (c *IBCChaincode) EndorseClientState(ctx contractapi.TransactionContextInterface, clientID string) (*commitment.CommitmentEntry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app app.Application) error {
		cctx, writer := app.MakeCacheContext(tmproto.Header{})

		cs, found := app.GetIBCKeeper().ClientKeeper.GetClientState(cctx, clientID)
		if !found {
			return sdkerrors.Wrapf(clienttypes.ErrClientNotFound, "clientID '%v' not found", clientID)
		}
		bz, err := clienttypes.MarshalClientState(app.AppCodec(), cs)
		if err != nil {
			return err
		}
		e, err := commitment.MakeClientStateCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(host.StoreKey)),
			clientID,
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
	return entry.ToCommitment(), nil
}

func (c *IBCChaincode) EndorseConsensusStateCommitment(ctx contractapi.TransactionContextInterface, clientID string, height uint64) (*commitment.CommitmentEntry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app app.Application) error {
		cctx, writer := app.MakeCacheContext(tmproto.Header{})
		h := c.makeHeight(height)
		css, ok := app.GetIBCKeeper().ClientKeeper.GetClientConsensusState(cctx, clientID, h)
		if !ok {
			return fmt.Errorf("consensusState not found: clientID=%v height=%v", clientID, height)
		}
		bz, err := clienttypes.MarshalConsensusState(app.AppCodec(), css)
		if err != nil {
			return err
		}
		e, err := commitment.MakeConsensusStateCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(host.StoreKey)), // TODO use fabric prefix instead of this
			clientID,
			h,
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
	return entry.ToCommitment(), nil
}

func (c *IBCChaincode) EndorseConnectionState(ctx contractapi.TransactionContextInterface, connectionID string) (*commitment.CommitmentEntry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app app.Application) error {
		cctx, writer := app.MakeCacheContext(tmproto.Header{})

		connection, found := app.GetIBCKeeper().ConnectionKeeper.GetConnection(cctx, connectionID)
		if !found {
			return sdkerrors.Wrapf(connectiontypes.ErrConnectionNotFound, "cannot relay ACK of open attempt: %v", connectionID)
		}
		bz, err := app.AppCodec().Marshal(&connection)
		if err != nil {
			return err
		}
		e, err := commitment.MakeConnectionStateCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(host.StoreKey)),
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
	return entry.ToCommitment(), nil
}

func (c *IBCChaincode) EndorseChannelState(ctx contractapi.TransactionContextInterface, portID, channelID string) (*commitment.CommitmentEntry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app app.Application) error {
		c, writer := app.MakeCacheContext(tmproto.Header{})

		channel, found := app.GetIBCKeeper().ChannelKeeper.GetChannel(c, portID, channelID)
		if !found {
			return sdkerrors.Wrap(channeltypes.ErrChannelNotFound, channelID)
		}
		bz, err := app.AppCodec().Marshal(&channel)
		if err != nil {
			return err
		}
		e, err := commitment.MakeChannelStateCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(host.StoreKey)),
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
	return entry.ToCommitment(), nil
}

func (c *IBCChaincode) EndorsePacketCommitment(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) (*commitment.CommitmentEntry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app app.Application) error {
		cctx, writer := app.MakeCacheContext(tmproto.Header{})
		cmbz := app.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(cctx, portID, channelID, sequence)
		if cmbz == nil {
			return errors.New("commitment not found")
		}

		e, err := commitment.MakePacketCommitmentEntry(
			commitmenttypes.NewMerklePrefix([]byte(host.StoreKey)), // TODO use fabric prefix instead of this
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
	return entry.ToCommitment(), nil
}

func (c *IBCChaincode) EndorsePacketAcknowledgement(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) (*commitment.CommitmentEntry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app app.Application) error {
		cctx, writer := app.MakeCacheContext(tmproto.Header{})
		ackBytes, ok := app.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(cctx, portID, channelID, sequence)
		if !ok {
			return errors.New("acknowledgement packet not found")
		}
		e, err := commitment.MakePacketAcknowledgementEntry(
			commitmenttypes.NewMerklePrefix([]byte(host.StoreKey)),
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
	return entry.ToCommitment(), nil
}

func (c *IBCChaincode) EndorsePacketReceiptAbsence(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) (*commitment.CommitmentEntry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app app.Application) error {
		cctx, writer := app.MakeCacheContext(tmproto.Header{})
		_, ok := app.GetIBCKeeper().ChannelKeeper.GetPacketReceipt(cctx, portID, channelID, sequence)
		if ok {
			return errors.New("the packet receipt found")
		}
		e, err := commitment.MakePacketReceiptAbsenceEntry(
			commitmenttypes.NewMerklePrefix([]byte(host.StoreKey)),
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
	return entry.ToCommitment(), nil
}

func (c *IBCChaincode) EndorseNextSequenceRecv(ctx contractapi.TransactionContextInterface, portID, channelID string) (*commitment.CommitmentEntry, error) {
	var entry *commitment.Entry
	if err := c.runner.RunFunc(ctx.GetStub(), func(app app.Application) error {
		cctx, writer := app.MakeCacheContext(tmproto.Header{})
		seq, found := app.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(cctx, portID, channelID)
		if !found {
			return sdkerrors.Wrapf(
				channeltypes.ErrSequenceReceiveNotFound,
				"port: %s, channel: %s", portID, channelID,
			)
		}
		e, err := commitment.MakeNextSequenceRecvEntry(
			commitmenttypes.NewMerklePrefix([]byte(host.StoreKey)), // TODO use fabric prefix instead of this
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
	return entry.ToCommitment(), nil
}

func (c IBCChaincode) makeHeight(height uint64) clienttypes.Height {
	return clienttypes.NewHeight(0, height)
}

func (c IBCChaincode) GetAppRunner() AppRunner {
	return c.runner
}

func (c *IBCChaincode) GetIgnoredFunctions() []string {
	return []string{"SetEventHandler", "GetAppRunner"}
}

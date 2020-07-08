package chaincode

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc-transfer/types"
	connection "github.com/cosmos/cosmos-sdk/x/ibc/03-connection"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	"github.com/datachainlab/fabric-ibc/app"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/datachainlab/fabric-ibc/tests"
	fabric "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric/msp"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
)

type TestChaincodeApp struct {
	cc *IBCChaincode

	signer sdk.AccAddress
	prvKey crypto.PrivKey
	cdc    codec.Marshaler

	// Fabric
	fabChannelID   string
	fabChaincodeID fabrictypes.ChaincodeID
	endorser       msp.SigningIdentity

	// Sequence
	seq *commitment.Sequence

	// IBC
	clientID     string
	connectionID string
	portID       string
	channelID    string
	channelOrder channel.Order
}

func MakeTestChaincodeApp(
	prvKey crypto.PrivKey,
	fabChannelID string,
	fabChaincodeID fabrictypes.ChaincodeID,
	endorser msp.SigningIdentity,
	clientID string,
	connectionID string,
	portID string,
	channelID string,
	channelOrder channel.Order,
) TestChaincodeApp {
	cdc, _ := app.MakeCodecs()
	cc := NewIBCChaincode()
	return TestChaincodeApp{
		cc: cc,

		signer: sdk.AccAddress(prvKey.PubKey().Address()),
		prvKey: prvKey,
		cdc:    cdc,

		fabChannelID:   fabChannelID,
		fabChaincodeID: fabChaincodeID,
		endorser:       endorser,

		clientID:     clientID,
		connectionID: connectionID,
		portID:       portID,
		channelID:    channelID,
		channelOrder: channelOrder,
	}
}

func (ca *TestChaincodeApp) init(ctx contractapi.TransactionContextInterface) error {
	err := ca.cc.InitChaincode(ctx, "{}")
	if err != nil {
		return err
	}
	seq, err := ca.getEndorsedCurrentSequence(ctx)
	if err != nil {
		return err
	}
	ca.seq = seq
	return nil
}

func (ca TestChaincodeApp) runMsg(stub shim.ChaincodeStubInterface, msgs ...sdk.Msg) error {
	events, err := ca.cc.runner.RunMsg(stub, makeStdTxBytes(ca.cdc, ca.prvKey, msgs...))
	if err != nil {
		return err
	}
	if testing.Verbose() {
		for _, event := range events {
			log.Println(event.String())
		}
	}
	return nil
}

func (ca *TestChaincodeApp) updateSequence(ctx contractapi.TransactionContextInterface) (*commitment.Sequence, error) {
	seqJson, err := ca.cc.UpdateSequence(ctx)
	if err != nil {
		return nil, err
	}
	seq := new(commitment.Sequence)
	if err = json.Unmarshal([]byte(seqJson), seq); err != nil {
		return nil, err
	}
	ca.seq = seq
	return seq, nil
}

func (ca TestChaincodeApp) createMsgCreateClient(t *testing.T, ctx contractapi.TransactionContextInterface) *fabric.MsgCreateClient {
	var pcBytes []byte = makePolicy([]string{"SampleOrgMSP"})
	ci := fabric.NewChaincodeInfo(ca.fabChannelID, ca.fabChaincodeID, pcBytes, nil)
	ch := fabric.NewChaincodeHeader(ca.seq.Value, ca.seq.Timestamp, fabric.Proof{})
	h := fabric.NewHeader(ch, ci)
	msg := fabric.NewMsgCreateClient(ca.clientID, h, ca.signer)
	require.NoError(t, msg.ValidateBasic())
	return &msg
}

func (ca TestChaincodeApp) createMsgUpdateClient(t *testing.T) *fabric.MsgUpdateClient {
	var sigs [][]byte
	var pcBytes []byte = makePolicy([]string{"SampleOrgMSP"})
	ci := fabric.NewChaincodeInfo(ca.fabChannelID, ca.fabChaincodeID, pcBytes, sigs)
	proof, err := tests.MakeProof(ca.endorser, commitment.MakeSequenceCommitmentEntryKey(ca.seq.Value), ca.seq.Bytes())
	require.NoError(t, err)
	ch := fabric.NewChaincodeHeader(ca.seq.Value, ca.seq.Timestamp, *proof)
	h := fabric.NewHeader(ch, ci)
	msg := fabric.NewMsgUpdateClient(ca.clientID, h, ca.signer)
	require.NoError(t, msg.ValidateBasic())
	return &msg
}

func (ca TestChaincodeApp) createMsgConnectionOpenInit(
	t *testing.T,
	counterParty TestChaincodeApp,
) *connection.MsgConnectionOpenInit {
	msg := connection.NewMsgConnectionOpenInit(
		ca.connectionID,
		ca.clientID,
		counterParty.connectionID,
		counterParty.clientID,
		commitmenttypes.NewMerklePrefix([]byte("ibc")),
		ca.signer,
	)
	require.NoError(t, msg.ValidateBasic())
	return msg
}

func (ca TestChaincodeApp) createMsgConnectionOpenTry(
	t *testing.T,
	counterPartyCtx contractapi.TransactionContextInterface,
	counterParty TestChaincodeApp,
) *connection.MsgConnectionOpenTry {
	proofHeight, proofInit, err := counterParty.makeProofConnectionState(counterPartyCtx, counterParty.connectionID)
	require.NoError(t, err)
	consensusHeight, proofConsensus, err := counterParty.makeProofConsensus(counterPartyCtx, counterParty.clientID, 1) // TODO the height should be given from argument
	require.NoError(t, err)

	msg := connection.NewMsgConnectionOpenTry(
		ca.connectionID,
		ca.clientID,
		counterParty.connectionID,
		counterParty.clientID,
		commitmenttypes.NewMerklePrefix([]byte("ibc")),
		[]string{"1.0.0"},
		proofInit,
		proofConsensus,
		proofHeight,
		consensusHeight,
		ca.signer,
	)
	require.NoError(t, msg.ValidateBasic())
	return msg
}

func (ca TestChaincodeApp) createMsgConnectionOpenAck(
	t *testing.T,
	counterPartyCtx contractapi.TransactionContextInterface,
	counterParty TestChaincodeApp,
) *connection.MsgConnectionOpenAck {
	proofHeight, proofTry, err := counterParty.makeProofConnectionState(counterPartyCtx, counterParty.connectionID)
	require.NoError(t, err)
	consensusHeight, proofConsensus, err := counterParty.makeProofConsensus(counterPartyCtx, counterParty.clientID, 1) // TODO the height should be given from argument
	require.NoError(t, err)

	msg := connection.NewMsgConnectionOpenAck(
		ca.connectionID,
		proofTry,
		proofConsensus,
		proofHeight,
		consensusHeight,
		"1.0.0",
		ca.signer,
	)
	require.NoError(t, msg.ValidateBasic())
	return msg
}

func (ca TestChaincodeApp) createMsgConnectionOpenConfirm(
	t *testing.T,
	counterPartyCtx contractapi.TransactionContextInterface,
	counterParty TestChaincodeApp,
) *connection.MsgConnectionOpenConfirm {
	proofHeight, proofConfirm, err := counterParty.makeProofConnectionState(counterPartyCtx, counterParty.connectionID)
	require.NoError(t, err)

	msg := connection.NewMsgConnectionOpenConfirm(
		ca.connectionID,
		proofConfirm,
		proofHeight,
		ca.signer,
	)
	require.NoError(t, msg.ValidateBasic())
	return msg
}

func (ca TestChaincodeApp) createMsgChannelOpenInit(
	t *testing.T,
	counterParty TestChaincodeApp,
) *channel.MsgChannelOpenInit {
	msg := channel.NewMsgChannelOpenInit(
		ca.portID,
		ca.channelID,
		ibctransfertypes.Version,
		ca.channelOrder,
		[]string{ca.connectionID},
		counterParty.portID,
		counterParty.channelID,
		ca.signer,
	)
	require.NoError(t, msg.ValidateBasic())
	return msg
}

func (ca TestChaincodeApp) createMsgChannelOpenTry(
	t *testing.T,
	counterPartyCtx contractapi.TransactionContextInterface,
	counterParty TestChaincodeApp,
) *channel.MsgChannelOpenTry {
	proofHeight, proofInit, err := ca.makeProofChannelState(counterPartyCtx, counterParty.portID, counterParty.channelID)
	require.NoError(t, err)
	msg := channel.NewMsgChannelOpenTry(
		ca.portID,
		ca.channelID,
		ibctransfertypes.Version,
		ca.channelOrder,
		[]string{ca.connectionID},
		counterParty.portID,
		counterParty.channelID,
		ibctransfertypes.Version,
		proofInit,
		proofHeight,
		ca.signer,
	)
	require.NoError(t, msg.ValidateBasic())
	return msg
}

func (ca TestChaincodeApp) createMsgChannelOpenAck(
	t *testing.T,
	counterPartyCtx contractapi.TransactionContextInterface,
	counterParty TestChaincodeApp,
) *channel.MsgChannelOpenAck {
	proofHeight, proofTry, err := ca.makeProofChannelState(counterPartyCtx, counterParty.portID, counterParty.channelID)
	require.NoError(t, err)
	msg := channel.NewMsgChannelOpenAck(
		ca.portID,
		ca.channelID,
		ibctransfertypes.Version,
		proofTry,
		proofHeight,
		ca.signer,
	)
	require.NoError(t, msg.ValidateBasic())
	return msg
}

func (ca TestChaincodeApp) createMsgChannelOpenConfirm(
	t *testing.T,
	counterPartyCtx contractapi.TransactionContextInterface,
	counterParty TestChaincodeApp,
) *channel.MsgChannelOpenConfirm {
	proofHeight, proofAck, err := ca.makeProofChannelState(counterPartyCtx, counterParty.portID, counterParty.channelID)
	require.NoError(t, err)
	msg := channel.NewMsgChannelOpenConfirm(
		ca.portID,
		ca.channelID,
		proofAck,
		proofHeight,
		ca.signer,
	)
	require.NoError(t, msg.ValidateBasic())
	return msg
}

func (ca TestChaincodeApp) createMsgTransfer(
	t *testing.T,
	counterParty TestChaincodeApp,
	amount sdk.Coins,
	receiver sdk.AccAddress,
	timeoutHeight uint64,
	timeoutTimestamp uint64,
) *ibctransfertypes.MsgTransfer {
	return ibctransfertypes.NewMsgTransfer(
		ca.portID,
		ca.channelID,
		amount,
		ca.signer,
		receiver.String(),
		timeoutHeight,
		timeoutTimestamp,
	)
}

func (ca TestChaincodeApp) createMsgPacketForTransfer(
	t *testing.T,
	counterPartyCtx contractapi.TransactionContextInterface,
	counterParty TestChaincodeApp,
	packet channel.Packet,
) *channel.MsgPacket {
	proofHeight, packetProof, err := counterParty.makeProofPacketCommitment(counterPartyCtx, counterParty.portID, counterParty.channelID, 1)
	require.NoError(t, err)
	return channel.NewMsgPacket(
		packet,
		packetProof,
		proofHeight,
		ca.signer,
	)
}

func (ca TestChaincodeApp) createMsgAcknowledgement(
	t *testing.T,
	counterPartyCtx contractapi.TransactionContextInterface,
	counterParty TestChaincodeApp,
	packet channel.Packet,
) *channel.MsgAcknowledgement {
	proofHeight, proof, err := counterParty.makeProofPacketAcknowledgement(counterPartyCtx, counterParty.portID, counterParty.channelID, 1)
	require.NoError(t, err)
	ack := ibctransfertypes.FungibleTokenPacketAcknowledgement{Success: true}
	return channel.NewMsgAcknowledgement(packet, ack.GetBytes(), proof, proofHeight, ca.signer)
}

func (ca TestChaincodeApp) createMsgTimeoutPacket(
	t *testing.T,
	counterPartyCtx contractapi.TransactionContextInterface,
	counterParty TestChaincodeApp,
	nextSequenceRecv uint64,
	order channel.Order,
	packet channel.Packet,
) *channel.MsgTimeout {
	var proofHeight uint64
	var proof []byte
	var err error
	if order == channel.UNORDERED {
		panic("not implemented error")
	} else if order == channel.ORDERED {
		proofHeight, proof, err = counterParty.makeProofNextSequenceRecv(counterPartyCtx, counterParty.portID, counterParty.channelID, nextSequenceRecv)
		require.NoError(t, err)
	} else {
		panic(fmt.Sprintf("unknown channel order type: %v", order.String()))
	}

	return channel.NewMsgTimeout(packet, nextSequenceRecv, proof, proofHeight, ca.signer)
}

func (ca TestChaincodeApp) getEndorsedCurrentSequence(ctx contractapi.TransactionContextInterface) (*commitment.Sequence, error) {
	entryJson, err := ca.cc.EndorseSequenceCommitment(ctx)
	if err != nil {
		return nil, err
	}
	entry := new(commitment.Entry)
	if err = json.Unmarshal([]byte(entryJson), entry); err != nil {
		return nil, err
	}

	var seq commitment.Sequence
	if err := proto.Unmarshal(entry.Value, &seq); err != nil {
		return nil, err
	}
	return &seq, nil
}

func (ca TestChaincodeApp) makeMockEndorsedCommitmentProof(entry *commitment.Entry) (*fabric.Proof, error) {
	return tests.MakeProof(ca.endorser, entry.Key, entry.Value)
}

func (ca TestChaincodeApp) makeProofConsensus(ctx contractapi.TransactionContextInterface, clientID string, height uint64) (uint64, []byte, error) {
	entry, err := ca.cc.EndorseConsensusStateCommitment(ctx, clientID, height)
	proof, err := ca.makeMockEndorsedCommitmentProof(entry)
	if err != nil {
		return 0, nil, err
	}
	bz, err := proto.Marshal(proof)
	if err != nil {
		return 0, nil, err
	}
	return 1, bz, nil // FIXME returns current height
}

func (ca TestChaincodeApp) makeProofConnectionState(ctx contractapi.TransactionContextInterface, connectionID string) (uint64, []byte, error) {
	entry, err := ca.cc.EndorseConnectionState(ctx, connectionID)
	if err != nil {
		return 0, nil, err
	}
	proof, err := ca.makeMockEndorsedCommitmentProof(entry)
	if err != nil {
		return 0, nil, err
	}
	bz, err := proto.Marshal(proof)
	if err != nil {
		return 0, nil, err
	}
	return 1, bz, nil // FIXME returns current height
}

func (ca TestChaincodeApp) makeProofChannelState(ctx contractapi.TransactionContextInterface, portID, channelID string) (uint64, []byte, error) {
	entry, err := ca.cc.EndorseChannelState(ctx, portID, channelID)
	if err != nil {
		return 0, nil, err
	}
	proof, err := ca.makeMockEndorsedCommitmentProof(entry)
	if err != nil {
		return 0, nil, err
	}
	bz, err := proto.Marshal(proof)
	if err != nil {
		return 0, nil, err
	}
	return 1, bz, nil // FIXME returns current height
}

func (ca TestChaincodeApp) makeProofPacketCommitment(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) (uint64, []byte, error) {
	entry, err := ca.cc.EndorsePacketCommitment(ctx, portID, channelID, sequence)
	if err != nil {
		return 0, nil, err
	}
	proof, err := ca.makeMockEndorsedCommitmentProof(entry)
	if err != nil {
		return 0, nil, err
	}
	bz, err := proto.Marshal(proof)
	if err != nil {
		return 0, nil, err
	}
	return 1, bz, nil // FIXME returns current height
}

func (ca TestChaincodeApp) makeProofPacketAcknowledgement(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) (uint64, []byte, error) {
	entry, err := ca.cc.EndorsePacketAcknowledgement(ctx, portID, channelID, sequence)
	if err != nil {
		return 0, nil, err
	}
	proof, err := ca.makeMockEndorsedCommitmentProof(entry)
	if err != nil {
		return 0, nil, err
	}
	bz, err := proto.Marshal(proof)
	if err != nil {
		return 0, nil, err
	}
	return 1, bz, nil // FIXME returns current height
}

func (ca TestChaincodeApp) makeProofNextSequenceRecv(ctx contractapi.TransactionContextInterface, portID, channelID string, nextSequenceRecv uint64) (uint64, []byte, error) {
	entry, err := ca.cc.EndorseNextSequenceRecv(ctx, portID, channelID)
	if err != nil {
		return 0, nil, err
	}
	proof, err := ca.makeMockEndorsedCommitmentProof(entry)
	if err != nil {
		return 0, nil, err
	}
	bz, err := proto.Marshal(proof)
	if err != nil {
		return 0, nil, err
	}
	return ca.seq.Value, bz, nil
}

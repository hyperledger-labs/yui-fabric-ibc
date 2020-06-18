package chaincode

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc-transfer/types"
	connection "github.com/cosmos/cosmos-sdk/x/ibc/03-connection"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
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
	tmtime "github.com/tendermint/tendermint/types/time"
)

type TestChaincodeApp struct {
	cc *IBCChaincode

	signer sdk.AccAddress
	prvKey crypto.PrivKey
	cdc    *std.Codec

	// Fabric
	fabChannelID   string
	fabChaincodeID fabrictypes.ChaincodeID
	endorser       msp.SigningIdentity

	// Sequence
	seq uint64

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
	cdc, _ := MakeCodecs()
	cc := NewIBCChaincode()
	return TestChaincodeApp{
		cc: cc,

		signer: sdk.AccAddress(prvKey.PubKey().Address()),
		prvKey: prvKey,
		cdc:    cdc,

		fabChannelID:   fabChannelID,
		fabChaincodeID: fabChaincodeID,
		endorser:       endorser,

		seq: 1,

		clientID:     clientID,
		connectionID: connectionID,
		portID:       portID,
		channelID:    channelID,
		channelOrder: channelOrder,
	}
}

func (ca TestChaincodeApp) init(ctx contractapi.TransactionContextInterface) error {
	err := ca.cc.InitChaincode(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (ca TestChaincodeApp) runMsg(stub shim.ChaincodeStubInterface, msgs ...sdk.Msg) error {
	return ca.cc.runner.RunMsg(stub, string(makeStdTxBytes(ca.cdc, ca.prvKey, msgs...)))
}

func (ca TestChaincodeApp) createMsgCreateClient(t *testing.T, ctx contractapi.TransactionContextInterface) *fabric.MsgCreateClient {
	var pcBytes []byte = makePolicy([]string{"SampleOrg"})
	ci := fabric.NewChaincodeInfo(ca.fabChannelID, ca.fabChaincodeID, pcBytes, nil)
	seq, err := ca.getEndorsedCurrentSequence(ctx)
	require.NoError(t, err)
	require.Equal(t, ca.seq, seq.GetValue())
	ch := fabric.NewChaincodeHeader(seq.Value, seq.Timestamp, fabric.Proof{})
	h := fabric.NewHeader(ch, ci)
	msg := fabric.NewMsgCreateClient(ca.clientID, h, ca.signer)
	require.NoError(t, msg.ValidateBasic())
	return &msg
}

func (ca TestChaincodeApp) createMsgUpdateClient(t *testing.T) *fabric.MsgUpdateClient {
	seq := ca.seq + 1
	var sigs [][]byte
	var pcBytes []byte = makePolicy([]string{"SampleOrg"})
	ci := fabric.NewChaincodeInfo(ca.fabChannelID, ca.fabChaincodeID, pcBytes, sigs)
	// TODO must use endorsed sequence
	ch := fabric.NewChaincodeHeader(seq, tmtime.Now().Unix(), fabric.Proof{})
	proof, err := tests.MakeProof(ca.endorser, commitment.MakeSequenceCommitmentEntryKey(seq), ch.Sequence.Bytes())
	require.NoError(t, err)
	ch.Proof = *proof
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

func (ca TestChaincodeApp) getEndorsedCurrentSequence(ctx contractapi.TransactionContextInterface) (*commitment.Sequence, error) {
	entry, err := ca.cc.EndorseSequenceCommitment(ctx)
	if err != nil {
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
	return 1, bz, nil
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
	return 1, bz, nil
}

package chaincode_test

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc-transfer/types"
	connection "github.com/cosmos/cosmos-sdk/x/ibc/03-connection"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	"github.com/datachainlab/fabric-ibc/app"
	"github.com/datachainlab/fabric-ibc/chaincode"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/datachainlab/fabric-ibc/example"
	"github.com/datachainlab/fabric-ibc/tests"
	"github.com/datachainlab/fabric-ibc/x/compat"
	fabric "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/msp"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TestChaincodeApp struct {
	cc *chaincode.IBCChaincode

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

	// MSP
	mspConfig msppb.MSPConfig
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
	mspConfig msppb.MSPConfig,
) TestChaincodeApp {
	cdc, _ := example.MakeCodecs()
	cc := chaincode.NewIBCChaincode(example.AppProvider, chaincode.DefaultDBProvider)
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

		mspConfig: mspConfig,
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
	events, err := ca.cc.RunMsg(stub, makeStdTxBytes(ca.cdc, ca.prvKey, msgs...))
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
	seq, err := ca.cc.UpdateSequence(ctx)
	if err != nil {
		return nil, err
	}
	ca.seq = seq
	return seq, nil
}

func (ca TestChaincodeApp) createMsgCreateClient(t *testing.T, ctx contractapi.TransactionContextInterface) *fabric.MsgCreateClient {
	mspID := "SampleOrgMSP"
	var pcBytes []byte = makePolicy([]string{mspID})
	ci := fabric.NewChaincodeInfo(ca.fabChannelID, ca.fabChaincodeID, pcBytes, pcBytes, nil)
	ch := fabric.NewChaincodeHeader(ca.seq.Value, ca.seq.Timestamp, fabric.CommitmentProof{})
	conf, _ := proto.Marshal(&ca.mspConfig)
	mps := fabric.NewMSPPolicies([]fabrictypes.MSPPolicy{fabric.NewMSPPolicy(fabrictypes.TypeCreate, mspID, pcBytes, &fabric.MessageProof{})})
	mcs := fabric.NewMSPConfigs([]fabrictypes.MSPConfig{fabric.NewMSPConfig(fabrictypes.TypeCreate, mspID, conf, &fabric.MessageProof{})})
	h := fabric.NewHeader(ch, ci, mps, mcs)
	msg := fabric.NewMsgCreateClient(ca.clientID, h, ca.signer)
	require.NoError(t, msg.ValidateBasic())
	return &msg
}

func (ca TestChaincodeApp) createMsgUpdateClient(t *testing.T) *fabric.MsgUpdateClient {
	mspID := "SampleOrgMSP"
	var pcBytes []byte = makePolicy([]string{mspID})
	ci := fabric.NewChaincodeInfo(ca.fabChannelID, ca.fabChaincodeID, pcBytes, pcBytes, nil)
	mproof, err := tests.MakeMessageProof(ca.endorser, ci.GetSignBytes())
	require.NoError(t, err)
	ci.Proof = mproof
	cproof, err := tests.MakeCommitmentProof(ca.endorser, commitment.MakeSequenceCommitmentEntryKey(ca.seq.Value), ca.seq.Bytes())
	require.NoError(t, err)
	ch := fabric.NewChaincodeHeader(ca.seq.Value, ca.seq.Timestamp, *cproof)
	conf, _ := proto.Marshal(&ca.mspConfig)
	policy := fabric.NewMSPPolicy(fabrictypes.TypeUpdate, mspID, pcBytes, nil)
	policyProof, err := tests.MakeMessageProof(ca.endorser, policy.GetSignBytes())
	require.NoError(t, err)
	policy.Proof = policyProof
	mps := fabric.NewMSPPolicies([]fabrictypes.MSPPolicy{policy})

	mconf := fabric.NewMSPConfig(fabrictypes.TypeUpdate, mspID, conf, nil)
	mconfProof, err := tests.MakeMessageProof(ca.endorser, mconf.GetSignBytes())
	require.NoError(t, err)
	mconf.Proof = mconfProof
	mcs := fabric.NewMSPConfigs([]fabrictypes.MSPConfig{mconf})

	h := fabric.NewHeader(ch, ci, mps, mcs)
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
	ce, err := ca.cc.EndorseSequenceCommitment(ctx)
	if err != nil {
		return nil, err
	}
	entry, err := commitment.EntryFromCommitment(ce)
	if err != nil {
		return nil, err
	}
	var seq commitment.Sequence
	if err := proto.Unmarshal(entry.Value, &seq); err != nil {
		return nil, err
	}
	return &seq, nil
}

func (ca TestChaincodeApp) makeMockEndorsedCommitmentProof(entry *commitment.Entry) (*fabric.CommitmentProof, error) {
	return tests.MakeCommitmentProof(ca.endorser, entry.Key, entry.Value)
}

func (ca TestChaincodeApp) makeProofConsensus(ctx contractapi.TransactionContextInterface, clientID string, height uint64) (uint64, []byte, error) {
	ce, err := ca.cc.EndorseConsensusStateCommitment(ctx, clientID, height)
	entry, err := commitment.EntryFromCommitment(ce)
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

func (ca TestChaincodeApp) makeProofConnectionState(ctx contractapi.TransactionContextInterface, connectionID string) (uint64, []byte, error) {
	ce, err := ca.cc.EndorseConnectionState(ctx, connectionID)
	if err != nil {
		return 0, nil, err
	}
	entry, err := commitment.EntryFromCommitment(ce)
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
	ce, err := ca.cc.EndorseChannelState(ctx, portID, channelID)
	if err != nil {
		return 0, nil, err
	}
	entry, err := commitment.EntryFromCommitment(ce)
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
	ce, err := ca.cc.EndorsePacketCommitment(ctx, portID, channelID, sequence)
	if err != nil {
		return 0, nil, err
	}
	entry, err := commitment.EntryFromCommitment(ce)
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
	ce, err := ca.cc.EndorsePacketAcknowledgement(ctx, portID, channelID, sequence)
	if err != nil {
		return 0, nil, err
	}
	entry, err := commitment.EntryFromCommitment(ce)
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
	ce, err := ca.cc.EndorseNextSequenceRecv(ctx, portID, channelID)
	if err != nil {
		return 0, nil, err
	}
	entry, err := commitment.EntryFromCommitment(ce)
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

func (ca TestChaincodeApp) query(ctx contractapi.TransactionContextInterface, req app.RequestQuery) (*app.ResponseQuery, error) {
	bz, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return ca.cc.Query(ctx, string(bz))
}

func TestResponseSerializer(t *testing.T) {
	require := require.New(t)

	cc := chaincode.NewIBCChaincode(example.AppProvider, chaincode.DefaultDBProvider)
	chaincode, err := contractapi.NewChaincode(cc)
	require.NoError(err)

	stub0 := compat.MakeFakeStub()

	// Initialize chaincode
	res := chaincode.Init(stub0)
	require.EqualValues(200, res.Status, res.String())

	now := time.Now()
	stub0.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: now.Unix()}, nil)
	stub0.GetFunctionAndParametersReturns("InitChaincode", []string{"{}"})
	res = chaincode.Invoke(stub0)
	require.EqualValues(200, res.Status, res.String())

	// UpdateSequence
	now = now.Add(10 * time.Second)
	stub0.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: now.Unix()}, nil)
	stub0.GetFunctionAndParametersReturns("UpdateSequence", nil)
	res = chaincode.Invoke(stub0)
	require.EqualValues(200, res.Status, res.String())

	// EndorseSequenceCommitment
	now = now.Add(10 * time.Second)
	stub0.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: now.Unix()}, nil)
	stub0.GetFunctionAndParametersReturns("EndorseSequenceCommitment", []string{"2"})
	res = chaincode.Invoke(stub0)
	require.EqualValues(200, res.Status, res.String())

	// Query
	bz := channeltypes.SubModuleCdc.MustMarshalJSON(channeltypes.QueryAllChannelsParams{Limit: 100, Page: 1})
	req := app.RequestQuery{Data: string(bz), Path: "/custom/ibc/channel/channels"}
	jbz, err := json.Marshal(req)
	require.NoError(err)
	now = now.Add(10 * time.Second)
	stub0.GetTxTimestampReturns(&timestamppb.Timestamp{Seconds: now.Unix()}, nil)
	stub0.GetFunctionAndParametersReturns("Query", []string{string(jbz)})
	res = chaincode.Invoke(stub0)
	require.EqualValues(200, res.Status, res.String())
}

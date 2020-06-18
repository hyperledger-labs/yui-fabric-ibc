package chaincode

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	connection "github.com/cosmos/cosmos-sdk/x/ibc/03-connection"
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
}

func MakeTestChaincodeApp(
	prvKey crypto.PrivKey,
	fabChannelID string,
	fabChaincodeID fabrictypes.ChaincodeID,
	endorser msp.SigningIdentity,
	clientID string,
	connectionID string,
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
	}
}

func (ca TestChaincodeApp) runMsg(stub shim.ChaincodeStubInterface, msgs ...sdk.Msg) error {
	return ca.cc.runner.RunMsg(stub, string(makeStdTxBytes(ca.cdc, ca.prvKey, msgs...)))
}

func (ca TestChaincodeApp) createMsgCreateClient(t *testing.T) *fabric.MsgCreateClient {
	var pcBytes []byte = makePolicy([]string{"SampleOrg"})
	ci := fabric.NewChaincodeInfo(ca.fabChannelID, ccid, pcBytes, nil)
	ch := fabric.NewChaincodeHeader(ca.seq, tmtime.Now().UnixNano(), fabric.Proof{})
	h := fabric.NewHeader(ch, ci)
	msg := fabric.NewMsgCreateClient(clientID0, h, ca.signer)
	if err := msg.ValidateBasic(); err != nil {
		require.NoError(t, err)
	}
	return &msg
}

func (ca TestChaincodeApp) updateMsgCreateClient(t *testing.T) *fabric.MsgUpdateClient {
	seq := ca.seq + 1
	var sigs [][]byte
	var pcBytes []byte = makePolicy([]string{"SampleOrg"})
	ci := fabric.NewChaincodeInfo(fabchannelID, ccid, pcBytes, sigs)
	ch := fabric.NewChaincodeHeader(seq, tmtime.Now().UnixNano(), fabric.Proof{})
	proof, err := tests.MakeProof(ca.endorser, commitment.MakeSequenceCommitmentEntryKey(seq), ch.Sequence.Bytes())
	if err != nil {
		panic(err)
	}
	ch.Proof = *proof
	h := fabric.NewHeader(ch, ci)
	msg := fabric.NewMsgUpdateClient(clientID0, h, ca.signer)
	if err := msg.ValidateBasic(); err != nil {
		panic(err)
	}
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
	if err := msg.ValidateBasic(); err != nil {
		require.NoError(t, err)
	}
	return msg
}

// TODO fix this
func (ca TestChaincodeApp) createMsgConnectionOpenTry(
	t *testing.T,
	counterPartyCtx contractapi.TransactionContextInterface,
	counterParty TestChaincodeApp,
) *connection.MsgConnectionOpenTry {
	proofHeight, proofInit, err := counterParty.makeProofConnectionOpenInit(counterPartyCtx, counterParty.connectionID)
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
	if err := msg.ValidateBasic(); err != nil {
		require.NoError(t, err)
	}
	return msg
}

func (ca TestChaincodeApp) makeMockEndorsedCommitmentProof(entry *commitment.Entry) (*fabric.Proof, error) {
	return tests.MakeProof(ca.endorser, entry.Key, entry.Value)
}

func (ca TestChaincodeApp) makeProofConnectionOpenInit(ctx contractapi.TransactionContextInterface, connectionID string) (uint64, []byte, error) {
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

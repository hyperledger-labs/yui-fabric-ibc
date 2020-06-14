package chaincode

import (
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	connection "github.com/cosmos/cosmos-sdk/x/ibc/03-connection"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	"github.com/datachainlab/fabric-ibc/tests"
	"github.com/datachainlab/fabric-ibc/x/compat"
	fabric "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/common"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	msptesttools "github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/libs/log"
	tmtime "github.com/tendermint/tendermint/types/time"
	dbm "github.com/tendermint/tm-db"
)

func TestApp(t *testing.T) {
	require := require.New(t)

	// setup the MSP manager so that we can sign/verify
	require.NoError(msptesttools.LoadMSPSetupForTesting())
	cryptoProvider, err := sw.NewDefaultSecurityLevelWithKeystore(sw.NewDummyKeyStore())
	require.NoError(err)
	lcMSP := mspmgmt.GetLocalMSP(cryptoProvider)
	endorser, err := lcMSP.GetDefaultSigningIdentity()
	require.NoError(err)

	cdc, _ := MakeCodecs()
	logger := log.NewTMLogger(os.Stdout)
	// var dbProvider = staticTMDBProvider{db: &traceDB{dbm.NewMemDB()}}.Provider
	var dbProvider = DefaultDBProvider
	runner := NewAppRunner(logger, dbProvider)
	stub := compat.MakeFakeStub()

	prv := secp256k1.GenPrivKey()
	mb := NewMsgBuilder(prv, endorser)

	{
		msg, err := mb.makeMsgCreateClient(1)
		require.NoError(err)
		require.NoError(runner.RunMsg(stub, string(makeStdTxBytes(cdc, prv, msg))))
	}

	{
		msg, err := mb.makeMsgUpdateClient(2)
		require.NoError(err)
		require.NoError(runner.RunMsg(stub, string(makeStdTxBytes(cdc, prv, msg))))
	}

	{
		msg, err := mb.makeMsgConnectionOpenInit()
		require.NoError(err)
		require.NoError(runner.RunMsg(stub, string(makeStdTxBytes(cdc, prv, msg))))
	}

	// TODO add tests for other handshake step
	// BLOCKED BY: https://github.com/cosmos/cosmos-sdk/pull/6274
}

type MsgBuilder struct {
	signer   sdk.AccAddress
	endorser msp.SigningIdentity
}

func NewMsgBuilder(prv crypto.PrivKey, endorser msp.SigningIdentity) MsgBuilder {
	addr := prv.PubKey().Address()
	signer := sdk.AccAddress(addr)
	return MsgBuilder{
		signer:   signer,
		endorser: endorser,
	}
}

const (
	connectionID0 = "connection0"
	clientID0     = "ibcclient0"
	connectionID1 = "connection1"
	clientID1     = "ibcclient1"
)

const (
	fabchannelID = "dummyChannel"
)

var ccid = fabrictypes.ChaincodeID{
	Name:    "dummyCC",
	Version: "dummyVer",
}

func makePolicy(mspids []string) []byte {
	return protoutil.MarshalOrPanic(&common.ApplicationPolicy{
		Type: &common.ApplicationPolicy_SignaturePolicy{
			SignaturePolicy: policydsl.SignedByNOutOfGivenRole(int32(len(mspids)/2+1), msppb.MSPRole_MEMBER, mspids),
		},
	})
}

func (b MsgBuilder) makeMsgCreateClient(seq int64) (*fabric.MsgCreateClient, error) {
	var sigs [][]byte
	var pcBytes []byte = makePolicy([]string{"SampleOrg"})
	ci := fabric.NewChaincodeInfo(fabchannelID, ccid, pcBytes, sigs)
	ch := fabric.NewChaincodeHeader(seq, uint64(tmtime.Now().UnixNano()), fabric.Proof{})
	proof, err := tests.MakeProof(b.endorser, fabric.VerifyChaincodeHeaderPath(seq), ch.GetEndorseBytes())
	if err != nil {
		return nil, err
	}
	ch.Proof = *proof
	h := fabric.NewHeader(ch, ci)
	msg := fabric.NewMsgCreateClient(clientID0, h, b.signer)
	if err := msg.ValidateBasic(); err != nil {
		panic(err)
	}
	return &msg, nil
}

func (b MsgBuilder) makeMsgUpdateClient(seq int64) (*fabric.MsgUpdateClient, error) {
	var sigs [][]byte
	var pcBytes []byte = makePolicy([]string{"SampleOrg"})
	ci := fabric.NewChaincodeInfo(fabchannelID, ccid, pcBytes, sigs)
	ch := fabric.NewChaincodeHeader(seq, uint64(tmtime.Now().UnixNano()), fabric.Proof{})
	proof, err := tests.MakeProof(b.endorser, fabric.VerifyChaincodeHeaderPath(seq), ch.GetEndorseBytes())
	if err != nil {
		return nil, err
	}
	ch.Proof = *proof
	h := fabric.NewHeader(ch, ci)
	msg := fabric.NewMsgUpdateClient(clientID0, h, b.signer)
	if err := msg.ValidateBasic(); err != nil {
		panic(err)
	}
	return &msg, nil
}

func (b MsgBuilder) makeMsgConnectionOpenInit() (*connection.MsgConnectionOpenInit, error) {
	msg := connection.NewMsgConnectionOpenInit(connectionID0, clientID0, connectionID1, clientID1, commitmenttypes.NewMerklePrefix([]byte("ibc")), b.signer)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	return msg, nil
}

func makeStdTxBytes(cdc *std.Codec, prv crypto.PrivKey, msgs ...sdk.Msg) []byte {
	tx := auth.StdTx{
		Msgs: msgs,
		Signatures: []auth.StdSignature{
			{PubKey: prv.PubKey().Bytes(), Signature: make([]byte, 64)}, // FIXME set valid signature
		},
	}
	if err := tx.ValidateBasic(); err != nil {
		panic(err)
	}

	bz, err := cdc.MarshalJSON(tx)
	if err != nil {
		panic(err)
	}
	return bz
}

func tmDBProvider(_ shim.ChaincodeStubInterface) dbm.DB {
	return dbm.NewMemDB()
}

type staticTMDBProvider struct {
	db dbm.DB
}

func (p staticTMDBProvider) Provider(_ shim.ChaincodeStubInterface) dbm.DB {
	return p.db
}

package compat

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	localhost "github.com/cosmos/cosmos-sdk/x/ibc/09-localhost/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/datachainlab/fabric-ibc/tests"
	client "github.com/datachainlab/fabric-ibc/x/ibc/02-client"
	clientkeeper "github.com/datachainlab/fabric-ibc/x/ibc/02-client/keeper"
	fabric "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/common"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/hyperledger/fabric/common/policydsl"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	msptesttools "github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmtime "github.com/tendermint/tendermint/types/time"
)

type stakingKeeper struct {
}

func (sk *stakingKeeper) GetHistoricalInfo(ctx sdk.Context, height int64) (stakingtypes.HistoricalInfo, bool) {
	panic("not implemented error")
}

func (sk *stakingKeeper) UnbondingTime(ctx sdk.Context) time.Duration {
	return time.Hour
}

func NewStakingKeeper() client.StakingKeeper {
	return &stakingKeeper{}
}

func NewMockStub() shim.ChaincodeStubInterface {
	return MakeFakeStub()
}

func MakeCodec() *codec.Codec {
	cdc := codec.New()
	client.RegisterCodec(cdc)
	fabrictypes.RegisterCodec(cdc)
	localhost.RegisterCodec(cdc)
	return cdc
}

type LocalhostHeader struct {
	height uint64
}

func (h LocalhostHeader) GetHeight() uint64 {
	return h.height
}

func (LocalhostHeader) ClientType() clientexported.ClientType {
	return clientexported.Localhost
}

func TestIBC02Client(t *testing.T) {
	assert := assert.New(t)

	cdc := MakeCodec()
	sk := NewStakingKeeper()

	keys := sdk.NewKVStoreKeys(
		ibc.StoreKey,
		client.SubModuleName,
	)
	stub := NewMockStub()
	clientState := localhost.NewClientState("chain-id", 1)
	ctx := MakeContext(stub, keys)
	k := clientkeeper.NewKeeper(cdc, keys[ibc.StoreKey], sk)

	ctx.KVStore(keys[ibc.StoreKey])

	_, err := k.CreateClient(ctx, clientState, nil)
	assert.NoError(err)

	h := LocalhostHeader{height: 2}
	_, err = k.UpdateClient(ctx, clientState.GetID(), h)
	assert.NoError(err)
}

const (
	channelID = "dummyChannel"
	clientID  = "fabricclient"
)

var ccid = fabrictypes.ChaincodeID{
	Name:    "dummyCC",
	Version: "dummyVer",
}

func TestCodec(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	{
		var c fabrictypes.ChaincodeInfo
		bz, err := cdc.MarshalBinaryBare(c)
		assert.NoError(err)

		err = cdc.UnmarshalBinaryBare(bz, &c)
		assert.NoError(err)
	}

	{
		var c fabrictypes.ChaincodeHeader
		bz, err := cdc.MarshalBinaryBare(c)
		assert.NoError(err)

		err = cdc.UnmarshalBinaryBare(bz, &c)
		assert.NoError(err)
	}

	{
		var c fabric.ClientState
		bz, err := cdc.MarshalBinaryBare(c)
		assert.NoError(err)

		err = cdc.UnmarshalBinaryBare(bz, &c)
		assert.NoError(err)
	}

	{
		var cc = fabric.ClientState{ID: "myid"}
		var c clientexported.ClientState = cc

		bz, err := cdc.MarshalBinaryBare(c)
		assert.NoError(err)

		var ci clientexported.ClientState
		err = cdc.UnmarshalBinaryBare(bz, &ci)
		assert.NoError(err)
	}

	{
		var cc = localhost.ClientState{ID: "myid"}
		var c clientexported.ClientState = cc

		bz, err := cdc.MarshalBinaryBare(c)
		assert.NoError(err)

		var ci clientexported.ClientState
		err = cdc.UnmarshalBinaryBare(bz, &ci)
		assert.NoError(err)
	}
}

func TestCreateClient(t *testing.T) {
	require := require.New(t)

	// setup the MSP manager so that we can sign/verify
	require.NoError(msptesttools.LoadMSPSetupForTesting())
	cryptoProvider, err := sw.NewDefaultSecurityLevelWithKeystore(sw.NewDummyKeyStore())
	require.NoError(err)
	lcMSP := mspmgmt.GetLocalMSP(cryptoProvider)
	signer, err := lcMSP.GetDefaultSigningIdentity()
	require.NoError(err)

	/// Setup context
	keys := sdk.NewKVStoreKeys(
		ibc.StoreKey,
		client.SubModuleName,
	)
	stub := NewMockStub()
	ctx := MakeContext(stub, keys)

	cdc := MakeCodec()
	sk := NewStakingKeeper()
	clientKeeper := clientkeeper.NewKeeper(cdc, keys[ibc.StoreKey], sk)
	/// END

	var seq uint64 = 1
	// CreateClient
	{
		/// Build Msg
		var sigs [][]byte
		var pcBytes []byte = makePolicy([]string{"SampleOrg"})
		ci := fabric.NewChaincodeInfo(channelID, ccid, pcBytes, sigs)
		ch := fabric.NewChaincodeHeader(seq, tmtime.Now().UnixNano(), fabrictypes.Proof{})
		proof, err := tests.MakeProof(signer, commitment.MakeSequenceCommitmentKey(seq), ch.Sequence.Bytes())
		require.NoError(err)
		ch.Proof = *proof

		h := fabric.NewHeader(ch, ci)
		signer := sdk.AccAddress("signer0")
		msg := fabric.NewMsgCreateClient(clientID, h, signer)
		require.NoError(msg.ValidateBasic())
		/// END

		_, err = client.HandleMsgCreateClient(ctx, clientKeeper, msg)
		require.NoError(err)
		seq++
	}

	// UpdateClient
	{
		/// Build Msg
		var sigs [][]byte
		var pcBytes []byte = makePolicy([]string{"SampleOrg"})
		ci := fabric.NewChaincodeInfo(channelID, ccid, pcBytes, sigs)
		ch := fabric.NewChaincodeHeader(seq, tmtime.Now().UnixNano(), fabrictypes.Proof{})
		proof, err := tests.MakeProof(signer, commitment.MakeSequenceCommitmentKey(seq), ch.Sequence.Bytes())
		require.NoError(err)
		ch.Proof = *proof

		h := fabric.NewHeader(ch, ci)
		signer := sdk.AccAddress("signer0")
		msg := fabric.NewMsgUpdateClient(clientID, h, signer)
		require.NoError(msg.ValidateBasic())
		/// END

		_, err = client.HandleMsgUpdateClient(ctx, clientKeeper, msg)
		require.NoError(err)
		seq++
	}
}

func makeSignedDataList(pr *fabric.Proof) []*protoutil.SignedData {
	var sigSet []*protoutil.SignedData
	for i := 0; i < len(pr.Signatures); i++ {
		msg := make([]byte, len(pr.Proposal)+len(pr.Identities[i]))
		copy(msg[:len(pr.Proposal)], pr.Proposal)
		copy(msg[len(pr.Proposal):], pr.Identities[i])

		sigSet = append(
			sigSet,
			&protoutil.SignedData{
				Data:      msg,
				Identity:  pr.Identities[i],
				Signature: pr.Signatures[i],
			},
		)
	}
	return sigSet
}

func makePolicy(mspids []string) []byte {
	return protoutil.MarshalOrPanic(&common.ApplicationPolicy{
		Type: &common.ApplicationPolicy_SignaturePolicy{
			SignaturePolicy: policydsl.SignedByNOutOfGivenRole(int32(len(mspids)/2+1), msppb.MSPRole_MEMBER, mspids),
		},
	})
}

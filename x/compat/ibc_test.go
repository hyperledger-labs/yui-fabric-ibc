package compat

import (
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	localhost "github.com/cosmos/cosmos-sdk/x/ibc/09-localhost/types"
	ibchost "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/datachainlab/fabric-ibc/tests"
	client "github.com/datachainlab/fabric-ibc/x/ibc/02-client"
	clientkeeper "github.com/datachainlab/fabric-ibc/x/ibc/02-client/keeper"
	fabric "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric"
	fabrictests "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/tests"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/common"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
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

func MakeContext(stub shim.ChaincodeStubInterface, keys map[string]*sdk.KVStoreKey) sdk.Context {
	cms := store.NewCommitMultiStore(NewDB(stub))
	for _, key := range keys {
		cms.MountStoreWithDB(key, sdk.StoreTypeIAVL, nil)
	}
	if err := cms.LoadLatestVersion(); err != nil {
		panic(err)
	}
	return sdk.NewContext(cms, abci.Header{}, false, log.NewTMLogger(os.Stdout))
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

	conf, err := fabrictypes.DefaultConfig()
	require.NoError(err)
	mspID := "SampleOrgMSP"
	// setup the MSP manager so that we can sign/verify
	mconf, bconf, err := fabrictests.GetLocalMspConfig(conf.MSPsDir, mspID)
	require.NoError(err)
	lcMSP, err := fabrictests.SetupLocalMsp(mconf, bconf)
	require.NoError(err)
	signer, err := lcMSP.GetDefaultSigningIdentity()
	require.NoError(err)

	/// Setup context
	keys := sdk.NewKVStoreKeys(
		ibchost.StoreKey,
		client.SubModuleName,
	)
	stub := NewMockStub()
	ctx := MakeContext(stub, keys)

	cdc := MakeCodec()
	sk := NewStakingKeeper()
	csk := fabric.NewConsensusStateKeeper(stub, nil)
	clientKeeper := clientkeeper.NewKeeper(cdc, keys[ibchost.StoreKey], sk, csk)
	/// END

	var seq uint64 = 1
	// CreateClient
	{
		/// Build Msg
		var pcBytes []byte = makePolicy([]string{mspID})
		ci := fabric.NewChaincodeInfo(channelID, ccid, pcBytes, pcBytes, nil)
		ch := fabric.NewChaincodeHeader(seq, tmtime.Now().UnixNano(), fabrictypes.CommitmentProof{})
		proof, err := tests.MakeCommitmentProof(signer, commitment.MakeSequenceCommitmentEntryKey(seq), ch.Sequence.Bytes())
		require.NoError(err)
		ch.Proof = *proof

		err = fabrictests.GetVerifyingConfig(mconf)
		require.NoError(err)
		conf, err := proto.Marshal(mconf)
		require.NoError(err)

		mhs := fabric.NewMSPHeaders([]fabrictypes.MSPHeader{fabric.NewMSPHeader(fabrictypes.MSPHeaderTypeCreate, mspID, conf, pcBytes, &fabric.MessageProof{})})
		h := fabric.NewHeader(&ch, &ci, &mhs)
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
		var pcBytes []byte = makePolicy([]string{mspID})
		ci := fabric.NewChaincodeInfo(channelID, ccid, pcBytes, pcBytes, nil)
		mProof, err := tests.MakeMessageProof(signer, ci.GetSignBytes())
		require.NoError(err)
		ci.Proof = mProof
		ch := fabric.NewChaincodeHeader(seq, tmtime.Now().UnixNano(), fabrictypes.CommitmentProof{})
		cproof, err := tests.MakeCommitmentProof(signer, commitment.MakeSequenceCommitmentEntryKey(seq), ch.Sequence.Bytes())
		require.NoError(err)
		ch.Proof = *cproof

		err = fabrictests.GetVerifyingConfig(mconf)
		require.NoError(err)

		h := fabric.NewHeader(&ch, &ci, nil)
		signer := sdk.AccAddress("signer0")
		msg := fabric.NewMsgUpdateClient(clientID, h, signer)
		require.NoError(msg.ValidateBasic())
		/// END

		_, err = client.HandleMsgUpdateClient(ctx, clientKeeper, msg)
		require.NoError(err)
		seq++
	}
}

func TestClientWithMSPHeaders(t *testing.T) {
	require := require.New(t)

	conf, err := fabrictypes.DefaultConfig()
	require.NoError(err)
	mspID := "SampleOrgMSP"
	// setup the MSP manager so that we can sign/verify
	mconf, bconf, err := fabrictests.GetLocalMspConfig(conf.MSPsDir, mspID)
	require.NoError(err)
	lcMSP, err := fabrictests.SetupLocalMsp(mconf, bconf)
	require.NoError(err)
	signer, err := lcMSP.GetDefaultSigningIdentity()
	require.NoError(err)

	/// Setup context
	keys := sdk.NewKVStoreKeys(
		ibchost.StoreKey,
		client.SubModuleName,
	)
	stub := NewMockStub()
	ctx := MakeContext(stub, keys)

	cdc := MakeCodec()
	sk := NewStakingKeeper()
	csk := fabric.NewConsensusStateKeeper(stub, nil)
	clientKeeper := clientkeeper.NewKeeper(cdc, keys[ibchost.StoreKey], sk, csk)
	/// END

	var seq uint64 = 1
	// CreateClient
	{
		/// Build Msg
		var pcBytes []byte = makePolicy([]string{mspID})
		ci := fabric.NewChaincodeInfo(channelID, ccid, pcBytes, pcBytes, nil)
		ch := fabric.NewChaincodeHeader(seq, tmtime.Now().UnixNano(), fabrictypes.CommitmentProof{})
		proof, err := tests.MakeCommitmentProof(signer, commitment.MakeSequenceCommitmentEntryKey(seq), ch.Sequence.Bytes())
		require.NoError(err)
		ch.Proof = *proof

		err = fabrictests.GetVerifyingConfig(mconf)
		require.NoError(err)
		conf, err := proto.Marshal(mconf)
		require.NoError(err)
		mhs := fabric.NewMSPHeaders([]fabrictypes.MSPHeader{fabric.NewMSPHeader(fabrictypes.MSPHeaderTypeCreate, mspID, conf, pcBytes, &fabric.MessageProof{})})
		h := fabric.NewHeader(&ch, &ci, &mhs)
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
		var pcBytes []byte = makePolicy([]string{mspID})
		ci := fabric.NewChaincodeInfo(channelID, ccid, pcBytes, pcBytes, nil)
		mProof, err := tests.MakeMessageProof(signer, ci.GetSignBytes())
		require.NoError(err)
		ci.Proof = mProof
		ch := fabric.NewChaincodeHeader(seq, tmtime.Now().UnixNano(), fabrictypes.CommitmentProof{})
		cproof, err := tests.MakeCommitmentProof(signer, commitment.MakeSequenceCommitmentEntryKey(seq), ch.Sequence.Bytes())
		require.NoError(err)
		ch.Proof = *cproof

		err = fabrictests.GetVerifyingConfig(mconf)
		require.NoError(err)

		pcBytes = makePolicy([]string{mspID, "OTHER_MSP"})
		mh := fabric.NewMSPHeader(fabrictypes.MSPHeaderTypeUpdatePolicy, mspID, nil, pcBytes, &fabric.MessageProof{})
		mhProof, err := tests.MakeMessageProof(signer, mh.GetSignBytes())
		require.NoError(err)
		mh.Proof = mhProof
		mhs := fabric.NewMSPHeaders([]fabrictypes.MSPHeader{mh})
		h := fabric.NewHeader(&ch, &ci, &mhs)
		signer := sdk.AccAddress("signer0")
		msg := fabric.NewMsgUpdateClient(clientID, h, signer)
		require.NoError(msg.ValidateBasic())
		/// END

		_, err = client.HandleMsgUpdateClient(ctx, clientKeeper, msg)
		require.NoError(err)
		seq++
	}
}

func makePolicy(mspids []string) []byte {
	return protoutil.MarshalOrPanic(&common.ApplicationPolicy{
		Type: &common.ApplicationPolicy_SignaturePolicy{
			SignaturePolicy: policydsl.SignedByNOutOfGivenRole(int32(len(mspids)/2+1), msppb.MSPRole_MEMBER, mspids),
		},
	})
}

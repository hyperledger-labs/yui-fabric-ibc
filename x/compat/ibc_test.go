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
	client "github.com/datachainlab/fabric-ibc/x/ibc/02-client"
	clientkeeper "github.com/datachainlab/fabric-ibc/x/ibc/02-client/keeper"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	tmtime "github.com/tendermint/tendermint/types/time"

	fabric "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
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

var ccid = peer.ChaincodeID{
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
	assert := assert.New(t)

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

	// CreateClient
	{
		/// Build Msg
		ch := fabric.NewChaincodeHeader(1, tmtime.Now(), fabric.Proof{})
		var sigs [][]byte
		var pcBytes []byte
		ci := fabric.NewChaincodeInfo(channelID, ccid, pcBytes, sigs)

		h := fabric.NewHeader(ch, ci)
		signer := sdk.AccAddress("signer0")
		msg := fabric.NewMsgCreateClient(clientID, h, signer)
		assert.NoError(msg.ValidateBasic())
		/// END

		_, err := client.HandleMsgCreateClient(ctx, clientKeeper, msg)
		assert.NoError(err)
	}

	// UpdateClient
	{
		/// Build Msg
		ch := fabric.NewChaincodeHeader(2, tmtime.Now(), fabric.Proof{})
		var sigs [][]byte
		var pcBytes []byte = makePolicy([]string{"org0"})
		ci := fabric.NewChaincodeInfo(channelID, ccid, pcBytes, sigs)

		h := fabric.NewHeader(ch, ci)
		signer := sdk.AccAddress("signer0")
		msg := fabric.NewMsgUpdateClient(clientID, h, signer)
		assert.NoError(msg.ValidateBasic())
		/// END

		_, err := client.HandleMsgUpdateClient(ctx, clientKeeper, msg)
		assert.NoError(err)
	}
}

func makePolicy(mspids []string) []byte {
	return protoutil.MarshalOrPanic(&common.ApplicationPolicy{
		Type: &common.ApplicationPolicy_SignaturePolicy{
			SignaturePolicy: policydsl.SignedByNOutOfGivenRole(int32(len(mspids)/2+1), msppb.MSPRole_MEMBER, mspids),
		},
	})
}

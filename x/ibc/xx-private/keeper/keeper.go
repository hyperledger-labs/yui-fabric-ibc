package keeper

import (
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channelexported "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/exported"
	"github.com/datachainlab/fabric-ibc/x/ibc/xx-private/types"
)

var _ types.ChannelKeeper = (*PrivateChannelKeeper)(nil)

type PrivateChannelKeeper struct {
	types.ChannelKeeper
}

func NewPrivateChannelKeeper(channelKeeper types.ChannelKeeper) PrivateChannelKeeper {
	return PrivateChannelKeeper{ChannelKeeper: channelKeeper}
}

func (k PrivateChannelKeeper) SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet channelexported.PacketI) error {
	// TODO implement
	return k.ChannelKeeper.SendPacket(ctx, channelCap, packet)
}

func (k PrivateChannelKeeper) RecvPacket(ctx sdk.Context, packet channelexported.PacketI, proof []byte, proofHeight uint64) (channelexported.PacketI, error) {
	// TODO implement
	return k.ChannelKeeper.RecvPacket(ctx, packet, proof, proofHeight)
}

// txID: TransactionID on fabric
func MakePrivateDataKey(txID string, seq uint32) string {
	return fmt.Sprintf("%v/%v", txID, seq)
}

func MakePrivatePacketData(key string, data []byte) []byte {
	h := hash(data)
	return []byte(fmt.Sprintf("%v/%X", key, h))
}

func hash(v []byte) []byte {
	h := sha256.Sum256(v)
	return h[:]
}

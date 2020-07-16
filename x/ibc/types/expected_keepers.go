package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	channelexported "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/exported"
)

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	RecvPacket(ctx sdk.Context, packet channelexported.PacketI, proof []byte, proofHeight uint64) (channelexported.PacketI, error)
	AcknowledgePacket(ctx sdk.Context, packet channelexported.PacketI, acknowledgement []byte, proof []byte, proofHeight uint64) (channelexported.PacketI, error)
	TimeoutPacket(ctx sdk.Context, packet channelexported.PacketI, proof []byte, proofHeight, nextSequenceRecv uint64) (channelexported.PacketI, error)
}

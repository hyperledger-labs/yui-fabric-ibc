package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	channelexported "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/exported"
)

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channel.Channel, found bool)
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
	SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet channelexported.PacketI) error
	PacketExecuted(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet channelexported.PacketI, acknowledgement []byte) error
	RecvPacket(ctx sdk.Context, packet channelexported.PacketI, proof []byte, proofHeight uint64) (channelexported.PacketI, error)
	AcknowledgePacket(ctx sdk.Context, packet channelexported.PacketI, acknowledgement []byte, proof []byte, proofHeight uint64) (channelexported.PacketI, error)
}

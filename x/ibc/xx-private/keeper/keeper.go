package keeper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	channelexported "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/exported"
	authtypes "github.com/datachainlab/fabric-ibc/x/auth/types"
	"github.com/datachainlab/fabric-ibc/x/ibc/xx-private/types"
)

var _ types.ChannelKeeper = (*Keeper)(nil)

// Keeper wraps ChannelKeeper to shield the data in packets it sends and receives
// Currently, we only support fabric's private data
type Keeper struct {
	types.ChannelKeeper
	collectionRouter CollectionRouter
}

// NewKeeper returns a new Keeper
func NewKeeper(channelKeeper types.ChannelKeeper, collectionRouter CollectionRouter) Keeper {
	return Keeper{ChannelKeeper: channelKeeper, collectionRouter: collectionRouter}
}

// SendPacket wraps ChannelKeeper.SendPacket to shield the packet data and store it using private data
func (k Keeper) SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet channelexported.PacketI) error {
	collection, useCollection := k.collectionRouter.Route(ctx, packet)
	if !useCollection {
		return k.ChannelKeeper.SendPacket(ctx, channelCap, packet)
	}

	// TODO currently, we only support channel.Packet type
	p := packet.(channel.Packet)

	key := KeyPrivateData(p)
	data := p.GetData()
	if err := authtypes.StubFromContext(ctx).PutPrivateData(collection, key, data); err != nil {
		return err
	}
	p.Data = MakePrivatePacketData(key, data)
	return k.ChannelKeeper.SendPacket(ctx, channelCap, p)
}

// CollectionRouter is a router returns a collection that given packet should be stored at
type CollectionRouter interface {
	Route(ctx sdk.Context, packet channelexported.PacketI) (collection string, useCollection bool)
}

// TODO should we support dynamic key?
const privKey = "fabricprivatedata"

// RecvPacket wraps ChannelKeeper.RecvPacket to unshield the packet data. Original data is given from transieint map.
func (k Keeper) RecvPacket(ctx sdk.Context, packet channelexported.PacketI, proof []byte, proofHeight uint64) (channelexported.PacketI, error) {
	_, useCollection := k.collectionRouter.Route(ctx, packet)
	if !useCollection {
		return k.ChannelKeeper.RecvPacket(ctx, packet, proof, proofHeight)
	}

	// TODO currently, we only support channel.Packet type
	p := packet.(channel.Packet)

	ts, err := authtypes.StubFromContext(ctx).GetTransient()
	if err != nil {
		return nil, err
	}
	v, ok := ts[privKey]
	if !ok {
		return nil, fmt.Errorf("transient data '%v' not found", privKey)
	}
	h := hash(v)
	_, hash, err := ParseShieldedData(p.GetData())
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(h, hash) {
		return nil, fmt.Errorf("transient data '%v' doesn't match packet data", privKey)
	}
	p.Data = v
	return k.ChannelKeeper.RecvPacket(ctx, p, proof, proofHeight)
}

// KeyPrivateData returns a private collection key that original data is stored at
func KeyPrivateData(packet channelexported.PacketI) string {
	return fmt.Sprintf("/private/%v/%v/%v", packet.GetSourceChannel(), packet.GetSourcePort(), packet.GetSequence())
}

// MakePrivatePacketData returns bytes concat a collection key and shielded data
// TODO add salt support?
func MakePrivatePacketData(key string, data []byte) []byte {
	h := hash(data)
	return []byte(fmt.Sprintf("%v/%X", key, h))
}

// ParseShieldedData parses a shielded data to returns a collection key and hash of original data
func ParseShieldedData(shieldedData []byte) (key string, hash []byte, err error) {
	parts := strings.Split(string(shieldedData), "/")
	hexHash := parts[len(parts)-1]

	hash, err = hex.DecodeString(hexHash)
	if err != nil {
		return "", nil, err
	}
	return strings.Join(parts[0:len(parts)-1-1], "/"), hash, nil
}

// hash returns a hash of given value
// TODO can we use same function used by fabric's private data?
func hash(v []byte) []byte {
	h := sha256.Sum256(v)
	return h[:]
}

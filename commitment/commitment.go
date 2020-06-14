package commitment

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	commitmentexported "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/exported"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
)

type Entry struct {
	Key   string
	Value []byte
}

func MakePacketCommitmentEntry(
	ctx sdk.Context,
	channelKeeper channel.Keeper,
	prefix commitmentexported.Prefix,
	portID, channelID string, sequence uint64) (*Entry, error) {

	cmbz := channelKeeper.GetPacketCommitment(ctx, portID, channelID, sequence)
	if cmbz == nil {
		return nil, errors.New("commitment not found")
	}
	key := MakePacketCommitmentEntryKey(prefix, portID, channelID, sequence)
	return &Entry{
		Key:   key,
		Value: cmbz,
	}, nil
}

func MakePacketCommitmentEntryKey(
	prefix commitmentexported.Prefix,
	portID, channelID string, sequence uint64,
) string {
	key := host.PacketCommitmentPath(portID, channelID, sequence)
	return MakeEntryKey(key, prefix)
}

func MakeEntryKey(key string, prefix commitmentexported.Prefix) string {
	return fmt.Sprintf("e/k:%v/%v", string(prefix.Bytes()), key)
}

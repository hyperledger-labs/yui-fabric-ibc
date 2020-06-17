package commitment

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	commitmentexported "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/exported"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
)

type Entry struct {
	Key   string
	Value []byte
}

/// ConsensusStateCommitment

func MakeConsensusStateCommitmentEntry(
	prefix commitmentexported.Prefix,
	clientID string, height uint64, consensusStateBytes []byte,
) (*Entry, error) {
	key := MakeConsensusStateCommitmentKey(prefix, clientID, height)
	return &Entry{
		Key:   key,
		Value: consensusStateBytes,
	}, nil
}

/// SequenceCommitment

func MakeSequenceCommitmentEntry(
	sequence *Sequence,
) (*Entry, error) {
	key := MakeSequenceCommitmentKey(sequence.Value)
	return &Entry{
		Key:   key,
		Value: sequence.Bytes(),
	}, nil
}

/// PacketCommitment

func MakePacketCommitmentEntry(
	prefix commitmentexported.Prefix,
	portID, channelID string, sequence uint64, packetCommitmentBytes []byte,
) (*Entry, error) {
	key := MakePacketCommitmentEntryKey(prefix, portID, channelID, sequence)
	return &Entry{
		Key:   key,
		Value: packetCommitmentBytes,
	}, nil
}

func MakePacketCommitmentEntryKey(
	prefix commitmentexported.Prefix,
	portID, channelID string, sequence uint64,
) string {
	key := host.PacketCommitmentPath(portID, channelID, sequence)
	return MakeEntryKey(prefix, key)
}

/// PacketAcknowledgement

func MakePacketAcknowledgementEntry(
	ctx sdk.Context,
	channelKeeper channel.Keeper,
	prefix commitmentexported.Prefix,
	portID, channelID string, sequence uint64) (*Entry, error) {

	ackbz, ok := channelKeeper.GetPacketAcknowledgement(ctx, portID, channelID, sequence)
	if !ok {
		return nil, errors.New("acknowledgement packet not found")
	}
	key := MakePacketAcknowledgementEntryKey(prefix, portID, channelID, sequence)
	return &Entry{
		Key:   key,
		Value: ackbz,
	}, nil
}

func MakePacketAcknowledgementEntryKey(
	prefix commitmentexported.Prefix,
	portID, channelID string, sequence uint64,
) string {
	key := host.PacketAcknowledgementPath(portID, channelID, sequence)
	return MakeEntryKey(prefix, key)
}

/// PacketAcknowledgementAbsence

func MakePacketAcknowledgementAbsenceEntry(
	ctx sdk.Context,
	channelKeeper channel.Keeper,
	prefix commitmentexported.Prefix,
	portID, channelID string, sequence uint64) (*Entry, error) {

	_, ok := channelKeeper.GetPacketAcknowledgement(ctx, portID, channelID, sequence)
	if ok {
		return nil, errors.New("acknowledgement packet found")
	}
	key := MakePacketAcknowledgementAbsenceEntryKey(prefix, portID, channelID, sequence)
	return &Entry{
		Key:   key,
		Value: []byte{},
	}, nil
}

func MakePacketAcknowledgementAbsenceEntryKey(
	prefix commitmentexported.Prefix,
	portID, channelID string, sequence uint64,
) string {
	key := host.PacketAcknowledgementPath(portID, channelID, sequence)
	return MakeEntryKey(prefix, key)
}

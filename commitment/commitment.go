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

/// ConsensusStateCommitment

func MakeConsensusStateCommitmentEntry(
	prefix commitmentexported.Prefix,
	clientID string, height uint64, consensusStateBytes []byte,
) (*Entry, error) {
	key := MakeConsensusStateCommitmentEntryKey(prefix, clientID, height)
	return &Entry{
		Key:   key,
		Value: consensusStateBytes,
	}, nil
}

func MakeConsensusStateCommitmentEntryKey(prefix commitmentexported.Prefix, clientID string, height uint64) string {
	return fmt.Sprintf("h/k:%v/clients/%v/%v/commitment", string(prefix.Bytes()), clientID, host.ConsensusStatePath(height))
}

/// SequenceCommitment

func MakeSequenceCommitmentEntry(
	sequence *Sequence,
) (*Entry, error) {
	key := MakeSequenceCommitmentEntryKey(sequence.Value)
	return &Entry{
		Key:   key,
		Value: sequence.Bytes(),
	}, nil
}

func MakeSequenceCommitmentEntryKey(seq uint64) string {
	return fmt.Sprintf("h/_/seq/%v/commitment", seq)
}

/// ConnectionStateCommitment

func MakeConnectionStateCommitmentEntry(
	prefix commitmentexported.Prefix,
	connectionID string,
	connectionBytes []byte,
) (*Entry, error) {
	key := MakeConnectionStateCommitmentEntryKey(prefix, connectionID)
	return &Entry{
		Key:   key,
		Value: connectionBytes,
	}, nil
}

func MakeConnectionStateCommitmentEntryKey(prefix commitmentexported.Prefix, connectionID string) string {
	return fmt.Sprintf("h/k:%v/%v/commitment", string(prefix.Bytes()), host.ConnectionPath(connectionID))
}

/// ChannelStateCommitment

func MakeChannelStateCommitmentEntry(
	prefix commitmentexported.Prefix,
	portID string,
	channelID string,
	channelBytes []byte,
) (*Entry, error) {
	key := MakeChannelStateCommitmentEntryKey(prefix, portID, channelID)
	return &Entry{
		Key:   key,
		Value: channelBytes,
	}, nil
}

func MakeChannelStateCommitmentEntryKey(prefix commitmentexported.Prefix, portID, channelID string) string {
	return fmt.Sprintf("h/k:%v/%v/commitment", string(prefix.Bytes()), host.ChannelPath(portID, channelID))
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

/// NextSequenceRecv

func MakeNextSequenceRecvEntry(
	prefix commitmentexported.Prefix,
	portID, channelID string, seq uint64,
) (*Entry, error) {
	key := MakeNextSequenceRecvEntryKey(prefix, portID, channelID)
	return &Entry{
		Key:   key,
		Value: sdk.Uint64ToBigEndian(seq),
	}, nil
}

func MakeNextSequenceRecvEntryKey(
	prefix commitmentexported.Prefix,
	portID, channelID string,
) string {
	key := host.NextSequenceRecvPath(portID, channelID)
	return MakeEntryKey(prefix, key)
}

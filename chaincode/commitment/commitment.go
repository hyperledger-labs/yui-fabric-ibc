package commitment

import (
	"encoding/base64"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

type Entry struct {
	Key   string
	Value []byte
}

func (e Entry) ToCommitment() *CommitmentEntry {
	return &CommitmentEntry{
		Key:   e.Key,
		Value: base64.StdEncoding.EncodeToString(e.Value),
	}
}

func EntryFromCommitment(ce *CommitmentEntry) (*Entry, error) {
	v, err := base64.StdEncoding.DecodeString(ce.Value)
	if err != nil {
		return nil, err
	}
	return &Entry{
		Key:   ce.Key,
		Value: v,
	}, nil
}

/// ClientStateCommitment

func MakeClientStateCommitmentEntry(
	prefix exported.Prefix,
	clientID string,
	clientStateBytes []byte,
) (*Entry, error) {
	key := MakeClientStateCommitmentEntryKey(prefix, clientID)
	return &Entry{
		Key:   key,
		Value: clientStateBytes,
	}, nil
}

func MakeClientStateCommitmentEntryKey(
	prefix exported.Prefix,
	clientID string,
) string {
	return fmt.Sprintf("h/k:%v/clients/%v/clientState", string(prefix.Bytes()), clientID)
}

/// ConsensusStateCommitment

func MakeConsensusStateCommitmentEntry(
	prefix exported.Prefix,
	clientID string, height exported.Height, consensusStateBytes []byte,
) (*Entry, error) {
	key := MakeConsensusStateCommitmentEntryKey(prefix, clientID, height)
	return &Entry{
		Key:   key,
		Value: consensusStateBytes,
	}, nil
}

func MakeConsensusStateCommitmentEntryKey(prefix exported.Prefix, clientID string, height exported.Height) string {
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
	prefix exported.Prefix,
	connectionID string,
	connectionBytes []byte,
) (*Entry, error) {
	key := MakeConnectionStateCommitmentEntryKey(prefix, connectionID)
	return &Entry{
		Key:   key,
		Value: connectionBytes,
	}, nil
}

func MakeConnectionStateCommitmentEntryKey(prefix exported.Prefix, connectionID string) string {
	return fmt.Sprintf("h/k:%v/%v/commitment", string(prefix.Bytes()), host.ConnectionPath(connectionID))
}

/// ChannelStateCommitment

func MakeChannelStateCommitmentEntry(
	prefix exported.Prefix,
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

func MakeChannelStateCommitmentEntryKey(prefix exported.Prefix, portID, channelID string) string {
	return fmt.Sprintf("h/k:%v/%v/commitment", string(prefix.Bytes()), host.ChannelPath(portID, channelID))
}

/// PacketCommitment

func MakePacketCommitmentEntry(
	prefix exported.Prefix,
	portID, channelID string, sequence uint64, packetCommitmentBytes []byte,
) (*Entry, error) {
	key := MakePacketCommitmentEntryKey(prefix, portID, channelID, sequence)
	return &Entry{
		Key:   key,
		Value: packetCommitmentBytes,
	}, nil
}

func MakePacketCommitmentEntryKey(
	prefix exported.Prefix,
	portID, channelID string, sequence uint64,
) string {
	key := host.PacketCommitmentPath(portID, channelID, sequence)
	return MakeEntryKey(prefix, key)
}

/// PacketAcknowledgement

func MakePacketAcknowledgementEntry(
	prefix exported.Prefix,
	portID, channelID string, sequence uint64, ackBytes []byte) (*Entry, error) {
	key := MakePacketAcknowledgementEntryKey(prefix, portID, channelID, sequence)
	return &Entry{
		Key:   key,
		Value: ackBytes,
	}, nil
}

func MakePacketAcknowledgementEntryKey(
	prefix exported.Prefix,
	portID, channelID string, sequence uint64,
) string {
	key := host.PacketAcknowledgementPath(portID, channelID, sequence)
	return MakeEntryKey(prefix, key)
}

/// PacketReceiptAbsence

func MakePacketReceiptAbsenceEntry(
	prefix exported.Prefix,
	portID, channelID string, sequence uint64) (*Entry, error) {
	key := MakePacketReceiptAbsenceEntryKey(prefix, portID, channelID, sequence)
	return &Entry{
		Key:   key,
		Value: []byte{0},
	}, nil
}

func MakePacketReceiptAbsenceEntryKey(
	prefix exported.Prefix,
	portID, channelID string, sequence uint64,
) string {
	key := host.PacketReceiptPath(portID, channelID, sequence)
	return MakeEntryKey(prefix, key)
}

/// NextSequenceRecv

func MakeNextSequenceRecvEntry(
	prefix exported.Prefix,
	portID, channelID string, seq uint64,
) (*Entry, error) {
	key := MakeNextSequenceRecvEntryKey(prefix, portID, channelID)
	return &Entry{
		Key:   key,
		Value: sdk.Uint64ToBigEndian(seq),
	}, nil
}

func MakeNextSequenceRecvEntryKey(
	prefix exported.Prefix,
	portID, channelID string,
) string {
	key := host.NextSequenceRecvPath(portID, channelID)
	return MakeEntryKey(prefix, key)
}

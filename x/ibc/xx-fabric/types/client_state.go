package types

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types"
	connectionexported "github.com/cosmos/cosmos-sdk/x/ibc/03-connection/exported"
	connectiontypes "github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
	channelexported "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/exported"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	commitmentexported "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/exported"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
)

const (
	Fabric clientexported.ClientType = 100
)

const (
	ClientTypeFabric string = "fabric"
)

var _ clientexported.ClientState = ClientState{}

// ClientState requires (read-only) access to keys outside the client prefix.
type ClientState struct {
	ID                  string          `json:"id" yaml:"id"`
	LastChaincodeHeader ChaincodeHeader `json:"last_chaincode_header" yaml:"last_chaincode_header"`
	LastChaincodeInfo   ChaincodeInfo   `json:"last_chaincode_info" yaml:"last_chaincode_info"`
}

// NewClientState creates a new ClientState instance
func NewClientState(
	id string,
	header Header,
) ClientState {
	return ClientState{
		ID:                  id,
		LastChaincodeHeader: *header.ChaincodeHeader,
		LastChaincodeInfo:   *header.ChaincodeInfo,
	}
}

// GetID returns the loop-back client state identifier.
func (cs ClientState) GetID() string {
	return cs.ID
}

// GetChainID returns an empty string
func (cs ClientState) GetChainID() string {
	return cs.LastChaincodeInfo.ChainID()
}

// ClientType is localhost.
func (cs ClientState) ClientType() clientexported.ClientType {
	return Fabric
}

// GetLatestHeight returns the latest height stored.
func (cs ClientState) GetLatestHeight() uint64 {
	return uint64(cs.LastChaincodeHeader.Sequence)
}

// IsFrozen returns false.
func (cs ClientState) IsFrozen() bool {
	return false
}

// GetLatestTimestamp returns latest block time.
func (cs ClientState) GetLatestTimestamp() time.Time {
	return cs.LastChaincodeHeader.Timestamp
}

// Validate performs a basic validation of the client state fields.
func (cs ClientState) Validate() error {
	if strings.TrimSpace(cs.GetChainID()) == "" {
		return errors.New("chain id cannot be blank")
	}
	if cs.GetLatestHeight() <= 0 {
		return fmt.Errorf("height must be positive: %d", cs.GetLatestHeight())
	}
	return host.ClientIdentifierValidator(cs.ID)
}

// VerifyClientConsensusState verifies a proof of the consensus state of the
// Solo Machine client stored on the target machine.
func (cs ClientState) VerifyClientConsensusState(
	store sdk.KVStore,
	cdc *codec.Codec,
	root commitmentexported.Root,
	_ uint64,
	counterpartyClientIdentifier string,
	consensusHeight uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	consensusState clientexported.ConsensusState,
) error {
	clientPrefixedPath := "clients/" + counterpartyClientIdentifier + "/" + host.ConsensusStatePath(consensusHeight)
	path, err := commitmenttypes.ApplyPrefix(prefix, clientPrefixedPath)
	if err != nil {
		return err
	}

	if cs.IsFrozen() {
		return clienttypes.ErrClientFrozen
	}

	bz, err := cdc.MarshalBinaryBare(consensusState)
	if err != nil {
		return err
	}

	fabProof, ok := proof.(Proof)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "proof type %T is not type SignatureProof", proof)
	}

	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.PolicyBytes, fabProof, path.String(), bz); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// VerifyConnectionState verifies a proof of the connection state of the
// specified connection end stored locally.
func (cs ClientState) VerifyConnectionState(
	store sdk.KVStore,
	cdc codec.Marshaler,
	_ uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	connectionID string,
	connectionEnd connectionexported.ConnectionI,
	_ clientexported.ConsensusState,
) error {
	path, err := commitmenttypes.ApplyPrefix(prefix, host.ConnectionPath(connectionID))
	if err != nil {
		return err
	}

	if cs.IsFrozen() {
		return clienttypes.ErrClientFrozen
	}

	connection, ok := connectionEnd.(connectiontypes.ConnectionEnd)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "invalid connection type %T", connectionEnd)
	}

	bz, err := cdc.MarshalBinaryBare(&connection)
	if err != nil {
		return err
	}

	fabProof, ok := proof.(Proof)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "proof type %T is not type SignatureProof", proof)
	}

	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.PolicyBytes, fabProof, path.String(), bz); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// VerifyChannelState verifies a proof of the channel state of the specified
// channel end, under the specified port, stored on the target machine.
func (cs ClientState) VerifyChannelState(
	store sdk.KVStore,
	cdc codec.Marshaler,
	_ uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	portID,
	channelID string,
	channel channelexported.ChannelI,
	consensusState clientexported.ConsensusState,
) error {
	path, err := commitmenttypes.ApplyPrefix(prefix, host.ChannelPath(portID, channelID))
	if err != nil {
		return err
	}

	if cs.IsFrozen() {
		return clienttypes.ErrClientFrozen
	}

	channelEnd, ok := channel.(channeltypes.Channel)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "invalid channel type %T", channel)
	}

	bz, err := cdc.MarshalBinaryBare(&channelEnd)
	if err != nil {
		return err
	}

	fabProof, ok := proof.(Proof)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "proof type %T is not type SignatureProof", proof)
	}

	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.PolicyBytes, fabProof, path.String(), bz); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}

	return nil
}

// VerifyPacketCommitment verifies a proof of an outgoing packet commitment at
// the specified port, specified channel, and specified sequence.
func (cs ClientState) VerifyPacketCommitment(
	store sdk.KVStore,
	_ uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
	_ clientexported.ConsensusState,
) error {
	path, err := commitmenttypes.ApplyPrefix(prefix, host.PacketCommitmentPath(portID, channelID, sequence))
	if err != nil {
		return err
	}
	if cs.IsFrozen() {
		return clienttypes.ErrClientFrozen
	}

	fabProof, ok := proof.(Proof)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "proof type %T is not type SignatureProof", proof)
	}

	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.PolicyBytes, fabProof, path.String(), commitmentBytes); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// VerifyPacketAcknowledgement verifies a proof of an incoming packet
// acknowledgement at the specified port, specified channel, and specified sequence.
func (cs ClientState) VerifyPacketAcknowledgement(
	store sdk.KVStore,
	_ uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
	_ clientexported.ConsensusState,
) error {
	path, err := commitmenttypes.ApplyPrefix(prefix, host.PacketAcknowledgementPath(portID, channelID, sequence))
	if err != nil {
		return err
	}
	if cs.IsFrozen() {
		return clienttypes.ErrClientFrozen
	}

	fabProof, ok := proof.(Proof)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "proof type %T is not type SignatureProof", proof)
	}

	bz := channeltypes.CommitAcknowledgement(acknowledgement)
	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.PolicyBytes, fabProof, path.String(), bz); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// VerifyPacketAcknowledgementAbsence verifies a proof of the absence of an
// incoming packet acknowledgement at the specified port, specified channel, and
// specified sequence.
func (cs ClientState) VerifyPacketAcknowledgementAbsence(
	store sdk.KVStore,
	_ uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	portID,
	channelID string,
	sequence uint64,
	_ clientexported.ConsensusState,
) error {
	path, err := commitmenttypes.ApplyPrefix(prefix, host.PacketAcknowledgementPath(portID, channelID, sequence))
	if err != nil {
		return err
	}
	if cs.IsFrozen() {
		return clienttypes.ErrClientFrozen
	}
	fabProof, ok := proof.(Proof)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "proof type %T is not type SignatureProof", proof)
	}
	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.PolicyBytes, fabProof, path.String(), nil); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// VerifyNextSequenceRecv verifies a proof of the next sequence number to be
// received of the specified channel at the specified port.
func (cs ClientState) VerifyNextSequenceRecv(
	store sdk.KVStore,
	_ uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	portID,
	channelID string,
	nextSequenceRecv uint64,
	_ clientexported.ConsensusState,
) error {
	path, err := commitmenttypes.ApplyPrefix(prefix, host.NextSequenceRecvPath(portID, channelID))
	if err != nil {
		return err
	}
	if cs.IsFrozen() {
		return clienttypes.ErrClientFrozen
	}
	fabProof, ok := proof.(Proof)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "proof type %T is not type SignatureProof", proof)
	}

	bz := sdk.Uint64ToBigEndian(nextSequenceRecv)
	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.PolicyBytes, fabProof, path.String(), bz); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

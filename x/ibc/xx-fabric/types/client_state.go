package types

import (
	"errors"
	"fmt"
	"strings"

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
	"github.com/datachainlab/fabric-ibc/commitment"
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
	return cs.LastChaincodeInfo.GetChainID()
}

// ClientType is localhost.
func (cs ClientState) ClientType() clientexported.ClientType {
	return Fabric
}

// GetLatestHeight returns the latest height stored.
func (cs ClientState) GetLatestHeight() uint64 {
	return uint64(cs.LastChaincodeHeader.Sequence.Value)
}

// IsFrozen returns false.
func (cs ClientState) IsFrozen() bool {
	return false
}

// GetLatestTimestamp returns latest block time.
func (cs ClientState) GetLatestTimestamp() int64 {
	return cs.LastChaincodeHeader.Sequence.Timestamp
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
	_ sdk.KVStore,
	cdc codec.Marshaler,
	aminoCdc *codec.Codec,
	_ commitmentexported.Root,
	height uint64,
	counterpartyClientIdentifier string,
	consensusHeight uint64,
	prefix commitmentexported.Prefix,
	proof []byte,
	consensusState clientexported.ConsensusState,
) error {
	fabProof, err := sanitizeVerificationArgs(cdc, cs, height, prefix, proof, consensusState)
	if err != nil {
		return err
	}

	bz, err := aminoCdc.MarshalBinaryBare(consensusState)
	if err != nil {
		return err
	}

	key := commitment.MakeConsensusStateCommitmentEntryKey(prefix, counterpartyClientIdentifier, consensusHeight)
	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz); err != nil {
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
	height uint64,
	prefix commitmentexported.Prefix,
	proof []byte,
	connectionID string,
	connectionEnd connectionexported.ConnectionI,
	consensusState clientexported.ConsensusState,
) error {
	fabProof, err := sanitizeVerificationArgs(cdc, cs, height, prefix, proof, consensusState)
	if err != nil {
		return err
	}

	key := commitment.MakeConnectionStateCommitmentEntryKey(prefix, connectionID)

	connection, ok := connectionEnd.(connectiontypes.ConnectionEnd)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "invalid connection type %T", connectionEnd)
	}

	bz, err := cdc.MarshalBinaryBare(&connection)
	if err != nil {
		return err
	}

	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz); err != nil {
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
	height uint64,
	prefix commitmentexported.Prefix,
	proof []byte,
	portID,
	channelID string,
	channel channelexported.ChannelI,
	consensusState clientexported.ConsensusState,
) error {
	fabProof, err := sanitizeVerificationArgs(cdc, cs, height, prefix, proof, consensusState)
	if err != nil {
		return err
	}

	key := commitment.MakeChannelStateCommitmentEntryKey(prefix, portID, channelID)
	channelEnd, ok := channel.(channeltypes.Channel)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "invalid channel type %T", channel)
	}

	bz, err := cdc.MarshalBinaryBare(&channelEnd)
	if err != nil {
		return err
	}

	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz); err != nil {
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
	cdc codec.Marshaler,
	height uint64,
	prefix commitmentexported.Prefix,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
	consensusState clientexported.ConsensusState,
) error {
	fabProof, err := sanitizeVerificationArgs(cdc, cs, height, prefix, proof, consensusState)
	if err != nil {
		return err
	}

	key := commitment.MakePacketCommitmentEntryKey(prefix, portID, channelID, sequence)
	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, commitmentBytes); err != nil {
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
	cdc codec.Marshaler,
	height uint64,
	prefix commitmentexported.Prefix,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
	consensusState clientexported.ConsensusState,
) error {
	fabProof, err := sanitizeVerificationArgs(cdc, cs, height, prefix, proof, consensusState)
	if err != nil {
		return err
	}

	key := commitment.MakePacketAcknowledgementEntryKey(prefix, portID, channelID, sequence)
	bz := channeltypes.CommitAcknowledgement(acknowledgement)
	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz); err != nil {
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
	cdc codec.Marshaler,
	height uint64,
	prefix commitmentexported.Prefix,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	consensusState clientexported.ConsensusState,
) error {
	fabProof, err := sanitizeVerificationArgs(cdc, cs, height, prefix, proof, consensusState)
	if err != nil {
		return err
	}

	key := commitment.MakePacketAcknowledgementAbsenceEntryKey(prefix, portID, channelID, sequence)
	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, nil); err != nil {
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
	cdc codec.Marshaler,
	height uint64,
	prefix commitmentexported.Prefix,
	proof []byte,
	portID,
	channelID string,
	nextSequenceRecv uint64,
	consensusState clientexported.ConsensusState,
) error {
	fabProof, err := sanitizeVerificationArgs(cdc, cs, height, prefix, proof, consensusState)
	if err != nil {
		return err
	}

	key := commitment.MakeNextSequenceRecvEntryKey(prefix, portID, channelID)
	bz := sdk.Uint64ToBigEndian(nextSequenceRecv)
	if ok, err := VerifyEndorsement(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// sanitizeVerificationArgs perfoms the basic checks on the arguments that are
// shared between the verification functions and returns the unmarshalled
// merkle proof and an error if one occurred.
func sanitizeVerificationArgs(
	cdc codec.Marshaler,
	cs ClientState,
	height uint64,
	prefix commitmentexported.Prefix,
	proofBytes []byte,
	consensusState clientexported.ConsensusState,
) (proof Proof, err error) {
	if cs.GetLatestHeight() < height {
		return Proof{}, sdkerrors.Wrapf(
			sdkerrors.ErrInvalidHeight,
			"client state (%s) height < proof height (%d < %d)", cs.ID, cs.GetLatestHeight(), height,
		)
	}

	if cs.IsFrozen() {
		return Proof{}, clienttypes.ErrClientFrozen
	}

	if prefix == nil {
		return Proof{}, sdkerrors.Wrap(commitmenttypes.ErrInvalidPrefix, "prefix cannot be empty")
	}

	// FIXME comment out this
	// _, ok := prefix.(*Prefix)
	// if !ok {
	// 	return Proof{}, sdkerrors.Wrapf(commitmenttypes.ErrInvalidPrefix, "invalid prefix type %T, expected *Prefix", prefix)
	// }
	_, ok := prefix.(*commitmenttypes.MerklePrefix)
	if !ok {
		return Proof{}, sdkerrors.Wrapf(commitmenttypes.ErrInvalidPrefix, "invalid prefix type %T, expected *MerklePrefix", prefix)
	}

	if proofBytes == nil {
		return Proof{}, sdkerrors.Wrap(commitmenttypes.ErrInvalidProof, "proof cannot be empty")
	}

	if err = cdc.UnmarshalBinaryBare(proofBytes, &proof); err != nil {
		return Proof{}, sdkerrors.Wrap(commitmenttypes.ErrInvalidProof, "failed to unmarshal proof into commitment merkle proof")
	}

	if consensusState == nil {
		return Proof{}, sdkerrors.Wrap(clienttypes.ErrInvalidConsensus, "consensus state cannot be empty")
	}

	_, ok = consensusState.(ConsensusState)
	if !ok {
		return Proof{}, sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "invalid consensus type %T, expected %T", consensusState, ConsensusState{})
	}

	return proof, nil
}

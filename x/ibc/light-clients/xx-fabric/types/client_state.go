package types

import (
	"errors"
	"fmt"
	"strings"

	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
)

const (
	Fabric string = "hyperledgerfabric"
)

var _ exported.ClientState = (*ClientState)(nil)
var _ proto.Message = (*ClientState)(nil)

// NewClientState creates a new ClientState instance
func NewClientState(
	id string,
	chaincodeHeader ChaincodeHeader,
	chaincodeInfo ChaincodeInfo,
	mspInfos MSPInfos,
) *ClientState {
	return &ClientState{
		Id:                  id,
		LastChaincodeHeader: chaincodeHeader,
		LastChaincodeInfo:   chaincodeInfo,
		LastMspInfos:        mspInfos,
	}
}

// GetID returns the loop-back client state identifier.
func (cs ClientState) GetID() string {
	return cs.Id
}

// GetChainID returns an empty string
func (cs ClientState) GetChainID() string {
	return cs.LastChaincodeInfo.GetChainID()
}

// ClientType is localhost.
func (cs ClientState) ClientType() string {
	return Fabric
}

// GetLatestHeight returns the latest height.
func (cs ClientState) GetLatestHeight() exported.Height {
	return clienttypes.NewHeight(0, cs.LastChaincodeHeader.Sequence.Value)
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
	if cs.GetLatestHeight().LTE(clienttypes.NewHeight(0, 0)) {
		return fmt.Errorf("height must be '>=': %d", cs.GetLatestHeight())
	}
	return host.ClientIdentifierValidator(cs.Id)
}

// GetProofSpecs returns the format the client expects for proof verification
// as a string array specifying the proof type for each position in chained proof
func (cs ClientState) GetProofSpecs() []*ics23.ProofSpec {
	return nil
}

// ZeroCustomFields returns solomachine client state with client-specific fields FrozenSequence,
// and AllowUpdateAfterProposal zeroed out
func (cs ClientState) ZeroCustomFields() exported.ClientState {
	return NewClientState(
		cs.Id, cs.LastChaincodeHeader, cs.LastChaincodeInfo, cs.LastMspInfos,
	)
}

// ExportMetadata exports all the consensus metadata in the client store so they can be included in clients genesis
// and imported by a ClientKeeper
func (cs ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	return nil
}

// Status returns the status of the fabric client.
func (cs ClientState) Status(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
) exported.Status {
	// NOTE: Currently, fabric client doesn't support other statuses
	return exported.Active
}

// Initialize will check that initial consensus state is a Fabric consensus state
func (cs ClientState) Initialize(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, consState exported.ConsensusState) error {
	if _, ok := consState.(*ConsensusState); !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "invalid initial consensus state. expected type: %T, got: %T",
			&ConsensusState{}, consState)
	}
	return nil
}

// VerifyUpgradeAndUpdateState returns an error since solomachine client does not support upgrades
func (cs ClientState) VerifyUpgradeAndUpdateState(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore,
	upgradedClient exported.ClientState, upgradedConsState exported.ConsensusState,
	proofUpgradeClient, proofUpgradeConsState []byte,
) (exported.ClientState, exported.ConsensusState, error) {
	return nil, nil, sdkerrors.Wrap(clienttypes.ErrInvalidUpgradeClient, "cannot upgrade fabric client")
}

func (cs ClientState) CheckMisbehaviourAndUpdateState(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	clientStore sdk.KVStore,
	misbehaviour exported.Misbehaviour,
) (exported.ClientState, error) {
	panic("unsupported operation")
}

// VerifyClientState verifies a proof of the client state of the running chain
// stored on the target machine
func (cs ClientState) VerifyClientState(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	prefix exported.Prefix,
	counterpartyClientIdentifier string,
	proof []byte,
	clientState exported.ClientState,
) error {
	fabProof, _, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	if clientState == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidClient, "client state cannot be empty")
	}

	bz, err := clienttypes.MarshalClientState(cdc, clientState)
	if err != nil {
		return err
	}

	configs, err := cs.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}

	key := commitment.MakeClientStateCommitmentEntryKey(prefix, counterpartyClientIdentifier)
	if ok, err := VerifyEndorsedCommitment(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz, configs); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// VerifyClientConsensusState verifies a proof of the consensus state of the
// fabric client stored on the target machine.
func (cs ClientState) VerifyClientConsensusState(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	counterpartyClientIdentifier string,
	consensusHeight exported.Height,
	prefix exported.Prefix,
	proof []byte,
	consensusState exported.ConsensusState,
) error {
	fabProof, _, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	bz, err := clienttypes.MarshalConsensusState(cdc, consensusState)
	if err != nil {
		return err
	}

	configs, err := cs.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}

	key := commitment.MakeConsensusStateCommitmentEntryKey(prefix, counterpartyClientIdentifier, consensusHeight)
	if ok, err := VerifyEndorsedCommitment(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz, configs); err != nil {
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
	cdc codec.BinaryCodec,
	height exported.Height,
	prefix exported.Prefix,
	proof []byte,
	connectionID string,
	connectionEnd exported.ConnectionI,
) error {
	fabProof, _, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	key := commitment.MakeConnectionStateCommitmentEntryKey(prefix, connectionID)

	connection, ok := connectionEnd.(connectiontypes.ConnectionEnd)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "invalid connection type %T", connectionEnd)
	}

	bz, err := cdc.Marshal(&connection)
	if err != nil {
		return err
	}

	configs, err := cs.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}

	if ok, err := VerifyEndorsedCommitment(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz, configs); err != nil {
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
	cdc codec.BinaryCodec,
	height exported.Height,
	prefix exported.Prefix,
	proof []byte,
	portID,
	channelID string,
	channel exported.ChannelI,
) error {
	fabProof, _, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	key := commitment.MakeChannelStateCommitmentEntryKey(prefix, portID, channelID)
	channelEnd, ok := channel.(channeltypes.Channel)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "invalid channel type %T", channel)
	}

	bz, err := cdc.Marshal(&channelEnd)
	if err != nil {
		return err
	}

	configs, err := cs.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}

	if ok, err := VerifyEndorsedCommitment(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz, configs); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// VerifyPacketCommitment verifies a proof of an outgoing packet commitment at
// the specified port, specified channel, and specified sequence.
func (cs ClientState) VerifyPacketCommitment(
	ctx sdk.Context,
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	prefix exported.Prefix,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
) error {
	fabProof, _, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	configs, err := cs.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}

	key := commitment.MakePacketCommitmentEntryKey(prefix, portID, channelID, sequence)
	if ok, err := VerifyEndorsedCommitment(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, commitmentBytes, configs); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// VerifyPacketAcknowledgement verifies a proof of an incoming packet
// acknowledgement at the specified port, specified channel, and specified sequence.
func (cs ClientState) VerifyPacketAcknowledgement(
	ctx sdk.Context,
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	prefix exported.Prefix,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
) error {
	fabProof, _, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	configs, err := cs.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}

	key := commitment.MakePacketAcknowledgementEntryKey(prefix, portID, channelID, sequence)
	bz := channeltypes.CommitAcknowledgement(acknowledgement)
	if ok, err := VerifyEndorsedCommitment(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz, configs); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// VerifyPacketAcknowledgementAbsence verifies a proof of the absence of an
// incoming packet acknowledgement at the specified port, specified channel, and
// specified sequence.
func (cs ClientState) VerifyPacketReceiptAbsence(
	ctx sdk.Context,
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	prefix exported.Prefix,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
) error {
	fabProof, _, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	configs, err := cs.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}

	key := commitment.MakePacketReceiptAbsenceEntryKey(prefix, portID, channelID, sequence)
	if ok, err := VerifyEndorsedCommitment(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, []byte{0}, configs); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// VerifyNextSequenceRecv verifies a proof of the next sequence number to be
// received of the specified channel at the specified port.
func (cs ClientState) VerifyNextSequenceRecv(
	ctx sdk.Context,
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	prefix exported.Prefix,
	proof []byte,
	portID,
	channelID string,
	nextSequenceRecv uint64,
) error {
	fabProof, _, err := produceVerificationArgs(store, cdc, cs, height, prefix, proof)
	if err != nil {
		return err
	}

	configs, err := cs.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}

	key := commitment.MakeNextSequenceRecvEntryKey(prefix, portID, channelID)
	bz := sdk.Uint64ToBigEndian(nextSequenceRecv)
	if ok, err := VerifyEndorsedCommitment(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, bz, configs); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("unexpected value")
	}
	return nil
}

// produceVerificationArgs perfoms the basic checks on the arguments that are
// shared between the verification functions and returns the unmarshalled
// merkle proof, the consensus state and an error if one occurred.
func produceVerificationArgs(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	cs ClientState,
	height exported.Height,
	prefix exported.Prefix,
	proof []byte,
) (commitmentProof CommitmentProof, consensusState *ConsensusState, err error) {
	if cs.GetLatestHeight().LT(height) {
		return CommitmentProof{}, nil, sdkerrors.Wrapf(
			sdkerrors.ErrInvalidHeight,
			"client state height < proof height (%d < %d)", cs.GetLatestHeight(), height,
		)
	}

	if cs.IsFrozen() {
		return CommitmentProof{}, nil, clienttypes.ErrClientFrozen
	}

	if prefix == nil {
		return CommitmentProof{}, nil, sdkerrors.Wrap(commitmenttypes.ErrInvalidPrefix, "prefix cannot be empty")
	}

	_, ok := prefix.(*commitmenttypes.MerklePrefix)
	if !ok {
		return CommitmentProof{}, nil, sdkerrors.Wrapf(commitmenttypes.ErrInvalidPrefix, "invalid prefix type %T, expected *MerklePrefix", prefix)
	}

	if proof == nil {
		return CommitmentProof{}, nil, sdkerrors.Wrap(commitmenttypes.ErrInvalidProof, "proof cannot be empty")
	}

	if err = cdc.Unmarshal(proof, &commitmentProof); err != nil {
		return CommitmentProof{}, nil, sdkerrors.Wrap(commitmenttypes.ErrInvalidProof, "failed to unmarshal proof into commitment merkle proof")
	}

	consensusState, err = GetConsensusState(store, cdc, height)
	if err != nil {
		return CommitmentProof{}, nil, err
	}

	return commitmentProof, consensusState, nil
}

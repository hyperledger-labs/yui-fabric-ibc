package types

import (
	"errors"
	"fmt"
	"strings"

	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	connectiontypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/23-commitment/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/golang/protobuf/proto"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
)

const (
	// Fabric clientexported.ClientType = 100
	ClientTypeFabric string = "fabric"
)

var _ exported.ClientState = ClientState{}

// XXX we need better alias name
type MSPPBConfig = msppb.MSPConfig

func InitializeFromMsg(msg MsgCreateClient) (ClientState, error) {
	return Initialize(msg.ClientID, msg.Header)
}

func Initialize(id string, header Header) (ClientState, error) {
	if header.ChaincodeHeader == nil || header.ChaincodeInfo == nil || header.MSPHeaders == nil {
		return ClientState{}, errors.New("each property of Header must not be empty")
	}
	if err := header.ValidateBasic(); err != nil {
		return ClientState{}, err
	}
	mspInfos, err := generateMSPInfos(header)
	if err != nil {
		return ClientState{}, err
	}
	return NewClientState(id, *header.ChaincodeHeader, *header.ChaincodeInfo, *mspInfos), nil
}

// NewClientState creates a new ClientState instance
func NewClientState(
	id string,
	chaincodeHeader ChaincodeHeader,
	chaincodeInfo ChaincodeInfo,
	mspInfos MSPInfos,
) ClientState {
	return ClientState{
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
	return ClientTypeFabric
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

// VerifyClientConsensusState verifies a proof of the consensus state of the
// Solo Machine client stored on the target machine.
func (cs ClientState) VerifyClientConsensusState(
	store sdk.KVStore,
	cdc codec.BinaryMarshaler,
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

	bz, err := codec.MarshalAny(cdc, consensusState)
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
	cdc codec.BinaryMarshaler,
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

	bz, err := cdc.MarshalBinaryBare(&connection)
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
	cdc codec.BinaryMarshaler,
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

	bz, err := cdc.MarshalBinaryBare(&channelEnd)
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
	store sdk.KVStore,
	cdc codec.Marshaler,
	height exported.Height,
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
	store sdk.KVStore,
	cdc codec.BinaryMarshaler,
	height exported.Height,
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
	store sdk.KVStore,
	cdc codec.BinaryMarshaler,
	height exported.Height,
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

	key := commitment.MakePacketAcknowledgementAbsenceEntryKey(prefix, portID, channelID, sequence)
	if ok, err := VerifyEndorsedCommitment(cs.LastChaincodeInfo.GetFabricChaincodeID(), cs.LastChaincodeInfo.EndorsementPolicy, fabProof, key, []byte{}, configs); err != nil {
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
	height exported.Height,
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
	cdc codec.BinaryMarshaler,
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

	if err = cdc.UnmarshalBinaryBare(proof, &commitmentProof); err != nil {
		return CommitmentProof{}, nil, sdkerrors.Wrap(commitmenttypes.ErrInvalidProof, "failed to unmarshal proof into commitment merkle proof")
	}

	consensusState, err = GetConsensusState(store, cdc, height)
	if err != nil {
		return CommitmentProof{}, nil, err
	}

	return commitmentProof, consensusState, nil
}

func NewMSPInfo(mspID string, config, policy []byte) MSPInfo {
	return MSPInfo{
		MSPID:  mspID,
		Config: config,
		Policy: policy,
	}
}

func (mi MSPInfos) GetMSPPBConfigs() ([]MSPPBConfig, error) {
	configs := []MSPPBConfig{}
	for _, mi := range mi.Infos {
		// freezed MSPInfo is skipped
		if mi.Freezed {
			continue
		}
		if mi.Config == nil {
			// valid MSPInfo should have a config
			return nil, errors.New("a MSPInfo has no config")
		}
		var mspConfig MSPPBConfig
		if err := proto.Unmarshal(mi.Config, &mspConfig); err != nil {
			return nil, err
		}
		configs = append(configs, mspConfig)
	}
	return configs, nil
}

// return true whether the target MSPInfo is freezed or not
func (mi MSPInfos) HasMSPID(mspID string) bool {
	idx := indexOfMSPID(mi, mspID)
	return idx != -1
}

func (mi MSPInfos) IndexOf(mspID string) int {
	idx := indexOfMSPID(mi, mspID)
	return idx
}

func (mi MSPInfos) FindMSPInfo(mspID string) (*MSPInfo, error) {
	idx := indexOfMSPID(mi, mspID)
	if idx < 0 {
		return nil, errors.New("MSPInfo not found")
	}
	return &mi.Infos[idx], nil
}

// assume header.ValidateBasic() == nil
func generateMSPInfos(header Header) (*MSPInfos, error) {
	var infos MSPInfos
	for _, mh := range header.MSPHeaders.Headers {
		if mh.Type != MSPHeaderTypeCreate {
			continue
		}
		infos.Infos = append(infos.Infos, MSPInfo{
			MSPID:   mh.MSPID,
			Config:  mh.Config,
			Policy:  mh.Policy,
			Freezed: false,
		})
	}
	return &infos, nil
}

// return the index of the mspID or -1 if mspID is not present.
func indexOfMSPID(mi MSPInfos, mspID string) int {
	for i, info := range mi.Infos {
		if info.MSPID == mspID {
			return i
		}
	}
	return -1
}

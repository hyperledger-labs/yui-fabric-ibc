package fabric

import (
	"github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/keeper"
	"github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
)

// nolint: golint
type (
	ClientState          = types.ClientState
	Header               = types.Header
	ConsensusState       = types.ConsensusState
	ChaincodeHeader      = types.ChaincodeHeader
	ChaincodeInfo        = types.ChaincodeInfo
	MSPInfo              = types.MSPInfo
	MsgCreateClient      = types.MsgCreateClient
	MsgUpdateClient      = types.MsgUpdateClient
	CommitmentProof      = types.CommitmentProof
	MessageProof         = types.MessageProof
	ChaincodeID          = types.ChaincodeID
	ConsensusStateKeeper = keeper.ConsensusStateKeeper
)

// nolint: golint
var (
	NewHeader               = types.NewHeader
	NewMsgCreateClient      = types.NewMsgCreateClient
	NewMsgUpdateClient      = types.NewMsgUpdateClient
	NewConsensusState       = types.NewConsensusState
	NewChaincodeInfo        = types.NewChaincodeInfo
	NewChaincodeHeader      = types.NewChaincodeHeader
	NewMSPPolicies          = types.NewMSPPolicies
	NewMSPConfigs           = types.NewMSPConfigs
	NewMSPPolicy            = types.NewMSPPolicy
	NewMSPConfig            = types.NewMSPConfig
	NewClientState          = types.NewClientState
	InitializeFromMsg       = types.InitializeFromMsg
	RegisterCodec           = types.RegisterCodec
	NewConsensusStateKeeper = keeper.NewConsensusStateKeeper

	Fabric           = types.Fabric
	ClientTypeFabric = types.ClientTypeFabric
)

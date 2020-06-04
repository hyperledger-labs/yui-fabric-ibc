package fabric

import (
	"github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
)

type (
	ClientState     = types.ClientState
	Header          = types.Header
	ConsensusState  = types.ConsensusState
	MsgCreateClient = types.MsgCreateClient
	Proof           = types.Proof
)

var (
	NewHeader          = types.NewHeader
	NewMsgCreateClient = types.NewMsgCreateClient
	NewMsgUpdateClient = types.NewMsgUpdateClient
	NewConsensusState  = types.NewConsensusState
	NewChaincodeInfo   = types.NewChaincodeInfo
	NewChaincodeHeader = types.NewChaincodeHeader
	NewClientState     = types.NewClientState
	RegisterCodec      = types.RegisterCodec

	Fabric           = types.Fabric
	ClientTypeFabric = types.ClientTypeFabric
)

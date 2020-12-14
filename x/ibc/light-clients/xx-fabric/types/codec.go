package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
)

// RegisterInterfaces registers the tendermint concrete client-related
// implementations and interfaces.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*exported.ClientState)(nil),
		&ClientState{},
	)
	registry.RegisterImplementations(
		(*exported.ConsensusState)(nil),
		&ConsensusState{},
	)
	registry.RegisterImplementations(
		(*exported.Header)(nil),
		&Header{},
	)
	registry.RegisterImplementations(
		(*exported.Header)(nil),
		&Header{},
	)
}

// // ModuleCdc defines the IBC client codec.
// var ModuleCdc *codec.Codec

// // RegisterCodec registers concrete types and interfaces on the given codec.
// func RegisterCodec(cdc *codec.Codec) {
// 	// msg types
// 	cdc.RegisterConcrete(MsgCreateClient{}, "ibc/fabric/MsgCreateClient", nil)
// 	cdc.RegisterConcrete(MsgUpdateClient{}, "ibc/fabric/MsgUpdateClient", nil)

// 	// fabric types
// 	cdc.RegisterConcrete(peer.ProposalResponse{}, "fabric-protos-go/peer/ProposalResponse", nil)

// 	// fabric-ibc types
// 	cdc.RegisterConcrete(ClientState{}, "ibc/fabric/types/ClientState", nil)
// 	cdc.RegisterConcrete(ConsensusState{}, "ibc/fabric/types/ConsensusState", nil)
// 	cdc.RegisterConcrete(ChaincodeHeader{}, "ibc/fabric/types/ChaincodeHeader", nil)
// 	cdc.RegisterConcrete(ChaincodeInfo{}, "ibc/fabric/types/ChaincodeInfo", nil)
// 	cdc.RegisterConcrete(CommitmentProof{}, "ibc/fabric/types/CommitmentProof", nil)
// 	cdc.RegisterConcrete(MessageProof{}, "ibc/fabric/types/MessageProof", nil)
// 	cdc.RegisterConcrete(peer.ChaincodeID{}, "ibc/fabric/types/ChaincodeID", nil)
// }

// func init() {
// 	cdc := codec.New()
// 	RegisterCodec(cdc)
// 	ModuleCdc = cdc
// }

package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/hyperledger/fabric-protos-go/peer"
)

// ModuleCdc defines the IBC client codec.
var ModuleCdc *codec.Codec

func RegisterCodec(cdc *codec.Codec) {
	// msg types
	cdc.RegisterConcrete(MsgCreateClient{}, "ibc/fabric/MsgCreateClient", nil)
	cdc.RegisterConcrete(MsgUpdateClient{}, "ibc/fabric/MsgUpdateClient", nil)

	// fabric types
	cdc.RegisterConcrete(peer.ProposalResponse{}, "fabric-protos-go/peer/ProposalResponse", nil)

	// fabric-ibc types
	cdc.RegisterConcrete(ClientState{}, "ibc/fabric/types/ClientState", nil)
	cdc.RegisterConcrete(ChaincodeHeader{}, "ibc/fabric/types/ChaincodeHeader", nil)
	cdc.RegisterConcrete(ChaincodeInfo{}, "ibc/fabric/types/ChaincodeInfo", nil)
	cdc.RegisterConcrete(Proof{}, "ibc/fabric/types/Proof", nil)
	cdc.RegisterConcrete(peer.ChaincodeID{}, "ibc/fabric/types/ChaincodeID", nil)
}

func init() {
	cdc := codec.New()
	RegisterCodec(cdc)
	ModuleCdc = cdc
}

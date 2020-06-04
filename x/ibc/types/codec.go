package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	connection "github.com/cosmos/cosmos-sdk/x/ibc/03-connection"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	ibctmtypes "github.com/cosmos/cosmos-sdk/x/ibc/07-tendermint/types"
	localhosttypes "github.com/cosmos/cosmos-sdk/x/ibc/09-localhost/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	client "github.com/datachainlab/fabric-ibc/x/ibc/02-client"
	fabric "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric"
)

// RegisterCodec registers the necessary x/ibc interfaces and concrete types
// on the provided Amino codec. These types are used for Amino JSON serialization.
func RegisterCodec(cdc *codec.Codec) {
	client.RegisterCodec(cdc)
	connection.RegisterCodec(cdc)
	channel.RegisterCodec(cdc)
	ibctmtypes.RegisterCodec(cdc)
	localhosttypes.RegisterCodec(cdc)
	fabric.RegisterCodec(cdc)
	commitmenttypes.RegisterCodec(cdc)
}

// RegisterInterfaces registers x/ibc interfaces into protobuf Any.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	connection.RegisterInterfaces(registry)
	channel.RegisterInterfaces(registry)
}

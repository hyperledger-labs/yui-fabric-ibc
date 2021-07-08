package compat

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	connectionkeeper "github.com/cosmos/ibc-go/modules/core/03-connection/keeper"
	ibckeeper "github.com/cosmos/ibc-go/modules/core/keeper"
	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
	fabrickeeper "github.com/hyperledger-labs/yui-fabric-ibc/x/ibc/light-clients/xx-fabric/keeper"
)

// ApplyPatchToIBCKeeper applies patches to ibc keeper
func ApplyPatchToIBCKeeper(k ibckeeper.Keeper, cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace, seqMgr commitment.SequenceManager) *ibckeeper.Keeper {
	clientKeeper := fabrickeeper.NewClientKeeper(k.ClientKeeper, seqMgr)
	k.ConnectionKeeper = connectionkeeper.NewKeeper(cdc, key, paramSpace, clientKeeper)
	return &k
}

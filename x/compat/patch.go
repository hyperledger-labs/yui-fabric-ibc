package compat

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	connectionkeeper "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/keeper"
	ibckeeper "github.com/cosmos/cosmos-sdk/x/ibc/core/keeper"
	"github.com/datachainlab/fabric-ibc/commitment"
	fabrickeeper "github.com/datachainlab/fabric-ibc/x/ibc/light-clients/xx-fabric/keeper"
)

// ApplyPatchToIBCKeeper applies patches to ibc keeper
func ApplyPatchToIBCKeeper(k ibckeeper.Keeper, cdc codec.BinaryMarshaler, key sdk.StoreKey, seqMgr *commitment.SequenceManager) *ibckeeper.Keeper {
	clientKeeper := fabrickeeper.NewClientKeeper(k.ClientKeeper, seqMgr)
	k.ConnectionKeeper = connectionkeeper.NewKeeper(cdc, key, clientKeeper)
	return &k
}

package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	clientkeeper "github.com/cosmos/ibc-go/modules/core/02-client/keeper"
	connectionkeeper "github.com/cosmos/ibc-go/modules/core/03-connection/keeper"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/modules/core/keeper"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode/commitment"
	"github.com/hyperledger-labs/yui-fabric-ibc/light-clients/fabric/types"
)

var _ connectiontypes.ClientKeeper = (*ClientKeeper)(nil)
var _ channeltypes.ClientKeeper = (*ClientKeeper)(nil)

// ClientKeeper override `GetSelfConsensusState` and `ValidateSelfClient` in the keeper of ibc-client
// original method doesn't yet support a consensus state for general client
type ClientKeeper struct {
	clientkeeper.Keeper

	seqMgr commitment.SequenceManager
}

func NewClientKeeper(k clientkeeper.Keeper, seqMgr commitment.SequenceManager) ClientKeeper {
	return ClientKeeper{Keeper: k, seqMgr: seqMgr}
}

// GetSelfConsensusState introspects the (self) past historical info at a given height
// and returns the expected consensus state at that height.
// For now, can only retrieve self consensus states for the current version
func (k ClientKeeper) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, bool) {
	seq, err := k.seqMgr.GetSequenceFromHeight(ctx, height)
	if err != nil {
		return nil, false
	}
	return types.NewConsensusState(seq.Timestamp), true
}

func (k ClientKeeper) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	cs, ok := clientState.(*types.ClientState)
	if !ok {
		return fmt.Errorf("unexpected client state type: %T", clientState)
	}
	_ = cs
	return nil
}

// ApplyPatchToIBCKeeper applies patches to ibc keeper
func ApplyPatchToIBCKeeper(k ibckeeper.Keeper, cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace, seqMgr commitment.SequenceManager) *ibckeeper.Keeper {
	clientKeeper := NewClientKeeper(k.ClientKeeper, seqMgr)
	k.ConnectionKeeper = connectionkeeper.NewKeeper(cdc, key, paramSpace, clientKeeper)
	return &k
}

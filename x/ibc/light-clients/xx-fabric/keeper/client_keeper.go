package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clientkeeper "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/keeper"
	connectiontypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/fabric-ibc/commitment"
	authtypes "github.com/datachainlab/fabric-ibc/x/auth/types"
	"github.com/datachainlab/fabric-ibc/x/ibc/light-clients/xx-fabric/types"
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
	seq, err := k.seqMgr.GetSequence(authtypes.StubFromContext(ctx), height.GetVersionHeight())
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

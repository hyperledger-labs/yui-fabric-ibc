package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
)

// TODO These definitions should be moved into types package

type SelfConsensusStateKeeper interface {
	Get(ctx sdk.Context, height uint64) (exported.ConsensusState, bool)
}

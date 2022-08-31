package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// CheckSubstituteAndUpdateState will try to update the client with the state of the
// substitute if and only if the proposal passes and one of the following conditions are
// satisfied:
func (cs ClientState) CheckSubstituteAndUpdateState(
	ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore,
	substituteClientStore sdk.KVStore, substituteClient exported.ClientState,
) (exported.ClientState, error) {
	panic("not implemented error")
}

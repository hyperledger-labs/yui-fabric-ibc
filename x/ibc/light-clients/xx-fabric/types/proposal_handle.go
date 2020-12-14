package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
)

// CheckProposedHeaderAndUpdateState will try to update the client with the new header if and
// only if the proposal passes and one of the following two conditions is satisfied:
// 		1) AllowUpdateAfterExpiry=true and Expire(ctx.BlockTime) = true
// 		2) AllowUpdateAfterMisbehaviour and IsFrozen() = true
// In case 2) before trying to update the client, the client will be unfrozen by resetting
// the FrozenHeight to the zero Height. If AllowUpdateAfterMisbehaviour is set to true,
// expired clients will also be updated even if AllowUpdateAfterExpiry is set to false.
// Note, that even if the update happens, it may not be successful. The header may fail
// validation checks and an error will be returned in that case.
func (cs ClientState) CheckProposedHeaderAndUpdateState(
	ctx sdk.Context, cdc codec.BinaryMarshaler, clientStore sdk.KVStore,
	header exported.Header,
) (exported.ClientState, exported.ConsensusState, error) {
	panic("unsupported operation")
}

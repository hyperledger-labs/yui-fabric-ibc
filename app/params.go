package app

import sdk "github.com/cosmos/cosmos-sdk/types"

// ParamStore defines the interface the parameter store used by the BaseApp must
// fulfill.
type ParamStore interface {
	Get(ctx sdk.Context, key []byte, ptr interface{})
	Has(ctx sdk.Context, key []byte) bool
	Set(ctx sdk.Context, key []byte, param interface{})
}

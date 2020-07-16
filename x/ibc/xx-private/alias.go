package private

import (
	"github.com/datachainlab/fabric-ibc/x/ibc/xx-private/keeper"
	"github.com/datachainlab/fabric-ibc/x/ibc/xx-private/types"
)

// nolint: golint
type (
	Keeper = keeper.Keeper
)

// nolint: golint
var (
	ModuleName = types.ModuleName

	NewKeeper = keeper.NewKeeper
)

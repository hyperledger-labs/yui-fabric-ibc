package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clientkeeper "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/keeper"
	connectiontypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/fabric-ibc/commitment"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/light-clients/xx-fabric/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

var _ connectiontypes.ClientKeeper = (*ClientKeeper)(nil)
var _ channeltypes.ClientKeeper = (*ClientKeeper)(nil)

// ClientKeeper override `GetSelfConsensusState` in the keeper of ibc-client
// original method doesn't yet support a consensus state for general client
type ClientKeeper struct {
	clientkeeper.Keeper

	stub   shim.ChaincodeStubInterface
	seqMgr *commitment.SequenceManager
}

func NewClientKeeper(k clientkeeper.Keeper, stub shim.ChaincodeStubInterface, seqMgr *commitment.SequenceManager) ClientKeeper {
	return ClientKeeper{Keeper: k}
}

// GetSelfConsensusState introspects the (self) past historical info at a given height
// and returns the expected consensus state at that height.
// For now, can only retrieve self consensus states for the current version
func (k ClientKeeper) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, bool) {
	seq, err := k.seqMgr.GetSequence(k.stub, height.GetVersionHeight())
	if err != nil {
		return nil, false
	}
	return fabrictypes.NewConsensusState(seq.Timestamp), true
}

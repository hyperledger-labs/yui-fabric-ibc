package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	"github.com/datachainlab/fabric-ibc/commitment"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type ConsensusStateKeeper struct {
	stub   shim.ChaincodeStubInterface
	seqMgr *commitment.SequenceManager
}

func NewConsensusStateKeeper(stub shim.ChaincodeStubInterface, seqMgr *commitment.SequenceManager) ConsensusStateKeeper {
	return ConsensusStateKeeper{stub: stub, seqMgr: seqMgr}
}

func (k ConsensusStateKeeper) Get(_ sdk.Context, height uint64) (exported.ConsensusState, bool) {
	seq, err := k.seqMgr.GetSequence(k.stub, height)
	if err != nil {
		return nil, false
	}
	return fabrictypes.NewConsensusState(seq.Timestamp, seq.Value), true
}

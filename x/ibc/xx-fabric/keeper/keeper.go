package keeper

import (
	"fmt"

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
	fmt.Println("ConsensusStateKeeper: ", height)
	seq, err := k.seqMgr.GetSequence(k.stub, height)
	if err != nil {
		return nil, false
	}
	fmt.Println(seq.Timestamp, seq.Value)
	return fabrictypes.NewConsensusState(seq.Timestamp, seq.Value), true
}

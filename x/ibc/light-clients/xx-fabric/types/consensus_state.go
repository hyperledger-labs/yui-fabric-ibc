package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	"github.com/gogo/protobuf/proto"
)

var _ exported.ConsensusState = (*ConsensusState)(nil)
var _ proto.Message = (*ClientState)(nil)

func NewConsensusState(
	timestamp int64,
) *ConsensusState {
	return &ConsensusState{
		Timestamp: timestamp,
	}
}

func (cs ConsensusState) ClientType() string {
	return Fabric
}

func (cs ConsensusState) GetRoot() exported.Root {
	return nil
}

func (cs ConsensusState) GetTimestamp() uint64 {
	return uint64(cs.Timestamp)
}

// ValidateBasic defines a basic validation for the tendermint consensus state.
func (cs ConsensusState) ValidateBasic() error {
	if cs.Timestamp == 0 {
		return sdkerrors.Wrap(clienttypes.ErrInvalidConsensus, "timestamp cannot be zero Unix time")
	}
	return nil
}

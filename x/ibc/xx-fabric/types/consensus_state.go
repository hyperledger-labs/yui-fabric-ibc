package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types"
	commitmentexported "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/exported"
)

var _ clientexported.ConsensusState = (*ConsensusState)(nil)

type ConsensusState struct {
	Timestamp uint64
	Height    uint64
}

func NewConsensusState(
	timestamp uint64, height uint64,
) ConsensusState {
	return ConsensusState{
		Timestamp: timestamp,
		Height:    height,
	}
}

func (cs ConsensusState) ClientType() clientexported.ClientType {
	return Fabric
}

func (cs ConsensusState) GetHeight() uint64 {
	return cs.Height
}

func (cs ConsensusState) GetRoot() commitmentexported.Root {
	panic("fatal error")
}

func (cs ConsensusState) GetTimestamp() uint64 {
	return cs.Timestamp
}

// ValidateBasic defines a basic validation for the tendermint consensus state.
func (cs ConsensusState) ValidateBasic() error {
	if cs.Height == 0 {
		return sdkerrors.Wrap(clienttypes.ErrInvalidConsensus, "height cannot be 0")
	}
	if cs.Timestamp == 0 {
		return sdkerrors.Wrap(clienttypes.ErrInvalidConsensus, "timestamp cannot be zero Unix time")
	}
	return nil
}

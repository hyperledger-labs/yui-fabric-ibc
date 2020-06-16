package fabric

import (
	"errors"
	"time"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types"
	"github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
)

// TODO fix docs
// CheckValidityAndUpdateState checks if the provided header is valid and updates
// the consensus state if appropriate. It returns an error if:
// - the client or header provided are not parseable to tendermint types
// - the header is invalid
// - header height is lower than the latest client height
// - header valset commit verification fails
//
// Tendermint client validity checking uses the bisection algorithm described
// in the [Tendermint spec](https://github.com/tendermint/spec/blob/master/spec/consensus/light-client.md).
func CheckValidityAndUpdateState(
	clientState clientexported.ClientState, header clientexported.Header,
	currentTimestamp time.Time,
) (clientexported.ClientState, clientexported.ConsensusState, error) {
	// update endorsement policiy or chaincode version
	fabClientState, ok := clientState.(ClientState)
	if !ok {
		return nil, nil, sdkerrors.Wrapf(
			clienttypes.ErrInvalidClientType, "client state type %T is not fabric", clientState,
		)
	}

	fabHeader, ok := header.(types.Header)
	if !ok {
		return nil, nil, sdkerrors.Wrap(
			clienttypes.ErrInvalidHeader, "header is not from Fabric",
		)
	}

	if err := checkValidity(fabClientState, fabHeader, currentTimestamp); err != nil {
		return nil, nil, err
	}

	fabClientState, consensusState := update(fabClientState, fabHeader)
	return fabClientState, consensusState, nil
}

// checkValidity checks if the Fabric channel header is valid.
//
// CONTRACT: assumes header.Height > consensusState.Height
func checkValidity(
	clientState types.ClientState, header types.Header, currentTimestamp time.Time,
) error {
	if header.ChaincodeHeader == nil && header.ChaincodeInfo == nil {
		return errors.New("either ChaincodeHeader or ChaincodeInfo must be non-nil value")
	}

	if header.ChaincodeHeader != nil {
		// assert header timestamp is past latest clientstate timestamp
		if header.ChaincodeHeader.Sequence.Timestamp <= clientState.GetLatestTimestamp() {
			return sdkerrors.Wrapf(
				clienttypes.ErrInvalidHeader,
				"header blocktime ≤ latest client state block time (%v ≤ %v)",
				header.ChaincodeHeader.Sequence.Timestamp, clientState.GetLatestTimestamp(),
			)
		}

		if header.ChaincodeHeader.Sequence.Value != clientState.GetLatestHeight()+1 {
			return sdkerrors.Wrapf(
				clienttypes.ErrInvalidHeader,
				"header sequence != expected client state sequence (%d != %d)", header.ChaincodeHeader.Sequence, clientState.GetLatestHeight()+1,
			)
		}

		if err := types.VerifyChaincodeHeader(clientState, *header.ChaincodeHeader); err != nil {
			return sdkerrors.Wrap(
				clienttypes.ErrInvalidHeader,
				err.Error(),
			)
		}
	}

	if header.ChaincodeInfo != nil {
		if len(header.ChaincodeInfo.EndorsementPolicy) == 0 {
			return sdkerrors.Wrapf(
				clienttypes.ErrInvalidHeader,
				"ChaincodeInfo.EndorsementPolicy must be non-empty value",
			)
		}

		if err := types.VerifyChaincodeInfo(clientState, *header.ChaincodeInfo); err != nil {
			return sdkerrors.Wrap(
				clienttypes.ErrInvalidHeader,
				err.Error(),
			)
		}
	}

	return nil
}

func update(clientState ClientState, header Header) (ClientState, *ConsensusState) {
	if header.ChaincodeInfo != nil {
		clientState.LastChaincodeInfo = *header.ChaincodeInfo
		return clientState, nil
	}

	if header.ChaincodeHeader != nil {
		clientState.LastChaincodeHeader = *header.ChaincodeHeader
		consensusState := NewConsensusState(
			header.ChaincodeHeader.Sequence.Timestamp,
			header.GetHeight(),
		)
		return clientState, &consensusState
	}

	panic("either ChaincodeHeader or ChaincodeInfo must be non-nil value")
}

package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// CheckHeaderAndUpdateState checks if the provided header is valid and updates
// the consensus state if appropriate. It returns an error if:
// - the header provided is not parseable to a solo machine header
// - the header sequence does not match the current sequence
// - the header timestamp is less than the consensus state timestamp
// - the currently registered public key did not provide the update signature
func (cs ClientState) CheckHeaderAndUpdateState(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore,
	header exported.Header,
) (exported.ClientState, exported.ConsensusState, error) {
	fabHeader, ok := header.(*Header)
	if !ok {
		return nil, nil, sdkerrors.Wrap(
			clienttypes.ErrInvalidHeader, "header is not from Fabric",
		)
	}

	if err := checkValidity(cs, *fabHeader); err != nil {
		return nil, nil, err
	}

	newClientState, consensusState := update(cs, *fabHeader)
	return newClientState, consensusState, nil
}

// checkValidity checks if the Fabric channel header is valid.
//
// CONTRACT: assumes header.Height > consensusState.Height
func checkValidity(
	clientState ClientState, header Header,
) error {
	if err := header.ValidateBasic(); err != nil {
		return sdkerrors.Wrapf(
			clienttypes.ErrInvalidHeader,
			err.Error(),
		)
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

		h := clienttypes.NewHeight(0, header.ChaincodeHeader.Sequence.Value)
		// TODO use Increment function instead of creating a new instance
		if lh := clientState.GetLatestHeight(); !h.EQ(clienttypes.NewHeight(lh.GetRevisionNumber(), lh.GetRevisionHeight()+1)) {
			return sdkerrors.Wrapf(
				clienttypes.ErrInvalidHeader,
				"header sequence != expected client state sequence (%d != %d)", header.ChaincodeHeader.Sequence, lh.GetRevisionHeight()+1,
			)
		}

		if err := VerifyChaincodeHeader(clientState, *header.ChaincodeHeader); err != nil {
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

		if err := VerifyChaincodeInfo(clientState, *header.ChaincodeInfo); err != nil {
			return sdkerrors.Wrap(
				clienttypes.ErrInvalidHeader,
				err.Error(),
			)
		}
	}

	if header.MSPHeaders != nil {
		if err := VerifyMSPHeaders(clientState, *header.MSPHeaders); err != nil {
			return sdkerrors.Wrap(
				clienttypes.ErrInvalidHeader,
				err.Error(),
			)
		}
	}

	return nil
}

// Update

func update(clientState ClientState, header Header) (*ClientState, *ConsensusState) {
	var consensusState *ConsensusState

	if err := header.ValidateBasic(); err != nil {
		panic(err.Error())
	}

	if header.ChaincodeInfo != nil {
		clientState.LastChaincodeInfo = *header.ChaincodeInfo
	}

	if header.ChaincodeHeader != nil {
		clientState.LastChaincodeHeader = *header.ChaincodeHeader
		consensusState = NewConsensusState(
			header.ChaincodeHeader.Sequence.Timestamp,
		)
	}

	if header.MSPHeaders != nil {
		clientState = updateMSPInfos(clientState, *header.MSPHeaders)
	}

	return &clientState, consensusState
}

// assume MSPHeaders are sorted as validated by ValidateBasic() in advance
func updateMSPInfos(clientState ClientState, mhs MSPHeaders) ClientState {
	for _, mh := range mhs.Headers {
		clientState = updateMSPInfo(clientState, mh)
	}
	return clientState
}

func updateMSPInfo(clientState ClientState, mh MSPHeader) ClientState {
	switch mh.Type {
	case MSPHeaderTypeCreate:
		return createMSPInfo(clientState, mh)
	case MSPHeaderTypeUpdatePolicy:
		return updateMSPPolicy(clientState, mh)
	case MSPHeaderTypeUpdateConfig:
		return updateMSPConfig(clientState, mh)
	case MSPHeaderTypeFreeze:
		return freezeMSPInfo(clientState, mh)
	default:
		panic("invalid MSPHeaderType")
	}
}

func createMSPInfo(clientState ClientState, mh MSPHeader) ClientState {
	var newInfos []MSPInfo
	appended := false
	for _, info := range clientState.LastMspInfos.Infos {
		if !appended && CompareMSPID(mh.MSPID, info.MSPID) < 0 {
			newInfos = append(newInfos, NewMSPInfo(mh.MSPID, mh.Config, mh.Policy), info)
			appended = true
			continue
		}
		newInfos = append(newInfos, info)
	}
	if !appended {
		newInfos = append(newInfos, NewMSPInfo(mh.MSPID, mh.Config, mh.Policy))
	}

	clientState.LastMspInfos.Infos = newInfos
	return clientState
}

func updateMSPConfig(clientState ClientState, mh MSPHeader) ClientState {
	// assume idx != -1
	idx := clientState.LastMspInfos.IndexOf(mh.MSPID)
	clientState.LastMspInfos.Infos[idx].Config = mh.Config
	return clientState
}

func updateMSPPolicy(clientState ClientState, mh MSPHeader) ClientState {
	// assume idx != -1
	idx := clientState.LastMspInfos.IndexOf(mh.MSPID)
	clientState.LastMspInfos.Infos[idx].Policy = mh.Policy
	return clientState
}

func freezeMSPInfo(clientState ClientState, mh MSPHeader) ClientState {
	// assume idx != -1
	idx := clientState.LastMspInfos.IndexOf(mh.MSPID)
	clientState.LastMspInfos.Infos[idx].Freezed = true
	return clientState
}

package fabric

import (
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

	if header.MSPHeaders != nil {
		if err := types.VerifyMSPHeaders(clientState, *header.MSPHeaders); err != nil {
			return sdkerrors.Wrap(
				clienttypes.ErrInvalidHeader,
				err.Error(),
			)
		}
	}

	return nil
}

func update(clientState ClientState, header Header) (ClientState, *ConsensusState) {
	var consensusState *ConsensusState

	if err := header.ValidateBasic(); err != nil {
		panic(err.Error())
	}

	if header.ChaincodeInfo != nil {
		clientState.LastChaincodeInfo = *header.ChaincodeInfo
	}

	if header.ChaincodeHeader != nil {
		clientState.LastChaincodeHeader = *header.ChaincodeHeader
		cs := NewConsensusState(
			header.ChaincodeHeader.Sequence.Timestamp,
			header.GetHeight(),
		)
		consensusState = &cs
	}

	if header.MSPHeaders != nil {
		clientState = updateMSPInfos(clientState, *header.MSPHeaders)
	}

	return clientState, consensusState
}

// assume MSPHeaders are sorted as validated by ValidateBasic() in advance
func updateMSPInfos(clientState ClientState, mhs types.MSPHeaders) ClientState {
	for _, mh := range mhs.Headers {
		clientState = updateMSPInfo(clientState, mh)
	}
	return clientState
}

func updateMSPInfo(clientState ClientState, mh types.MSPHeader) ClientState {
	switch mh.Type {
	case types.MSPHeaderTypeCreate:
		return createMSPInfo(clientState, mh)
	case types.MSPHeaderTypeUpdatePolicy:
		return updateMSPPolicy(clientState, mh)
	case types.MSPHeaderTypeUpdateConfig:
		return updateMSPConfig(clientState, mh)
	case types.MSPHeaderTypeFreeze:
		return freezeMSPInfo(clientState, mh)
	default:
		panic("invalid MSPHeaderType")
	}
}

func createMSPInfo(clientState ClientState, mh types.MSPHeader) ClientState {
	var newInfos []MSPInfo
	appended := false
	for _, info := range clientState.LastMSPInfos.Infos {
		if !appended && mh.MSPID < info.MSPID {
			newInfos = append(newInfos, types.NewMSPInfo(mh.MSPID, mh.Config, mh.Policy), info)
			appended = true
			continue
		}
		newInfos = append(newInfos, info)
	}
	if !appended {
		newInfos = append(newInfos, types.NewMSPInfo(mh.MSPID, mh.Config, mh.Policy))
	}

	clientState.LastMSPInfos.Infos = newInfos
	return clientState
}

func updateMSPConfig(clientState ClientState, mh types.MSPHeader) ClientState {
	// assume idx != -1
	idx := clientState.LastMSPInfos.IndexOf(mh.MSPID)
	clientState.LastMSPInfos.Infos[idx].Config = mh.Config
	return clientState
}

func updateMSPPolicy(clientState ClientState, mh types.MSPHeader) ClientState {
	// assume idx != -1
	idx := clientState.LastMSPInfos.IndexOf(mh.MSPID)
	clientState.LastMSPInfos.Infos[idx].Policy = mh.Policy
	return clientState
}

func freezeMSPInfo(clientState ClientState, mh types.MSPHeader) ClientState {
	// assume idx != -1
	idx := clientState.LastMSPInfos.IndexOf(mh.MSPID)
	clientState.LastMSPInfos.Infos[idx].Freezed = true
	return clientState
}

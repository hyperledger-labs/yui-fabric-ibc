package fabric

import (
	"strings"
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

	if header.MSPPolicies != nil {
		if err := types.VerifyMSPPolicies(clientState, *header.MSPPolicies); err != nil {
			return sdkerrors.Wrap(
				clienttypes.ErrInvalidHeader,
				err.Error(),
			)
		}
	}

	if header.MSPConfigs != nil {
		if err := types.VerifyMSPConfigs(clientState, *header.MSPConfigs); err != nil {
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

	if header.MSPPolicies != nil {
		clientState = updateMSPPolicies(clientState, *header.MSPPolicies)
	}

	if header.MSPConfigs != nil {
		clientState = updateMSPConfigs(clientState, *header.MSPConfigs)
	}

	return clientState, consensusState
}

// this should be called before updateMSPPolicies for the same ClientState.
// if clientState does not have MSPIDs which mspPolicies have, new MSPInfos are inserted in the sorted positions.
// assume mspPolicies are sorted as validated by ValidateBasic()
func updateMSPPolicies(clientState ClientState, mspPolicies types.MSPPolicies) ClientState {
	var newInfos []MSPInfo
	lenP := len(mspPolicies.Policies)
	lenI := len(clientState.LastMSPInfos.Infos)
	pi := 0
	ii := 0
	for pi < lenP || ii < lenI {
		if pi == lenP {
			info := clientState.LastMSPInfos.Infos[ii]
			newInfos = append(newInfos, info)
			ii++
			continue
		}
		if ii == lenI {
			policy := mspPolicies.Policies[pi]
			newInfos = append(newInfos, types.NewMSPInfo(policy.MSPID, nil, policy.Policy))
			pi++
			continue
		}

		policy := mspPolicies.Policies[pi]
		info := clientState.LastMSPInfos.Infos[ii]
		comp := strings.Compare(policy.MSPID, info.MSPID)
		if comp == -1 {
			newInfos = append(newInfos, types.NewMSPInfo(policy.MSPID, nil, policy.Policy))
			pi++
		} else if comp == 0 {
			info.Policy = policy.Policy
			newInfos = append(newInfos, info)
			pi++
			ii++
		} else {
			newInfos = append(newInfos, info)
			ii++
		}
	}
	clientState.LastMSPInfos = types.MSPInfos{Infos: newInfos}
	return clientState
}

// this should be called after updateMSPPolicies for the same ClientState.
// assume clientState has a MSPInfo for each MSPID of mspConfigs, as verified by VerifyMSPConfig in advance
func updateMSPConfigs(clientState ClientState, mspConfigs types.MSPConfigs) ClientState {
	var newInfos []MSPInfo
	lenC := len(mspConfigs.Configs)
	lenI := len(clientState.LastMSPInfos.Infos)
	ci := 0
	ii := 0
	for ci < lenC && ii < lenI {
		config := mspConfigs.Configs[ci]
		info := clientState.LastMSPInfos.Infos[ii]
		comp := strings.Compare(config.MSPID, info.MSPID)
		if comp == 0 {
			info.Config = config.Config
			newInfos = append(newInfos, info)
			ci++
			ii++
		} else if comp == 1 {
			newInfos = append(newInfos, info)
			ii++
		} else {
			// after verifying, this never happens
			panic("mspConfigs must be verified")
		}
	}
	for ii < lenI {
		info := clientState.LastMSPInfos.Infos[ii]
		newInfos = append(newInfos, info)
		ii++
	}
	clientState.LastMSPInfos = types.MSPInfos{Infos: newInfos}
	return clientState
}

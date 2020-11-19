package types

import (
	"errors"
	"fmt"

	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
)

var _ clientexported.Header = (*Header)(nil)

func NewHeader(cheader ChaincodeHeader, cinfo ChaincodeInfo, mPolicies MSPPolicies, mConfigs MSPConfigs) Header {
	return Header{ChaincodeHeader: &cheader, ChaincodeInfo: &cinfo, MSPPolicies: &mPolicies, MSPConfigs: &mConfigs}
}

func (h Header) GetHeight() uint64 {
	return h.ChaincodeHeader.Sequence.Value
}

func (h Header) ClientType() clientexported.ClientType {
	return Fabric
}

func (h Header) ValidateBasic() error {
	if h.ChaincodeHeader == nil && h.ChaincodeInfo == nil && h.MSPConfigs == nil && h.MSPPolicies == nil {
		return errors.New("either ChaincodeHeader, ChaincodeInfo, MSPConfigs or MSPPolicies must be non-nil value")
	}
	if err := h.ChaincodeHeader.ValidateBasic(); err != nil {
		return err
	}
	if err := h.ChaincodeInfo.ValidateBasic(); err != nil {
		return err
	}
	if err := h.MSPPolicies.ValidateBasic(); err != nil {
		return err
	}
	if err := h.MSPConfigs.ValidateBasic(); err != nil {
		return err
	}
	return nil
}

// check whether MSPPolicies and MSPConfigs have IDs in same order.
// each pair must have the same ActionType
func (h Header) TargetsSameMSPs() bool {
	if h.MSPPolicies == nil || h.MSPPolicies.Policies == nil || h.MSPConfigs == nil || h.MSPConfigs.Configs == nil {
		return false
	}
	if len(h.MSPPolicies.Policies) != len(h.MSPConfigs.Configs) {
		return false
	}
	for pi, policy := range h.MSPPolicies.Policies {
		config := h.MSPConfigs.Configs[pi]
		if policy.MSPID != config.MSPID {
			return false
		}
		if policy.Type != config.Type {
			return false
		}
	}
	return true
}

func NewChaincodeHeader(seq uint64, timestamp int64, proof CommitmentProof) ChaincodeHeader {
	return ChaincodeHeader{
		Sequence: commitment.NewSequence(seq, timestamp),
		Proof:    proof,
	}
}

func (h ChaincodeHeader) ValidateBasic() error {
	return nil
}

func NewChaincodeInfo(chanID string, ccID ChaincodeID, endorsementPolicy, ibcPolicy []byte, proof *MessageProof) ChaincodeInfo {
	return ChaincodeInfo{
		ChannelId:         chanID,
		ChaincodeId:       ccID,
		EndorsementPolicy: endorsementPolicy,
		IbcPolicy:         ibcPolicy,
		Proof:             proof,
	}
}

func (ci ChaincodeInfo) ValidateBasic() error {
	return nil
}

func (ci ChaincodeInfo) GetChainID() string {
	return fmt.Sprintf("%v/%v", ci.ChannelId, ci.ChaincodeId.String())
}

func (ci ChaincodeInfo) GetFabricChaincodeID() peer.ChaincodeID {
	return peer.ChaincodeID{
		Path:    ci.ChaincodeId.Path,
		Name:    ci.ChaincodeId.Name,
		Version: ci.ChaincodeId.Version,
	}
}

func (ci ChaincodeInfo) GetSignBytes() []byte {
	ci.Proof = nil
	bz, err := proto.Marshal(&ci)
	if err != nil {
		panic(err)
	}
	return bz
}

func NewMSPConfig(actionType ActionType, mspID string, config []byte, proof *MessageProof) MSPConfig {
	return MSPConfig{
		Type:   actionType,
		MSPID:  mspID,
		Config: config,
		Proof:  proof,
	}
}

func (mc MSPConfig) ValidateBasic() error {
	if mc.MSPID == "" {
		return errors.New("a MSPID is empty")
	}
	if mc.Config == nil {
		return errors.New("a config is empty")
	}
	if mc.Proof == nil {
		return errors.New("a proof is empty")
	}
	return nil
}

func (mc MSPConfig) GetSignBytes() []byte {
	mc.Proof = nil
	bz, err := proto.Marshal(&mc)
	if err != nil {
		panic(err)
	}
	return bz
}

func NewMSPConfigs(configs []MSPConfig) MSPConfigs {
	return MSPConfigs{
		Configs: configs,
	}
}

func (mcs MSPConfigs) ValidateBasic() error {
	// check duplication and sorting
	m := map[string]bool{}
	prevID := ""

	for _, mc := range mcs.Configs {
		if err := mc.ValidateBasic(); err != nil {
			return err
		}
		if m[mc.MSPID] {
			return errors.New("some configs are duplicated")
		}
		if prevID >= mc.MSPID {
			return errors.New("MSPID must be sorted by ascending order")
		}
		m[mc.MSPID] = true
		prevID = mc.MSPID
	}
	return nil
}

func NewMSPPolicy(actionType ActionType, mspID string, policy []byte, proof *MessageProof) MSPPolicy {
	return MSPPolicy{
		Type:   actionType,
		MSPID:  mspID,
		Policy: policy,
		Proof:  proof,
	}
}

func (mp MSPPolicy) ValidateBasic() error {
	if mp.MSPID == "" {
		return errors.New("an MSPID is empty")
	}
	if mp.Policy == nil {
		return errors.New("a policy is empty")
	}
	if mp.Proof == nil {
		return errors.New("a proof is empty")
	}
	return nil
}

func (mp MSPPolicy) GetSignBytes() []byte {
	mp.Proof = nil
	bz, err := proto.Marshal(&mp)
	if err != nil {
		panic(err)
	}
	return bz
}

func NewMSPPolicies(policies []MSPPolicy) MSPPolicies {
	return MSPPolicies{
		Policies: policies,
	}
}

func (mps MSPPolicies) ValidateBasic() error {
	m := map[string]bool{}
	prevID := ""
	for _, mp := range mps.Policies {
		if err := mp.ValidateBasic(); err != nil {
			return err
		}
		if m[mp.MSPID] {
			return errors.New("some configs are duplicated")
		}
		if prevID >= mp.MSPID {
			return errors.New("MSPID must be sorted by ascending order")
		}
		m[mp.MSPID] = true
		prevID = mp.MSPID
	}
	return nil

}

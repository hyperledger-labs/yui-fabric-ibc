package types

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
)

// VerifyChaincodeHeader verifies ChaincodeHeader with last Endorsement Policy
func VerifyChaincodeHeader(clientState ClientState, h ChaincodeHeader) error {
	configs, err := clientState.LastMSPInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}

	lastci := clientState.LastChaincodeInfo
	ok, err := VerifyEndorsedCommitment(clientState.LastChaincodeInfo.GetFabricChaincodeID(), lastci.EndorsementPolicy, h.Proof, MakeSequenceCommitmentEntryKey(h.Sequence.Value), h.Sequence.Bytes(), configs)
	if err != nil {
		return err
	} else if !ok {
		return errors.New("failed to verify a endorsed commitment")
	}
	return nil
}

// VerifyChaincodeInfo verifies ChaincodeInfo with last IBC Policy
func VerifyChaincodeInfo(clientState ClientState, info ChaincodeInfo) error {
	if info.Proof == nil {
		return errors.New("a proof is empty")
	}
	configs, err := clientState.LastMSPInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}
	return VerifyEndorsedMessage(clientState.LastChaincodeInfo.IbcPolicy, *info.Proof, info.GetSignBytes(), configs)
}

func VerifyMSPConfigs(clientState ClientState, configs MSPConfigs) error {
	for _, config := range configs.Configs {
		if err := VerifyMSPConfig(clientState, config); err != nil {
			return err
		}
	}
	return nil
}

// verify whether MSPConfig suits the last MSPPolicy
// MSPPolicy must be created in advance
func VerifyMSPConfig(clientState ClientState, config MSPConfig) error {
	if config.Proof == nil {
		return errors.New("a proof is empty")
	}
	policy, err := clientState.LastMSPInfos.FindMSPPolicy(config.MSPID)
	if err != nil {
		return err
	}
	switch config.Type {
	case TypeCreate:
		return verifyMSPConfigCreate(clientState, policy, config)
	case TypeUpdate:
		return verifyMSPConfigUpdate(clientState, policy, config)
	default:
		return errors.New("invalid type")
	}
}

func verifyMSPConfigCreate(clientState ClientState, policy []byte, config MSPConfig) error {
	if _, err := clientState.LastMSPInfos.FindMSPConfig(config.MSPID); err == nil {
		return errors.New("MSPConfig has already been created")
	}
	lastConfigs, err := clientState.LastMSPInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}
	return VerifyEndorsedMessage(policy, *config.Proof, config.GetSignBytes(), lastConfigs)
}

func verifyMSPConfigUpdate(clientState ClientState, policy []byte, config MSPConfig) error {
	if _, err := clientState.LastMSPInfos.FindMSPConfig(config.MSPID); err != nil {
		return errors.New("MSPConfig has not been created")
	}
	lastConfigs, err := clientState.LastMSPInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}
	return VerifyEndorsedMessage(policy, *config.Proof, config.GetSignBytes(), lastConfigs)
}

func VerifyMSPPolicies(clientState ClientState, policies MSPPolicies) error {
	for _, policy := range policies.Policies {
		if err := VerifyMSPPolicy(clientState, policy); err != nil {
			return err
		}
	}
	return nil
}

// verifies whether MSPPolicy suits the last IBCPolicy
func VerifyMSPPolicy(clientState ClientState, policy MSPPolicy) error {
	if policy.Proof == nil {
		return errors.New("a proof is empty")
	}

	switch policy.Type {
	case TypeCreate:
		return verifyMSPPolicyCreate(clientState, policy)
	case TypeUpdate:
		return verifyMSPPolicyUpdate(clientState, policy)
	default:
		return errors.New("invalid type")
	}
}

func verifyMSPPolicyCreate(clientState ClientState, policy MSPPolicy) error {
	if _, err := clientState.LastMSPInfos.FindMSPPolicy(policy.MSPID); err == nil {
		return errors.New("MSPPolicy has already been created")
	}
	configs, err := clientState.LastMSPInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}
	return VerifyEndorsedMessage(clientState.LastChaincodeInfo.IbcPolicy, *policy.Proof, policy.GetSignBytes(), configs)
}

func verifyMSPPolicyUpdate(clientState ClientState, policy MSPPolicy) error {
	if _, err := clientState.LastMSPInfos.FindMSPPolicy(policy.MSPID); err != nil {
		return errors.New("MSPPolicy has not been created")
	}
	configs, err := clientState.LastMSPInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}
	return VerifyEndorsedMessage(clientState.LastChaincodeInfo.IbcPolicy, *policy.Proof, policy.GetSignBytes(), configs)
}

// VerifyEndorsedMessage verifies a value with given policy
func VerifyEndorsedMessage(policyBytes []byte, proof MessageProof, value []byte, configs []MSPPBConfig) error {
	policy, err := getPolicyEvaluator(policyBytes, configs)
	if err != nil {
		return err
	}
	sigs := makeSignedDataListWithMessageProof(proof, value)
	if err := policy.EvaluateSignedData(sigs); err != nil {
		return err
	}
	return nil
}

// VerifyEndorsedCommitment verifies a key-value entry with a policy
func VerifyEndorsedCommitment(ccID peer.ChaincodeID, policyBytes []byte, proof CommitmentProof, key string, value []byte, configs []MSPPBConfig) (bool, error) {
	policy, err := getPolicyEvaluator(policyBytes, configs)
	if err != nil {
		return false, err
	}
	if err := policy.EvaluateSignedData(proof.ToSignedData()); err != nil {
		return false, err
	}

	id, rwset, err := getTxReadWriteSetFromProposalResponsePayload(proof.Proposal)
	if err != nil {
		return false, err
	}
	if !equalChaincodeID(ccID, *id) {
		return false, fmt.Errorf("got unexpected chaincodeID: expected=%v actual=%v", ccID, *id)
	}
	return ensureWriteSetIncludesCommitment(rwset.NsRwSets, proof.NsIndex, proof.WriteSetIndex, key, value)
}

func getPolicyEvaluator(policyBytes []byte, configs []MSPPBConfig) (policies.Policy, error) {
	var ap peer.ApplicationPolicy
	if err := proto.Unmarshal(policyBytes, &ap); err != nil {
		return nil, err
	}
	sigp := ap.Type.(*peer.ApplicationPolicy_SignaturePolicy)

	mgr, err := setupVerifyingMSPManager(configs)
	if err != nil {
		return nil, err
	}
	pp := cauthdsl.EnvelopeBasedPolicyProvider{Deserializer: mgr}
	return pp.NewPolicy(sigp.SignaturePolicy)
}

func ensureWriteSetIncludesCommitment(set []*rwsetutil.NsRwSet, nsIdx, rwsIdx uint32, targetKey string, expectValue []byte) (bool, error) {
	rws := set[nsIdx].KvRwSet
	if len(rws.Writes) <= int(rwsIdx) {
		return false, fmt.Errorf("not found index '%v'(length=%v)", int(rwsIdx), len(rws.Writes))
	}

	w := rws.Writes[rwsIdx]
	if targetKey != w.Key {
		return false, fmt.Errorf("unexpected key '%v'(expected=%v)", w.Key, targetKey)
	}
	if bytes.Equal(w.Value, expectValue) {
		return true, nil
	} else {
		return false, nil
	}
}

func equalChaincodeID(a, b peer.ChaincodeID) bool {
	if a.Name == b.Name && a.Path == b.Path && a.Version == b.Version {
		return true
	} else {
		return false
	}
}

func getTxReadWriteSetFromProposalResponsePayload(proposal []byte) (*peer.ChaincodeID, *rwsetutil.TxRwSet, error) {
	var payload peer.ProposalResponsePayload
	if err := proto.Unmarshal(proposal, &payload); err != nil {
		return nil, nil, err
	}
	var cact peer.ChaincodeAction
	if err := proto.Unmarshal(payload.Extension, &cact); err != nil {
		return nil, nil, err
	}
	txRWSet := &rwsetutil.TxRwSet{}
	if err := txRWSet.FromProtoBytes(cact.Results); err != nil {
		return nil, nil, err
	}
	return cact.GetChaincodeId(), txRWSet, nil
}

func setupVerifyingMSPManager(mspConfigs []MSPPBConfig) (msp.MSPManager, error) {
	msps := []msp.MSP{}
	// if in local, this would depend `peer.localMspType` config
	for _, conf := range mspConfigs {
		m, err := setupVerifyingMSP(conf)
		if err != nil {
			return nil, err
		}
		msps = append(msps, m)
	}
	mgr := msp.NewMSPManager()
	err := mgr.Setup(msps)
	if err != nil {
		return nil, err
	}
	return mgr, nil
}

func setupVerifyingMSP(mspConf MSPPBConfig) (msp.MSP, error) {
	bccspConfig := factory.GetDefaultOpts()
	cryptoProvider, err := (&factory.SWFactory{}).Get(bccspConfig)
	if err != nil {
		return nil, err
	}
	opts := msp.Options[msp.ProviderTypeToString(msp.FABRIC)]
	m, err := msp.New(opts, cryptoProvider)
	if err != nil {
		return nil, err
	}
	if err := m.Setup(&mspConf); err != nil {
		return nil, err
	}
	return m, nil
}

func makeSignedDataListWithMessageProof(proof MessageProof, value []byte) []*protoutil.SignedData {
	var sigSet []*protoutil.SignedData
	for i := 0; i < len(proof.Signatures); i++ {
		sigSet = append(
			sigSet,
			&protoutil.SignedData{
				Data:      value,
				Identity:  proof.Identities[i],
				Signature: proof.Signatures[i],
			},
		)
	}
	return sigSet
}

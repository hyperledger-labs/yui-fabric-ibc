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
	configs, err := clientState.LastMspInfos.GetMSPPBConfigs()
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
	configs, err := clientState.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}
	return VerifyEndorsedMessage(clientState.LastChaincodeInfo.IbcPolicy, *info.Proof, info.GetSignBytes(), configs)
}

func VerifyMSPHeaders(clientState ClientState, mhs MSPHeaders) error {
	for _, mh := range mhs.Headers {
		if err := VerifyMSPHeader(clientState, mh); err != nil {
			return err
		}
	}
	return nil
}

// verify whether MSPHeader suits the last MSPInfos.
// assume MSPHeader.ValidateBasic() is nil
func VerifyMSPHeader(clientState ClientState, mh MSPHeader) error {
	switch mh.Type {
	case MSPHeaderTypeCreate:
		return verifyMSPCreate(clientState, mh)
	case MSPHeaderTypeUpdatePolicy:
		return verifyMSPUpdatePolicy(clientState, mh)
	case MSPHeaderTypeUpdateConfig:
		return verifyMSPUpdateConfig(clientState, mh)
	case MSPHeaderTypeFreeze:
		return verifyMSPFreeze(clientState, mh)
	default:
		return errors.New("invalid type")
	}
}

func verifyMSPCreate(clientState ClientState, mh MSPHeader) error {
	if mh.Proof == nil {
		return errors.New("proof is nil")
	}

	if clientState.LastMspInfos.HasMSPID(mh.MSPID) {
		return errors.New("MSPInfo has already been created")
	}

	policy := clientState.LastChaincodeInfo.IbcPolicy
	lastConfigs, err := clientState.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}
	return VerifyEndorsedMessage(policy, *mh.Proof, mh.GetSignBytes(), lastConfigs)
}

func verifyMSPUpdatePolicy(clientState ClientState, mh MSPHeader) error {
	if mh.Proof == nil {
		return errors.New("proof is nil")
	}

	mi, err := clientState.LastMspInfos.FindMSPInfo(mh.MSPID)
	if err != nil {
		return err
	}
	if mi.Freezed {
		return errors.New("MSPInfo is freezed")
	}

	policy := clientState.LastChaincodeInfo.IbcPolicy
	lastConfigs, err := clientState.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}
	return VerifyEndorsedMessage(policy, *mh.Proof, mh.GetSignBytes(), lastConfigs)
}

func verifyMSPUpdateConfig(clientState ClientState, mh MSPHeader) error {
	if mh.Proof == nil {
		return errors.New("proof is nil")
	}

	mi, err := clientState.LastMspInfos.FindMSPInfo(mh.MSPID)
	if err != nil {
		return err
	}
	if mi.Freezed {
		return errors.New("MSPInfo is freezed")
	}

	lastConfigs, err := clientState.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}
	return VerifyEndorsedMessage(mi.Policy, *mh.Proof, mh.GetSignBytes(), lastConfigs)
}

func verifyMSPFreeze(clientState ClientState, mh MSPHeader) error {
	if mh.Proof == nil {
		return errors.New("proof is nil")
	}

	mi, err := clientState.LastMspInfos.FindMSPInfo(mh.MSPID)
	if err != nil {
		return err
	}
	if mi.Freezed {
		return errors.New("MSPInfo is freezed")
	}
	policy := clientState.LastChaincodeInfo.IbcPolicy
	lastConfigs, err := clientState.LastMspInfos.GetMSPPBConfigs()
	if err != nil {
		return err
	}
	return VerifyEndorsedMessage(policy, *mh.Proof, mh.GetSignBytes(), lastConfigs)
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

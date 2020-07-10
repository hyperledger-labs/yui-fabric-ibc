package types

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
)

func GetPolicyEvaluator(policyBytes []byte, config Config) (policies.Policy, error) {
	var ap peer.ApplicationPolicy
	if err := proto.Unmarshal(policyBytes, &ap); err != nil {
		return nil, err
	}
	sigp := ap.Type.(*peer.ApplicationPolicy_SignaturePolicy)
	mgr, err := LoadVerifyingMsps(config)
	if err != nil {
		return nil, err
	}
	pp := cauthdsl.EnvelopeBasedPolicyProvider{Deserializer: mgr}
	return pp.NewPolicy(sigp.SignaturePolicy)
}

func MakeSignedDataList(pr *Proof) []*protoutil.SignedData {
	var sigSet []*protoutil.SignedData
	for i := 0; i < len(pr.Signatures); i++ {
		msg := make([]byte, len(pr.Proposal)+len(pr.Identities[i]))
		copy(msg[:len(pr.Proposal)], pr.Proposal)
		copy(msg[len(pr.Proposal):], pr.Identities[i])

		sigSet = append(
			sigSet,
			&protoutil.SignedData{
				Data:      msg,
				Identity:  pr.Identities[i],
				Signature: pr.Signatures[i],
			},
		)
	}
	return sigSet
}

func GetTxReadWriteSetFromProposalResponsePayload(proposal []byte) (*peer.ChaincodeID, *rwsetutil.TxRwSet, error) {
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

func EnsureWriteSetIncludesCommitment(set []*rwsetutil.NsRwSet, nsIdx, rwsIdx uint32, targetKey string, expectValue []byte) (bool, error) {
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

func VerifyEndorsement(ccID peer.ChaincodeID, policyBytes []byte, proof Proof, path string, value []byte) (bool, error) {
	// TODO parameterize
	config, err := DefaultConfig()
	if err != nil {
		return false, err
	}
	sigSet := MakeSignedDataList(&proof)
	policy, err := GetPolicyEvaluator(policyBytes, config)
	if err != nil {
		return false, err
	}
	if err := policy.EvaluateSignedData(sigSet); err != nil {
		return false, err
	}

	id, rwset, err := GetTxReadWriteSetFromProposalResponsePayload(proof.Proposal)
	if err != nil {
		return false, err
	}
	if !equalChaincodeID(ccID, *id) {
		return false, fmt.Errorf("unexpected chaincodID: %v", *id)
	}
	return EnsureWriteSetIncludesCommitment(rwset.NsRwSets, proof.NsIndex, proof.WriteSetIndex, path, value)
}

func equalChaincodeID(a, b peer.ChaincodeID) bool {
	if a.Name == b.Name && a.Path == b.Path && a.Version == b.Version {
		return true
	} else {
		return false
	}
}

func VerifyChaincodeHeader(clientState ClientState, h ChaincodeHeader) error {
	lastci := clientState.LastChaincodeInfo
	ok, err := VerifyEndorsement(clientState.LastChaincodeInfo.GetFabricChaincodeID(), lastci.EndorsementPolicy, h.Proof, MakeSequenceCommitmentEntryKey(h.Sequence.Value), h.Sequence.Bytes())
	if err != nil {
		return err
	} else if !ok {
		return errors.New("failed to verify the endorsement")
	}
	return nil
}

func VerifyChaincodeInfo(clientState ClientState, info ChaincodeInfo) error {
	// TODO implement
	return nil
}

func LoadVerifyingMsps(conf Config) (msp.MSPManager, error) {
	msps := []msp.MSP{}
	// if in local, this would depend `peer.localMspType` config
	for _, id := range conf.MSPIDs {
		dir := filepath.Join(conf.MSPsDir, id)
		bccspConfig := factory.GetDefaultOpts()
		// XXX provider type should be selectable
		mspConf, err := msp.GetVerifyingMspConfig(dir, id, msp.ProviderTypeToString(msp.FABRIC))
		if err != nil {
			return nil, err
		}
		opts := msp.Options[msp.ProviderTypeToString(msp.FABRIC)]
		cryptoProvider, err := (&factory.SWFactory{}).Get(bccspConfig)
		if err != nil {
			return nil, err
		}
		m, err := msp.New(opts, cryptoProvider)
		if err != nil {
			return nil, err
		}
		if err := m.Setup(mspConf); err != nil {
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

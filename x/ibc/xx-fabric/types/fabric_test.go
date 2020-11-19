package types

import (
	"os"
	"path/filepath"
	"testing"

	fabrictests "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/tests"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadVerifyingMSPs(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	msps, err := loadVerifyingMsps(configForTest())
	require.NoError(err)

	dict, err := msps.GetMSPs()
	require.NoError(err)
	assert.Equal(2, len(dict))

	mspID := "Org1MSP"

	id, err := dict[mspID].GetIdentifier()
	require.NoError(err)
	assert.Equal(mspID, id)
}

func TestGetPolicyEvaluator(t *testing.T) {
	conf := configForTest()
	mspID := "Org2MSP"
	fixture, err := fabrictests.GetMSPFixture(conf.MSPsDir, mspID)
	require.NoError(t, err)

	plcBytes := makePolicy([]string{mspID})
	require.NoError(t, err)
	plc, err := getPolicyEvaluator(plcBytes, []MSPPBConfig{*fixture.MSPConf})
	require.NoError(t, err)

	// Evaluate a CommitmentProof
	cproof, err := makeCommitmentProof(fixture.Signer, "key1", []byte("val1"))
	require.NoError(t, err)
	assert.NoError(t, plc.EvaluateSignedData(cproof.ToSignedData()))

	// Evaluate a MessageProof
	proof, err := makeMessageProof(fixture.Signer, []byte("value"))
	require.NoError(t, err)
	sigs := makeSignedDataListWithMessageProof(*proof, []byte("value"))
	require.NoError(t, plc.EvaluateSignedData(sigs))
}

func TestVerifyEndorsedMessage(t *testing.T) {
	conf := configForTest()
	org1, err := fabrictests.GetMSPFixture(conf.MSPsDir, "Org1MSP")
	require.NoError(t, err)

	type args struct {
		policyBytes []byte
		value       []byte
		configs     []MSPPBConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add more test cases.
		{"valid case", args{
			policyBytes: makePolicy([]string{org1.MSPID}), value: []byte("value"), configs: []MSPPBConfig{*org1.MSPConf},
		}, false},
		{"invalid for policy", args{
			policyBytes: makePolicy([]string{"OTHER_MSP"}), value: []byte("value"), configs: []MSPPBConfig{*org1.MSPConf},
		}, true},
	}
	for _, tt := range tests {
		// generate proof for each case
		sig, err := org1.Signer.Sign(tt.args.value)
		require.NoError(t, err)
		signerIdentity, err := org1.Signer.Serialize()
		require.NoError(t, err)
		proof := MessageProof{
			Identities: [][]byte{signerIdentity},
			Signatures: [][]byte{sig},
		}
		t.Run(tt.name, func(t *testing.T) {
			if err := VerifyEndorsedMessage(tt.args.policyBytes, proof, tt.args.value, tt.args.configs); (err != nil) != tt.wantErr {
				t.Errorf("VerifyEndorsedMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyMSPConfig(t *testing.T) {
	conf := configForTest()
	org1, err := fabrictests.GetMSPFixture(conf.MSPsDir, "Org1MSP")
	require.NoError(t, err)
	confOrg1, err := proto.Marshal(org1.MSPConf)
	require.NoError(t, err)

	org2, err := fabrictests.GetMSPFixture(conf.MSPsDir, "Org2MSP")
	require.NoError(t, err)

	type args struct {
		lastMSPInfos MSPInfos
		config       MSPConfig
		signer       msp.SigningIdentity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"valid create case", args{
			lastMSPInfos: MSPInfos{
				Infos: []MSPInfo{
					{MSPID: org1.MSPID, Config: confOrg1, Policy: makePolicy([]string{org1.MSPID})},
					{MSPID: org2.MSPID, Config: nil, Policy: makePolicy([]string{org1.MSPID})},
				},
			},
			config: MSPConfig{
				Type: TypeCreate, MSPID: org2.MSPID, Config: []byte("config"), Proof: nil,
			},
			signer: org1.Signer,
		}, false},
		{"valid update case", args{
			lastMSPInfos: MSPInfos{
				Infos: []MSPInfo{
					{MSPID: org1.MSPID, Config: confOrg1, Policy: makePolicy([]string{org1.MSPID})},
				},
			},
			config: MSPConfig{
				Type: TypeUpdate, MSPID: org1.MSPID, Config: []byte("config"), Proof: nil,
			},
			signer: org1.Signer,
		}, false},
		{"no msp policy", args{
			lastMSPInfos: MSPInfos{
				Infos: []MSPInfo{},
			},
			config: MSPConfig{
				Type: TypeCreate, MSPID: org1.MSPID, Config: []byte("config"), Proof: nil,
			},
			signer: org1.Signer,
		}, true},
		{"invalid signer for the msp policy", args{
			lastMSPInfos: MSPInfos{
				Infos: []MSPInfo{
					{MSPID: org1.MSPID, Config: confOrg1, Policy: makePolicy([]string{org1.MSPID})},
					{MSPID: org2.MSPID, Config: nil, Policy: makePolicy([]string{"OTHER_MSP"})},
				},
			},
			config: MSPConfig{
				Type: TypeCreate, MSPID: org2.MSPID, Config: []byte("config"), Proof: nil,
			},
			signer: org1.Signer,
		}, true},
		{"invalid ActionType", args{
			lastMSPInfos: MSPInfos{
				Infos: []MSPInfo{
					{MSPID: org1.MSPID, Config: confOrg1, Policy: makePolicy([]string{org1.MSPID})},
					{MSPID: org2.MSPID, Config: nil, Policy: makePolicy([]string{org1.MSPID})},
				},
			},
			config: MSPConfig{
				Type: TypeUpdate, MSPID: org2.MSPID, Config: []byte("config"), Proof: nil,
			},
			signer: org1.Signer,
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// generate proof for MSPConfig for each case
			sig, err := tt.args.signer.Sign(tt.args.config.GetSignBytes())
			require.NoError(t, err)
			signerIdentity, err := tt.args.signer.Serialize()
			require.NoError(t, err)
			proof := &MessageProof{
				Identities: [][]byte{signerIdentity},
				Signatures: [][]byte{sig},
			}
			tt.args.config.Proof = proof

			cs := ClientState{ID: "id", LastChaincodeHeader: ChaincodeHeader{}, LastChaincodeInfo: ChaincodeInfo{},
				LastMSPInfos: tt.args.lastMSPInfos}
			if err := VerifyMSPConfig(cs, tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("VerifyMSPConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyMSPPolicy(t *testing.T) {
	conf := configForTest()
	org1, err := fabrictests.GetMSPFixture(conf.MSPsDir, "Org1MSP")
	require.NoError(t, err)
	confOrg1, err := proto.Marshal(org1.MSPConf)
	require.NoError(t, err)

	org2, err := fabrictests.GetMSPFixture(conf.MSPsDir, "Org2MSP")
	require.NoError(t, err)

	type args struct {
		lastMSPInfos MSPInfos
		ibcPolicy    []byte
		policy       MSPPolicy
		signer       msp.SigningIdentity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"valid create case", args{
			lastMSPInfos: MSPInfos{[]MSPInfo{
				{MSPID: org1.MSPID, Config: confOrg1, Policy: makePolicy([]string{org1.MSPID})},
			}},
			ibcPolicy: makePolicy([]string{org1.MSPID}),
			policy: MSPPolicy{
				Type: TypeCreate, MSPID: org2.MSPID, Policy: []byte("policy"), Proof: nil,
			},
			signer: org1.Signer,
		}, false},
		{"valid update case", args{
			lastMSPInfos: MSPInfos{[]MSPInfo{
				{MSPID: org1.MSPID, Config: confOrg1, Policy: makePolicy([]string{org1.MSPID})},
			}},
			ibcPolicy: makePolicy([]string{org1.MSPID}),
			policy: MSPPolicy{
				Type: TypeUpdate, MSPID: org1.MSPID, Policy: []byte("policy"), Proof: nil,
			},
			signer: org1.Signer,
		}, false},
		{"need at least one MSPInfo before verifying", args{
			lastMSPInfos: MSPInfos{[]MSPInfo{}},
			ibcPolicy:    makePolicy([]string{org1.MSPID}),
			policy: MSPPolicy{
				Type: TypeCreate, MSPID: org1.MSPID, Policy: []byte("policy"), Proof: nil,
			},
			signer: org1.Signer,
		}, true},
		{"re-create is invalid", args{
			lastMSPInfos: MSPInfos{[]MSPInfo{
				{MSPID: org1.MSPID, Config: confOrg1, Policy: makePolicy([]string{org1.MSPID})},
			}},
			ibcPolicy: makePolicy([]string{org1.MSPID}),
			policy: MSPPolicy{
				Type: TypeCreate, MSPID: org1.MSPID, Policy: []byte("policy"), Proof: nil,
			},
			signer: org1.Signer,
		}, true},
		{"invalid signer", args{
			lastMSPInfos: MSPInfos{[]MSPInfo{
				{MSPID: org1.MSPID, Config: confOrg1, Policy: makePolicy([]string{org1.MSPID})},
			}},
			ibcPolicy: makePolicy([]string{org1.MSPID}),
			policy: MSPPolicy{
				Type: TypeCreate, MSPID: org2.MSPID, Policy: []byte("policy"), Proof: nil,
			},
			signer: org2.Signer,
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// generate proof for MSPConfig for each case
			sig, err := tt.args.signer.Sign(tt.args.policy.GetSignBytes())
			require.NoError(t, err)
			signerIdentity, err := tt.args.signer.Serialize()
			require.NoError(t, err)
			proof := &MessageProof{
				Identities: [][]byte{signerIdentity},
				Signatures: [][]byte{sig},
			}
			tt.args.policy.Proof = proof

			cs := ClientState{
				ID:                  "id",
				LastChaincodeHeader: ChaincodeHeader{},
				LastChaincodeInfo:   ChaincodeInfo{IbcPolicy: tt.args.ibcPolicy},
				LastMSPInfos:        tt.args.lastMSPInfos,
			}
			if err := VerifyMSPPolicy(cs, tt.args.policy); (err != nil) != tt.wantErr {
				t.Errorf("VerifyMSPPolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func configForTest() Config {
	wd, _ := os.Getwd()
	return Config{
		MSPsDir: filepath.Join(wd, "..", "..", "..", "..", "tests", "fixtures", "organizations", "peerOrganizations"),
		MSPIDs:  []string{"Org1MSP", "Org2MSP"},
	}
}

func makePolicy(mspids []string) []byte {
	return protoutil.MarshalOrPanic(&common.ApplicationPolicy{
		Type: &common.ApplicationPolicy_SignaturePolicy{
			SignaturePolicy: policydsl.SignedByNOutOfGivenRole(int32(len(mspids)/2+1), msppb.MSPRole_MEMBER, mspids),
		},
	})
}

func makeMessageProof(signer protoutil.Signer, value []byte) (*MessageProof, error) {
	pr := &MessageProof{}
	sig, err := signer.Sign(value)
	if err != nil {
		return nil, err
	}
	id, err := signer.Serialize()
	if err != nil {
		return nil, err
	}
	pr.Signatures = append(
		pr.Signatures,
		sig,
	)
	pr.Identities = append(
		pr.Identities,
		id,
	)
	return pr, nil
}

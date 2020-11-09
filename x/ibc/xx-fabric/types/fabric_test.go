package types

import (
	"os"
	"path/filepath"
	"testing"

	fabrictests "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/tests"
	"github.com/hyperledger/fabric-protos-go/common"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/common/policydsl"
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
	config := configForTest()
	mspID := "Org2MSP"
	plcBytes := makePolicy([]string{mspID})
	// load verifying msp configs inside it
	mconf, err := fabrictests.GetLocalVerifyingMspConfig(config.MSPsDir, mspID)
	require.NoError(t, err)
	plc, err := getPolicyEvaluator(plcBytes, []msppb.MSPConfig{*mconf})
	require.NoError(t, err)
	msp, err := fabrictests.GetLocalMsp(config.MSPsDir, mspID)
	require.NoError(t, err)
	si, err := msp.GetDefaultSigningIdentity()
	require.NoError(t, err)

	// Evaluate a CommitmentProof
	cproof, err := makeCommitmentProof(si, "key1", []byte("val1"))
	require.NoError(t, err)
	assert.NoError(t, plc.EvaluateSignedData(cproof.ToSignedData()))

	// Evaluate a MessageProof
	proof, err := makeMessageProof(si, []byte("value"))
	require.NoError(t, err)
	sigs := makeSignedDataListWithMessageProof(*proof, []byte("value"))
	require.NoError(t, plc.EvaluateSignedData(sigs))
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

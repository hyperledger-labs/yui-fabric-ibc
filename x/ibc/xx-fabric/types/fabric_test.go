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
	plc, err := getPolicyEvaluator(plcBytes, config)

	msp, err := fabrictests.GetLocalMsp(config.MSPsDir, mspID)
	require.NoError(t, err)
	si, err := msp.GetDefaultSigningIdentity()
	require.NoError(t, err)

	proof, err := makeProof(si, "key1", []byte("val1"))
	require.NoError(t, err)

	assert.NoError(t, plc.EvaluateSignedData(proof.ToSignedData()))
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

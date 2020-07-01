package typestest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/datachainlab/fabric-ibc/tests"
	"github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
	"github.com/hyperledger/fabric-protos-go/common"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMSPs(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	msps, err := types.LoadMSPs(configForTest())
	require.NoError(err)

	dict, err := msps.GetMSPs()
	require.NoError(err)
	assert.Equal(2, len(dict))

	mspID := "Org1MSP"

	id, err := dict[mspID].GetIdentifier()
	require.NoError(err)
	assert.Equal(mspID, id)

	t.Logf("%+v", dict[mspID])
	si, err := dict[mspID].GetDefaultSigningIdentity()
	require.NoError(err)
	assert.Equal(mspID, si.GetMSPIdentifier())
}

func TestGetPolicyEvaluator(t *testing.T) {
	config := configForTest()
	mspID := "Org2MSP"
	plcBytes := makePolicy([]string{mspID})
	plc, err := types.GetPolicyEvaluator(plcBytes, config)

	msps, _ := types.LoadMSPs(config)
	dict, _ := msps.GetMSPs()
	si, _ := dict[mspID].GetDefaultSigningIdentity()

	proof, err := tests.MakeProof(si, "key1", []byte("val1"))
	require.NoError(t, err)

	sigs := types.MakeSignedDataList(proof)
	assert.NoError(t, plc.EvaluateSignedData(sigs))
}

func configForTest() types.Config {
	wd, _ := os.Getwd()
	return types.Config{
		MSPsDir: filepath.Join(wd, "..", "..", "..", "..", "..", "tests", "fixtures", "organizations", "peerOrganizations"),
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

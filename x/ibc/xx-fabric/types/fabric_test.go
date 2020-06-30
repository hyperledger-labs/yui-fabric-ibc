package types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMSPs(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	msps, err := LoadMSPs(configForTest())
	require.NoError(err)

	dict, err := msps.GetMSPs()
	require.NoError(err)
	assert.Equal(3, len(dict))

	ideOrg1MSP, err := dict["Org1MSP"].GetIdentifier()
	require.NoError(err)
	assert.Equal("Org1MSP", ideOrg1MSP)
}

func configForTest() Config {
	wd, _ := os.Getwd()
	return Config{
		MSPsDir: filepath.Join(wd, "fixtures", "msps"),
		MSPIDs:  []string{"Org1MSP", "Org2MSP", "Org3MSP"},
	}
}

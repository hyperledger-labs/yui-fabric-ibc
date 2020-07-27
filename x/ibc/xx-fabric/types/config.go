package types

import (
	"fmt"
	"io/ioutil"
	"os"
)

const envFabricIbcPeerMSPsDir = "FABRIC_IBC_PEER_MSPS_DIR"
const envFabricIbcSignerMSPsDir = "FABRIC_IBC_SIGNER_MSPS_DIR"

type Config struct {
	MSPsDir string
	MSPIDs  []string
}

func DefaultPeerConfig() (Config, error) {
	dir := os.Getenv(envFabricIbcPeerMSPsDir)
	if dir == "" {
		return Config{}, fmt.Errorf("environment variable '%v' must be set", envFabricIbcPeerMSPsDir)
	}

	return defaultConfig(dir)
}

func DefaultSignerConfig() (Config, error) {
	dir := os.Getenv(envFabricIbcSignerMSPsDir)
	if dir == "" {
		return Config{}, fmt.Errorf("environment variable '%v' must be set", envFabricIbcSignerMSPsDir)
	}

	return defaultConfig(dir)
}

func defaultConfig(dir string) (Config, error) {
	var ids []string
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return Config{}, err
	}
	for _, fi := range fis {
		ids = append(ids, fi.Name())
	}
	return Config{
		MSPsDir: dir,
		MSPIDs:  ids,
	}, nil
}

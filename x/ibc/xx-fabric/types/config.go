package types

import (
	"fmt"
	"io/ioutil"
	"os"
)

const envFabricIbcMSPsDir = "FABRIC_IBC_MSPS_DIR"

type Config struct {
	MSPsDir string
	MSPIDs  []string
}

func DefaultConfig() (Config, error) {
	var ids []string
	dir := os.Getenv(envFabricIbcMSPsDir)
	if dir == "" {
		return Config{}, fmt.Errorf("environment variable '%v' must be set", envFabricIbcMSPsDir)
	} else {
		fis, err := ioutil.ReadDir(dir)
		if err != nil {
			return Config{}, err
		}
		for _, fi := range fis {
			ids = append(ids, fi.Name())
		}
	}
	return Config{
		MSPsDir: dir,
		MSPIDs:  ids,
	}, nil
}

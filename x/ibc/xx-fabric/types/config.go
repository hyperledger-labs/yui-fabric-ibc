package types

import (
	"io/ioutil"
	"os"
)

const envFabricIbcMSPsDir = "FABRIC_IBC_MSPS_DIR"

type Config struct {
	MSPsDir string
	MSPIDs  []string
}

func DefaultConfig() Config {
	var ids []string

	dir := os.Getenv(envFabricIbcMSPsDir)
	if dir != "" {
		fis, err := ioutil.ReadDir(dir)
		if err == nil {
			for _, fi := range fis {
				ids = append(ids, fi.Name())
			}
		}
	}
	return Config{
		MSPsDir: dir,
		MSPIDs:  ids,
	}
}

package types

import (
	"io/ioutil"
	"os"
)

type Config struct {
	MSPsDir string
	MSPIDs  []string
}

func DefaultConfig() Config {
	var ids []string

	dir := os.Getenv("FABRIC_IBC_MSPS_DIR")
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

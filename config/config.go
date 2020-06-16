package config

import "github.com/datachainlab/fabric-ibc/commitment"

type Config struct {
	CommitmentConfig commitment.CommitmentConfig
}

func DefaultConfig() Config {
	return Config{
		CommitmentConfig: commitment.DefaultConfig(),
	}
}

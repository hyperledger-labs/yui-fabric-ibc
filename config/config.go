package config

import "github.com/hyperledger-labs/yui-fabric-ibc/commitment"

type Config struct {
	CommitmentConfig commitment.CommitmentConfig
}

func DefaultConfig() Config {
	return Config{
		CommitmentConfig: commitment.DefaultConfig(),
	}
}

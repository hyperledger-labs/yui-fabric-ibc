package commitment

import "time"

type CommitmentConfig struct {
	MaxTimestampDiff time.Duration
}

func DefaultConfig() CommitmentConfig {
	return CommitmentConfig{
		MaxTimestampDiff: 30 * time.Second,
	}
}

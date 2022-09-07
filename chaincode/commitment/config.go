package commitment

import "time"

type CommitmentConfig struct {
	MinTimeInterval  time.Duration // Minimum time interval to advance the sequence
	MaxTimestampDiff time.Duration // Difference between client and endorser
}

func DefaultConfig() CommitmentConfig {
	return CommitmentConfig{
		MinTimeInterval:  5 * time.Second,
		MaxTimestampDiff: 30 * time.Second,
	}
}

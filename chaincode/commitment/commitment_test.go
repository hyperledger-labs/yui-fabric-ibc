package commitment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommitmentEntry(t *testing.T) {
	require := require.New(t)

	e0 := &Entry{Key: "key", Value: []byte("value")}
	c0 := e0.ToCommitment()
	e1, err := EntryFromCommitment(c0)
	require.NoError(err)
	require.Equal(e0, e1)
}

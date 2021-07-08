package commitment

import (
	"testing"
	"time"

	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	"github.com/golang/protobuf/ptypes/timestamp"
	testsstub "github.com/hyperledger-labs/yui-fabric-ibc/tests/stub"
	"github.com/stretchr/testify/require"
	tmtime "github.com/tendermint/tendermint/types/time"
)

func TestSequence(t *testing.T) {
	require := require.New(t)
	clientTk := newTimeKeeper()
	endorserTk := newTimeKeeper()

	stub := testsstub.MakeFakeStub()
	stub.GetTxTimestampStub = func() (*timestamp.Timestamp, error) {
		return &timestamp.Timestamp{Seconds: clientTk.Now().Unix()}, nil
	}

	prefix := commitmenttypes.NewMerklePrefix([]byte("ibc"))

	smgr := NewSequenceManager(DefaultConfig(), prefix).(*sequenceManager)
	smgr.SetClock(func() time.Time {
		return endorserTk.Now()
	})

	// valid sequence 1
	seq, err := smgr.InitSequence(stub)
	require.NoError(err)
	require.Equal(NewSequence(1, clientTk.Now().Unix()), *seq)
	clientTk.Add(10 * time.Second)

	// valid sequence 2
	_, err = smgr.UpdateSequence(stub)
	require.NoError(err)
	seq, err = smgr.getCurrentSequence(stub)
	require.NoError(err)
	require.Equal(NewSequence(2, clientTk.Now().Unix()), *seq)
	clientTk.Add(10 * time.Second)

	// invalid client timestamp(future)
	clientTk.Add(time.Minute)
	_, err = smgr.UpdateSequence(stub)
	require.Error(err)
	clientTk.Add(-time.Minute) // rollback

	// invalid client timestamp(past)
	clientTk.Add(-time.Minute)
	_, err = smgr.UpdateSequence(stub)
	require.Error(err)
	clientTk.Add(time.Minute)

	// valid sequence 3
	_, err = smgr.UpdateSequence(stub)
	require.NoError(err)
	seq, err = smgr.getCurrentSequence(stub)
	require.NoError(err)
	require.Equal(NewSequence(3, clientTk.Now().Unix()), *seq)
	clientTk.Add(10 * time.Second)
}

type timeKeeper struct {
	tm time.Time
}

func newTimeKeeper() *timeKeeper {
	return &timeKeeper{tm: tmtime.Now()}
}

func (tk timeKeeper) Now() time.Time {
	return tk.tm
}

func (tk *timeKeeper) Add(d time.Duration) {
	tk.tm = tk.tm.Add(d)
}

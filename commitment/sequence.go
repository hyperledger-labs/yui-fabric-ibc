package commitment

import (
	"errors"
	"time"

	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	proto "github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	tmtime "github.com/tendermint/tendermint/types/time"
)

func NewSequence(value uint64, tm int64) Sequence {
	return Sequence{Value: value, Timestamp: tm}
}

func (seq *Sequence) Bytes() []byte {
	bz, err := proto.Marshal(seq)
	if err != nil {
		panic(err)
	}
	return bz
}

type SequenceManager interface {
	InitSequence(stub shim.ChaincodeStubInterface) (*Sequence, error)
	GetCurrentSequence(stub shim.ChaincodeStubInterface) (*Sequence, error)
	GetSequence(stub shim.ChaincodeStubInterface, seq uint64) (*Sequence, error)
	UpdateSequence(stub shim.ChaincodeStubInterface) (*Sequence, error)
	ValidateTimestamp(now time.Time, prevSec int64, next *timestamp.Timestamp) error
	SetClock(clock func() time.Time)
}

type sequenceManager struct {
	config CommitmentConfig
	prefix exported.Prefix
	clock  func() time.Time
}

func NewSequenceManager(config CommitmentConfig, prefix exported.Prefix) SequenceManager {
	return &sequenceManager{config: config, prefix: prefix, clock: tmtime.Now}
}

func NewDefaultSequenceManager() SequenceManager {
	return NewSequenceManager(DefaultConfig(), commitmenttypes.NewMerklePrefix([]byte(host.StoreKey)))
}

func (m sequenceManager) InitSequence(stub shim.ChaincodeStubInterface) (*Sequence, error) {
	tm, err := stub.GetTxTimestamp()
	if err != nil {
		return nil, err
	}
	if err := m.ValidateTimestamp(m.clock(), 0, tm); err != nil {
		return nil, err
	}
	seq := NewSequence(1, tm.GetSeconds())
	if err = m.updateSequence(stub, seq); err != nil {
		return nil, err
	}
	return &seq, nil
}

func (m sequenceManager) GetCurrentSequence(stub shim.ChaincodeStubInterface) (*Sequence, error) {
	return m.getCurrentSequence(stub)
}

func (m sequenceManager) GetSequence(stub shim.ChaincodeStubInterface, seq uint64) (*Sequence, error) {
	return m.getSequence(stub, seq)
}

func (m sequenceManager) UpdateSequence(stub shim.ChaincodeStubInterface) (*Sequence, error) {
	current, err := m.getCurrentSequence(stub)
	if err != nil {
		return nil, err
	}

	tm, err := stub.GetTxTimestamp()
	if err != nil {
		return nil, err
	}
	if err := m.ValidateTimestamp(m.clock(), current.Timestamp, tm); err != nil {
		return nil, err
	}

	next := NewSequence(current.Value+1, tm.GetSeconds())
	if err := m.updateSequence(stub, next); err != nil {
		return nil, err
	}
	return &next, nil
}

func (m sequenceManager) ValidateTimestamp(now time.Time, prevSec int64, next *timestamp.Timestamp) error {
	if now.Unix()+int64(m.config.MaxTimestampDiff/time.Second) < next.GetSeconds() {
		return errors.New("the next timestamp indicates future time")
	}
	if now.Unix()-int64(m.config.MaxTimestampDiff/time.Second) > next.GetSeconds() {
		return errors.New("the next timestamp indicates past time")
	}
	if prevSec > 0 && prevSec+int64(m.config.MinTimeInterval/time.Second) > next.GetSeconds() {
		return errors.New("the next timestamp is less than the minimum time for advancing the sequence")
	}
	return nil
}

func (m *sequenceManager) SetClock(clock func() time.Time) {
	m.clock = clock
}

func (m sequenceManager) getCurrentSequence(stub shim.ChaincodeStubInterface) (*Sequence, error) {
	bz, err := stub.GetState(MakeCurrentSequenceKey(m.prefix))
	if err != nil {
		return nil, err
	} else if bz == nil {
		return nil, errors.New("key not found")
	}
	var seq Sequence
	if err := proto.Unmarshal(bz, &seq); err != nil {
		return nil, err
	}
	return &seq, nil
}

func (m sequenceManager) getSequence(stub shim.ChaincodeStubInterface, value uint64) (*Sequence, error) {
	bz, err := stub.GetState(MakeSequenceKey(m.prefix, value))
	if err != nil {
		return nil, err
	} else if bz == nil {
		return nil, errors.New("key not found")
	}
	var seq Sequence
	if err := proto.Unmarshal(bz, &seq); err != nil {
		return nil, err
	}
	return &seq, nil
}

func (m sequenceManager) updateSequence(stub shim.ChaincodeStubInterface, nextSeq Sequence) error {
	if err := stub.PutState(
		MakeCurrentSequenceKey(m.prefix),
		nextSeq.Bytes(),
	); err != nil {
		return err
	}

	if err := stub.PutState(
		MakeSequenceKey(m.prefix, nextSeq.Value),
		nextSeq.Bytes(),
	); err != nil {
		return err
	}

	return nil
}

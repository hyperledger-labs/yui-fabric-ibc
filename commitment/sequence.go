package commitment

import (
	"errors"
	"time"

	commitmentexported "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/exported"
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

type SequenceManager struct {
	config CommitmentConfig
	prefix commitmentexported.Prefix
	clock  func() time.Time
}

func NewSequenceManager(config CommitmentConfig, prefix commitmentexported.Prefix) SequenceManager {
	return SequenceManager{config: config, prefix: prefix, clock: tmtime.Now}
}

func (m SequenceManager) UpdateSequence(stub shim.ChaincodeStubInterface) error {
	current, err := m.getCurrentSequence(stub)
	if err != nil {
		return err
	}

	tm, err := stub.GetTxTimestamp()
	if err != nil {
		return err
	}
	if err := m.ValidateTimestamp(m.clock(), current.Timestamp, tm); err != nil {
		return err
	}

	next := NewSequence(current.Value+1, tm.GetSeconds())
	return m.updateSequence(stub, next)
}

func (m SequenceManager) ValidateTimestamp(now time.Time, prevSec int64, next *timestamp.Timestamp) error {
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

func (m SequenceManager) InitSequence(stub shim.ChaincodeStubInterface) (*Sequence, error) {
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

func (m SequenceManager) getCurrentSequence(stub shim.ChaincodeStubInterface) (*Sequence, error) {
	bz, err := stub.GetState(MakeCurrentSequenceKey(m.prefix))
	if err != nil {
		return nil, err
	}
	var seq Sequence
	if err := proto.Unmarshal(bz, &seq); err != nil {
		return nil, err
	}
	return &seq, nil
}

func (m SequenceManager) getSequence(stub shim.ChaincodeStubInterface, value uint64) (*Sequence, error) {
	bz, err := stub.GetState(MakeSequenceKey(m.prefix, value))
	if err != nil {
		return nil, err
	}
	var seq Sequence
	if err := proto.Unmarshal(bz, &seq); err != nil {
		return nil, err
	}
	return &seq, nil
}

func (m SequenceManager) updateSequence(stub shim.ChaincodeStubInterface, nextSeq Sequence) error {
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

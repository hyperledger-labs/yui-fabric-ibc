package compat

import (
	"log"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/hyperledger/fabric/core/chaincode/lifecycle/mock"
)

func MakeFakeStub() *mock.ChaincodeStub {
	fakeStub := &mock.ChaincodeStub{}

	fakePublicKVStore := map[string][]byte{}

	fakeStub.GetChannelIDReturns("test-channel")

	fakeStub.GetStateStub = func(key string) ([]byte, error) {
		log.Printf("Get key=%v value=%v", key, string(fakePublicKVStore[key]))
		return fakePublicKVStore[key], nil
	}
	fakeStub.PutStateStub = func(key string, value []byte) error {
		log.Printf("Put key=%v value=%v", key, string(value))
		fakePublicKVStore[key] = value
		return nil
	}
	fakeStub.GetStateByRangeStub = func(start, end string) (shim.StateQueryIteratorInterface, error) {
		fakeIterator := &mock.StateIterator{}
		i := 0
		for key, value := range fakePublicKVStore {
			if key >= start && key < end {
				fakeIterator.HasNextReturnsOnCall(i, true)
				fakeIterator.NextReturnsOnCall(i, &queryresult.KV{
					Key:   key,
					Value: value,
				}, nil)
				i++
			}
		}
		return fakeIterator, nil
	}

	return fakeStub
}

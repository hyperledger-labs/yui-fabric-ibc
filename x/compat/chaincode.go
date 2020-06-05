package compat

import (
	"encoding/hex"
	"sort"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/hyperledger/fabric/core/chaincode/lifecycle/mock"
)

func MakeFakeStub() *mock.ChaincodeStub {
	fakeStub := &mock.ChaincodeStub{}

	fakePublicKVStore := map[string][]byte{}

	fakeStub.GetChannelIDReturns("test-channel")

	fakeStub.GetStateStub = func(key string) ([]byte, error) {
		// log.Printf("Get key=%v(%v) value=%v(%x)", decodeHex(key), key, fakePublicKVStore[key], fakePublicKVStore[key])
		return fakePublicKVStore[key], nil
	}
	fakeStub.PutStateStub = func(key string, value []byte) error {
		// log.Printf("Put key=%v(%v) value=%v(%x)", decodeHex(key), key, value, value)
		fakePublicKVStore[key] = value
		return nil
	}
	fakeStub.DelStateStub = func(key string) error {
		// log.Printf("Delete key=%v(%v)", decodeHex(key), key)
		delete(fakePublicKVStore, key)
		return nil
	}
	fakeStub.GetStateByRangeStub = func(start, end string) (shim.StateQueryIteratorInterface, error) {
		fakeIterator := &mock.StateIterator{}
		var items []item
		for key, value := range fakePublicKVStore {
			if (key >= start || start == "") && (key < end || end == "") {
				items = append(items, item{key: key, value: value})
			}
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].key < items[j].key
		})
		for i := 0; i < len(items); i++ {
			key, value := items[i].key, items[i].value
			fakeIterator.HasNextReturnsOnCall(i, true)
			fakeIterator.NextReturnsOnCall(i, &queryresult.KV{
				Key:   key,
				Value: value,
			}, nil)
		}
		return fakeIterator, nil
	}

	return fakeStub
}

func decodeHex(s string) string {
	bz, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return string(bz)
}

type item struct {
	key   string
	value []byte
}

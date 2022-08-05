package stub

import (
	"encoding/hex"
	"fmt"
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
		// fmt.Printf("Get key=%v(%v)\n", decodeHex(key), key)
		return fakePublicKVStore[key], nil
	}
	fakeStub.PutStateStub = func(key string, value []byte) error {
		// fmt.Printf("Put key=%v(%v)\n", decodeHex(key), key)
		fakePublicKVStore[key] = value
		return nil
	}
	fakeStub.DelStateStub = func(key string) error {
		// fmt.Printf("Delete key=%v(%v)\n", decodeHex(key), key)
		delete(fakePublicKVStore, key)
		return nil
	}
	fakeStub.GetStateByRangeStub = func(start, end string) (shim.StateQueryIteratorInterface, error) {
		// fmt.Printf("GetStateByRange start=%v(%v) end=%v(%v)\n", decodeHex(start), start, decodeHex(end), end)
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

		iter := &itemIterator{seq: 0, items: items}
		fakeIterator.HasNextCalls(iter.HasNext)
		fakeIterator.NextCalls(iter.Next)

		return fakeIterator, nil
	}

	return fakeStub
}

type itemIterator struct {
	seq   int
	items []item
}

func (it *itemIterator) HasNext() bool {
	return it.seq < len(it.items)
}

func (it *itemIterator) Next() (*queryresult.KV, error) {
	if !it.HasNext() {
		return nil, fmt.Errorf("no such kv")
	}
	item := it.items[it.seq]
	it.seq++
	return &queryresult.KV{Key: item.key, Value: item.value}, nil
}

func decodeHex(s string) string {
	bz, err := hex.DecodeString(s)
	if err != nil {
		return s
	}
	return string(bz)
}

type item struct {
	key   string
	value []byte
}

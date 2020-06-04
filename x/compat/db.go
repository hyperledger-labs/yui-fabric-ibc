package compat

import (
	"reflect"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	dbm "github.com/tendermint/tm-db"
)

var _ dbm.DB = (*DB)(nil)

type DB struct {
	stub shim.ChaincodeStubInterface
}

func NewDB(stub shim.ChaincodeStubInterface) *DB {
	return &DB{stub: stub}
}

func (db *DB) Get(key []byte) ([]byte, error) {
	return db.stub.GetState(string(key))
}

func (db *DB) Has(key []byte) (bool, error) {
	v, err := db.Get(key)
	if v == nil && err == nil {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (db *DB) Set(key, value []byte) error {
	return db.stub.PutState(string(key), value)
}

func (db *DB) SetSync(key, value []byte) error {
	return db.Set(key, value)
}

func (db *DB) Delete(key []byte) error {
	return db.stub.DelState(string(key))
}

func (db *DB) DeleteSync(key []byte) error {
	return db.Delete(key)
}

func (db *DB) Iterator(start, end []byte) (dbm.Iterator, error) {
	iter, err := db.stub.GetStateByRange(string(start), string(end))
	if err != nil {
		return nil, err
	}
	return NewIterator(start, end, iter), nil
}

func (db *DB) ReverseIterator(start, end []byte) (dbm.Iterator, error) {
	iter, err := db.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	return ReverseIterator(iter), nil
}

func (db *DB) Close() error {
	return nil
}

func (db *DB) NewBatch() dbm.Batch {
	return &BatchDB{db: db}
}

func (db *DB) Print() error {
	panic("not implemented error")
}

func (db *DB) Stats() map[string]string {
	panic("not implemented error")
}

type BatchDB struct { // FIXME fix this poor impl
	db       *DB
	commands []command
}

func (db *BatchDB) Close() {}

func (db *BatchDB) Write() error {
	for _, cmd := range db.commands {
		if err := cmd.Exec(db.db, false); err != nil {
			return err
		}
	}
	return nil
}

func (db *BatchDB) WriteSync() error {
	for _, cmd := range db.commands {
		if err := cmd.Exec(db.db, true); err != nil {
			return err
		}
	}
	return nil
}

func (db *BatchDB) Set(key, value []byte) {
	db.commands = append(db.commands, setCommand{key: key, value: value})
}

func (db *BatchDB) Delete(key []byte) {
	db.commands = append(db.commands, deleteCommand{key: key})
}

type command interface {
	Exec(db *DB, sync bool) error
}

type setCommand struct {
	key   []byte
	value []byte
}

func (cmd setCommand) Exec(db *DB, sync bool) error {
	if sync {
		return db.SetSync(cmd.key, cmd.value)
	} else {
		return db.Set(cmd.key, cmd.value)
	}
}

type deleteCommand struct {
	key []byte
}

func (cmd deleteCommand) Exec(db *DB, sync bool) error {
	if sync {
		return db.DeleteSync(cmd.key)
	} else {
		return db.Delete(cmd.key)
	}
}

var _ dbm.Iterator = (*Iterator)(nil)

type Iterator struct {
	start []byte
	end   []byte

	current *queryresult.KV
	qi      shim.StateQueryIteratorInterface
}

func NewIterator(start, end []byte, qi shim.StateQueryIteratorInterface) *Iterator {
	iter := &Iterator{
		start: start,
		end:   end,
		qi:    qi,
	}
	iter.Next()
	return iter
}

func (iter *Iterator) Domain() ([]byte, []byte) {
	panic("not implemented error")
}

func (iter *Iterator) Valid() bool {
	return iter.qi.HasNext()
}

func (iter *Iterator) Next() {
	kv, err := iter.qi.Next()
	if err != nil {
		panic(err)
	}
	iter.current = kv
}

func (iter *Iterator) Key() []byte {
	return []byte(iter.current.Key)
}

func (iter *Iterator) Value() []byte {
	return []byte(iter.current.Value)
}

func (iter *Iterator) Error() error {
	return nil
}

func (iter *Iterator) Close() {
	if err := iter.qi.Close(); err != nil {
		panic(err)
	}
}

func ReverseIterator(itr dbm.Iterator) dbm.Iterator {
	var items []iterItem
	for itr.Next(); itr.Valid(); itr.Next() {
		items = append(items, iterItem{key: itr.Key(), value: itr.Value()})
	}
	reverseAnySlice(items)
	return NewSimpleIterator(items)
}

func reverseAnySlice(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

type iterItem struct {
	key   []byte
	value []byte
}

type simpleIterator struct {
	items   []iterItem
	current int
}

var _ dbm.Iterator = (*simpleIterator)(nil)

func NewSimpleIterator(items []iterItem) *simpleIterator {
	return &simpleIterator{items: items, current: 0}
}

func (iter *simpleIterator) Domain() ([]byte, []byte) {
	panic("not implemented error")
}

func (iter *simpleIterator) Valid() bool {
	return len(iter.items) > iter.current
}

func (iter *simpleIterator) Next() {
	if !iter.Valid() {
		panic("iterator has ended")
	}
	iter.current++
}

func (iter *simpleIterator) Key() []byte {
	return iter.items[iter.current].key
}

func (iter *simpleIterator) Value() []byte {
	return iter.items[iter.current].value
}

func (iter *simpleIterator) Error() error {
	return nil
}

func (iter *simpleIterator) Close() {}

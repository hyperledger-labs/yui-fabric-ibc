package compat

import (
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
	panic("not implemented error")
}

func (db *DB) Close() error {
	return nil
}

func (db *DB) NewBatch() dbm.Batch {
	return &BatchDB{db}
}

func (db *DB) Print() error {
	panic("not implemented error")
}

func (db *DB) Stats() map[string]string {
	panic("not implemented error")
}

type BatchDB struct { // TODO this is dummy
	*DB
}

func (db *BatchDB) Close() {

}

func (db *BatchDB) Write() error {
	return nil
}

func (db *BatchDB) WriteSync() error {
	return nil
}

func (db *BatchDB) Set(key, value []byte) {
	db.DB.Set(key, value)
}

func (db *BatchDB) Delete(key []byte) {
	db.DB.Delete(key)
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

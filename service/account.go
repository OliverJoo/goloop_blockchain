package service

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/pkg/errors"
	mp "github.com/ugorji/go/codec"
	"log"
	"math/big"
)

type accountSnapshot interface {
	trie.Object
	getBalance() *big.Int
	isContract() bool
	isEmpty() bool
	getValue(k []byte) ([]byte, error)
}

type accountState interface {
	getBalance() *big.Int
	setBalance(v *big.Int)
	isContract() bool
	getValue(k []byte) ([]byte, error)
	setValue(k, v []byte) error
	deleteValue(k []byte) error
	getSnapshot() accountSnapshot
	reset(snapshot accountSnapshot) error
}

type accountSnapshotImpl struct {
	balance     common.HexInt
	fIsContract bool
	store       trie.Immutable
	database    db.Database
}

func (s *accountSnapshotImpl) getBalance() *big.Int {
	v := new(big.Int)
	v.Set(&s.balance.Int)
	return v
}

func (s *accountSnapshotImpl) setBalance(*big.Int) {
	log.Printf("setBalance with readonly snapshot err=%+v",
		errors.New("PermissionDenied"))
}

func (s *accountSnapshotImpl) isContract() bool {
	return s.fIsContract
}

func (s *accountSnapshotImpl) Bytes() []byte {
	b, err := codec.MP.MarshalToBytes(s)
	if err != nil {
		panic(err)
	}
	return b
}

func (s *accountSnapshotImpl) CodecEncodeSelf(e *mp.Encoder) {
	e.Encode(s.balance)
	e.Encode(s.fIsContract)
	e.Encode(s.store.Hash())
}

func (s *accountSnapshotImpl) CodecDecodeSelf(d *mp.Decoder) {
	if err := d.Decode(&s.balance); err != nil {
		log.Fatalf("Fail to decode balance in account")
	}
	if err := d.Decode(&s.fIsContract); err != nil {
		log.Fatalf("Fail to decode isContract in account")
	}
	var hash []byte
	if err := d.Decode(&hash); err != nil {
		log.Fatalf("Fail to decode hash in account")
	} else {
		if len(hash) == 0 {
			s.store = nil
		} else {
			s.store = trie_manager.NewImmutable(s.database, hash)
		}
	}
}

func (s *accountSnapshotImpl) Reset(database db.Database, data []byte) error {
	s.database = database
	_, err := codec.MP.UnmarshalFromBytes(data, s)
	return err
}

func (s *accountSnapshotImpl) Flush() error {
	if sp, ok := s.store.(trie.Snapshot); ok {
		return sp.Flush()
	}
	return nil
}

func (s *accountSnapshotImpl) isEmpty() bool {
	return s.balance.BitLen() == 0 && s.store == nil
}

func (s *accountSnapshotImpl) Equal(object trie.Object) bool {
	if s2, ok := object.(*accountSnapshotImpl); ok {
		if s == s2 {
			return true
		}
		if s == nil || s2 == nil {
			return false
		}
		if s.fIsContract != s2.fIsContract ||
			s.balance.Cmp(&s2.balance.Int) != 0 {
			return false
		}
		if s.store == s2.store {
			return true
		}
		if s.store == nil || s2.store == nil {
			return false
		}
		return s.store.Equal(s2.store, false)
	} else {
		log.Panicf("Replacing accountSnapshotImpl with other object(%T)", object)
	}
	return false
}

func (s *accountSnapshotImpl) getValue(k []byte) ([]byte, error) {
	return s.store.Get(k)
}

type accountStateImpl struct {
	database    db.Database
	balance     common.HexInt
	fIsContract bool
	store       trie.Mutable
}

func (s *accountStateImpl) getBalance() *big.Int {
	v := new(big.Int)
	v.Set(&s.balance.Int)
	return v
}

func (s *accountStateImpl) setBalance(v *big.Int) {
	s.balance.Set(v)
}

func (s *accountStateImpl) isContract() bool {
	return s.fIsContract
}

func (s *accountStateImpl) getSnapshot() accountSnapshot {
	var store trie.Immutable
	if s.store != nil {
		store = s.store.GetSnapshot()
		if store.Empty() {
			store = nil
		}
	}
	return &accountSnapshotImpl{
		balance:     s.balance.Clone(),
		fIsContract: s.fIsContract,
		store:       store,
	}
}

func (s *accountStateImpl) reset(isnapshot accountSnapshot) error {
	snapshot, ok := isnapshot.(*accountSnapshotImpl)
	if !ok {
		log.Panicf("It tries to reset with invalid snapshot type=%T", s)
	}

	s.balance.Set(&snapshot.balance.Int)
	s.fIsContract = snapshot.fIsContract
	if s.store == nil && snapshot.store == nil {
		return nil
	}
	if s.store == nil {
		s.store = trie_manager.NewMutable(s.database, nil)
	}
	if snapshot.store == nil {
		s.store = nil
	} else {
		if err := s.store.Reset(snapshot.store); err != nil {
			log.Panicf("Fail to make accountStateImpl err=%v", err)
		}
	}
	return nil
}

func (s *accountStateImpl) getValue(k []byte) ([]byte, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.Get(k)
}

func (s *accountStateImpl) setValue(k, v []byte) error {
	if s.store == nil {
		s.store = trie_manager.NewMutable(s.database, nil)
	}
	return s.store.Set(k, v)
}

func (s *accountStateImpl) deleteValue(k []byte) error {
	if s.store == nil {
		return nil
	}
	return s.store.Delete(k)
}

func newAccountState(database db.Database, snapshot *accountSnapshotImpl) accountState {
	s := new(accountStateImpl)
	s.database = database
	if snapshot != nil {
		s.reset(snapshot)
	}
	return s
}

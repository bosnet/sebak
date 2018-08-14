package sebakstorage

import (
	"testing"
)

func newTestStateDB(t *testing.T) (*LevelDBBackend, *LevelDBBackend, *StateDB) {
	st, err := NewTestMemoryLevelDBBackend()
	if err != nil {
		t.Fatal(err)
	}
	ts, err := st.OpenTransaction()
	if err != nil {
		t.Fatal(err)
	}

	sdb := NewStateDB(ts)

	return st, ts, sdb
}

func TestStateDBInterface(t *testing.T) {
	var sdb DBBackend
	_, _, sdb = newTestStateDB(t)
	if _, err := sdb.Has("hello"); err != nil {
		t.Error(err)
	}
}

func TestStateDBNewCommit(t *testing.T) {

	st, _, sdb := newTestStateDB(t)
	defer st.Close()

	type Account struct {
		Address string
	}

	a := &Account{"address"}

	if err := sdb.New("b1", a); err != nil {
		t.Fatal(err)
	}

	if err := sdb.New("a1", a); err != nil {
		t.Fatal(err)
	}

	if len(sdb.changedkeys) != 2 {
		t.Fatal("statedb.changedkeys != 2")
	}

	hash, err := sdb.MakeHashString()
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Error("hash == ''")
	}

	a.Address = "changed_address"

	if err := sdb.Set("a1", a); err != nil {
		t.Fatal(err)
	}

	hash2, err := sdb.MakeHashString()
	if err != nil {
		t.Fatal(err)
	}

	if hash == hash2 {
		t.Logf("hash:%v hash2:%v", hash, hash2)
		t.Error("hash == hash2")
	}

	if ok, err := st.Has("a1"); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatal("a1 exists in db!")
	}

	if err := sdb.Remove("b1"); err != nil {
		t.Fatal(err)
	}

	if hash3, err := sdb.MakeHashString(); err != nil {
		t.Fatal(err)
	} else if hash == hash3 {
		t.Error("hash == hash3")
	}

	if err := sdb.Commit(); err != nil {
		t.Fatal(err)
	}

	if ok, err := st.Has("a1"); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatal("a1 is nil in db after commit!")
	}

	{
		var (
			a1 *Account
		)
		if err := st.Get("a1", &a1); err != nil {
			t.Fatal(err)
		}

		if a1.Address != a.Address {
			t.Logf("a1.Addr:%v", a1.Address)
			t.Error("a1.Address != a.Address")
		}
	}
}

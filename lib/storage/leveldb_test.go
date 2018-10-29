package storage

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
)

func TestLevelDBBackendNew(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	key := "showme"
	input := map[int]string{
		90: "99",
		91: "91",
		92: "92",
	}
	if err := st.New(key, input); err != nil {
		t.Errorf("failed to 'New' in leveldb: %v", err)
		return
	}

	fetched := map[int]string{}
	err := st.Get(key, &fetched)
	if err != nil {
		t.Errorf("failed to 'Get' in leveldb: %v", err)
		return
	}

	if !reflect.DeepEqual(input, fetched) {
		t.Errorf("failed to 'Get' the same input in leveldb")
		return
	}

	if err := st.New(key, input); err == nil {
		t.Errorf("'New' only for new key in leveldb")
		return
	}
}

func TestLevelDBBackendNews(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	input := map[string]int{}
	for i := 0; i < 100; i++ {
		input[fmt.Sprintf("%d", i)] = i
	}
	var args []Item
	for k, v := range input {
		args = append(
			args,
			Item{k, v},
		)
	}

	if err := st.News(args...); err != nil {
		t.Errorf("failed to `News`: %v", err)
	}

	for _, i := range args {
		if exists, err := st.Has(i.Key); !exists || err != nil {
			if !exists {
				t.Errorf("failed to `News`, key, '%s' is missing", i.Key)
			} else {
				t.Errorf("failed to `News`: %v", err)
			}
		}
	}
}

func TestLevelDBBackendHas(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	key := "showme"
	if exists, _ := st.Has(key); exists {
		t.Error("failed to 'Has' in leveldb")
		return
	}

	st.New(key, 10)

	if exists, _ := st.Has(key); !exists {
		t.Error("failed to 'Has' in leveldb")
		return
	}

	st.Remove(key)
	if exists, _ := st.Has(key); exists {
		t.Error("failed to 'Has' in leveldb")
		return
	}
}

func TestLevelDBBackendGetRaw(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	st.New("showme", "input")

	// when record does not exist, it should return errors.StorageRecordDoesNotExist
	if _, err := st.GetRaw("vacuum"); err != errors.StorageRecordDoesNotExist {
		t.Errorf("failed to GetRaw: want=%v have=%v", errors.StorageRecordDoesNotExist, err)
	}
}

func TestLevelDBBackendSet(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	key := "showme"
	input := 20

	if err := st.Set(key, input); err == nil {
		t.Errorf("'Set' must be failed with new key")
		return
	}

	st.New(key, input)

	if err := st.Set(key, input+1); err != nil {
		t.Errorf("failed to 'Set': %v", err)
		return
	}
}

func TestLevelDBBackendRemove(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	key := "showme"
	input := 20

	if err := st.Remove(key); err == nil {
		t.Errorf("'Remove' must be failed with new key")
		return
	}

	st.New(key, input)

	if err := st.Remove(key); err != nil {
		t.Errorf("failed to 'Rmove': %v", err)
		return
	}
	if exists, _ := st.Has(key); exists {
		t.Errorf("failed to 'Rmove': key must be removed")
		return
	}
}

func TestLevelDBIterator(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	total := 300
	filteredCount := 289

	expected := []string{}
	for i := 0; i < total; i++ {
		key := fmt.Sprintf("%03d", i)
		st.New(key, 0)

		if len(expected) < filteredCount {
			expected = append(expected, key)
		}
	}

	var collected []string
	it, closeFunc := st.GetIterator("", &DefaultListOptions{reverse: false})
	for {
		v, hasNext := it()
		if !hasNext {
			break
		}

		if v.N > uint64(filteredCount) {
			break
		}
		collected = append(collected, string(v.Key))
	}
	closeFunc()

	if len(collected) != filteredCount {
		t.Error("failed to fetch the exact number of items")
	}

	if !reflect.DeepEqual(expected, collected) {
		t.Error("failed to fetch the exact sequence of items")
	}

	return
}

func TestLevelDBIteratorSeek(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	total := 300

	expected := []string{}
	for i := 0; i < total; i++ {
		key := fmt.Sprintf("%03d", i)
		st.New(key, 0)

		expected = append(expected, key)
	}

	expected = expected[100:]

	var collected []string
	it, closeFunc := st.GetIterator("", &DefaultListOptions{reverse: false, cursor: []byte(fmt.Sprintf("%03d", 100))})
	for {
		v, hasNext := it()
		if !hasNext {
			break
		}

		collected = append(collected, string(v.Key))
	}
	closeFunc()

	if !reflect.DeepEqual(expected, collected) {
		t.Log(expected)
		t.Log(collected)
		t.Error("failed to fetch the exact sequence of items")
	}

	return
}

func TestLevelDBIteratorLimit(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	total := 300

	expected := []string{}
	for i := 0; i < total; i++ {
		key := fmt.Sprintf("%03d", i)
		st.New(key, 0)

		expected = append(expected, key)
	}

	expected = expected[:100]

	var collected []string
	it, closeFunc := st.GetIterator("", &DefaultListOptions{reverse: false, limit: 100})
	for {
		v, hasNext := it()
		if !hasNext {
			break
		}

		collected = append(collected, string(v.Key))
	}
	closeFunc()

	if !reflect.DeepEqual(expected, collected) {
		t.Log(expected)
		t.Log(collected)
		t.Error("failed to fetch the exact sequence of items")
	}

	return
}

func TestLevelDBIteratorReverseOrder(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	total := 30

	expected := []string{}
	for i := 0; i < total; i++ {
		key := fmt.Sprintf("%03d", i)
		st.New(key, 0)

		expected = append(expected, key)
	}

	var collected []string
	it, closeFunc := st.GetIterator("", &DefaultListOptions{reverse: true})
	for {
		v, hasNext := it()
		if !hasNext {
			break
		}

		collected = append(collected, string(v.Key))
	}
	closeFunc()

	for i, a := range expected {
		if a != collected[len(collected)-1-i] {
			t.Error("failed to reverse `GetIterator`")
		}
	}

	return
}

func TestLevelDBBackendTransactionNew(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	ts, _ := st.OpenTransaction()

	key0 := common.GetUniqueIDFromUUID()
	value0 := "findme"
	if err := ts.New(key0, value0); err != nil {
		t.Error(err)
		return
	}

	var returned string
	if err := ts.Get(key0, &returned); err != nil {
		t.Error(err)
		return
	}
	if returned != value0 {
		t.Errorf("wrong value returned; '%s' != '%s'", value0, returned)
		return
	}

	ts.Commit()

	var returnedAgain string
	if err := st.Get(key0, &returnedAgain); err != nil {
		t.Errorf("failed to get after 'Commit()': %v", err)
		return
	}
	if returnedAgain != value0 {
		t.Errorf("wrong value returned after 'Commit()'; '%s' != '%s'", value0, returnedAgain)
		return
	}

	return
}

func TestLevelDBBackendTransactionDiscard(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	ts, _ := st.OpenTransaction()

	key0 := common.GetUniqueIDFromUUID()
	value0 := "findme"
	if err := ts.New(key0, value0); err != nil {
		t.Error(err)
		return
	}

	var returned string
	if err := ts.Get(key0, &returned); err != nil {
		t.Error(err)
		return
	}
	if returned != value0 {
		t.Errorf("wrong value returned; '%s' != '%s'", value0, returned)
		return
	}

	ts.Discard()

	var returnedAgain string
	if err := st.Get(key0, &returnedAgain); err == nil {
		t.Errorf("value is stored after 'Discard()': %v", err)
		return
	}

	return
}

//TODO(anarcher): SubTests
func TestLevelDBWalk(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	kv := map[string]string{
		"test-1": "1",
		"test-2": "2",
		"test-3": "3",
		"test-4": "4",
		"test-5": "5",
	}
	for k, v := range kv {
		if err := st.New(k, v); err != nil {
			t.Fatal(err)
		}
	}

	if err := st.New("notest-1", "notest-1"); err != nil {
		t.Fatal(err)
	}

	var (
		walkedKeys []string
		cnt        int
	)

	walkOption := NewWalkOption("test-1", 10, false)
	err := st.Walk("test-", walkOption, func(k, v []byte) (bool, error) {
		cnt++
		walkedKeys = append(walkedKeys, string(k))
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if cnt != len(kv) {
		t.Errorf("want: %v have: %v", len(kv), cnt)
	}

	var keys []string
	for k, _ := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	require.Equal(t, keys, walkedKeys)

}

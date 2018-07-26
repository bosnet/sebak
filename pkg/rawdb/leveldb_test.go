package rawdb

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestNewLevelDb(t *testing.T) {
	name, err := ioutil.TempDir("/tmp", "sebak-ldb")
	defer os.RemoveAll(name)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	db, err := NewLevelDb(name)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	err = db.Put([]byte{1, 2, 3, 4}[:], []byte{4, 3, 2, 1}[:])
	bytes, err := db.Get([]byte{1, 2, 3, 4}[:])

	assert.Equal(t, []byte{4, 3, 2, 1}, bytes)
}

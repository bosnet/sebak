package eth

import (
	"bytes"
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"

	"boscoin.io/sebak/lib/storage"
)

func TestEthTrieHash(t *testing.T) {
	var root = ethcommon.Hash{}

	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	defer st.Close()

	db := NewEthDB(st)
	trie, err := NewTrie(root.Bytes(), db)
	if err != nil {
		t.Fatal(err)
	}

	hash1 := trie.Hash()

	if err := trie.Set([]byte("a"), []byte("b")); err != nil {
		t.Fatal(err)
	}

	hash2 := trie.Hash()

	hash3, err := trie.Commit()
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(hash1, hash2) {
		t.Errorf("hash1 == hash2 %v,%v", hash1, hash2)
	}

	if !bytes.Equal(hash2, hash3) {
		t.Errorf("hash2 != hash3 %v,%v", hash2, hash3)
	}
}

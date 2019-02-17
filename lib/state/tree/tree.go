package tree

// Tree is the interface that is Merkle tree db
//
// eth's trie or tendermint's IAVL can be an implementation of this interface
type ImmutableTree interface {
	Get(key []byte) (value []byte, err error)
	Hash() []byte //current working root hash
}

type MutableTree interface {
	ImmutableTree

	Set(key, value []byte) error
	Delete(key []byte) error

	Commit() ([]byte, error)
}

type Tree = MutableTree

// Loader is the interface that make new tree
//
// LoadMutableTree returns new tree db related with the hash
// LoadImmutableTree too.
type Loader interface {
	LoadMutableTree(hash []byte) (MutableTree, error)
	LoadImmutableTree(hash []byte) (ImmutableTree, error)
}

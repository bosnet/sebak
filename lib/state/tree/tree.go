package tree

// Tree is the interface that is Merkle tree db
//
// eth's trie or tendermint's IAVL can be an implementation of this interface
type Tree interface {
	Set(key, value []byte) error
	Get(key []byte) (value []byte, err error)
	Delete(key []byte) error

	Hash() []byte //current working root hash
	Commit() ([]byte, error)
}

// Builder is the interface that make new tree
//
// Build returns new tree db related with the hash
type Builder interface {
	Build(hash []byte) (Tree, error)
}

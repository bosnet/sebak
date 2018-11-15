package block

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/voting"
)

const (
	EventBlockPrefix string = "bk-saved"
)

type Block struct {
	Header
	Transactions        []string `json:"transactions"`         /* []Transaction.GetHash() */
	ProposerTransaction string   `json:"proposer_transaction"` /* ProposerTransaction */
	//PrevConsensusResult ConsensusResult

	Hash      string `json:"hash"`
	Proposer  string `json:"proposer"` /* Node.Address() */
	Round     uint64 `json:"round"`
	Confirmed string `json:"confirmed" rlp:"-"`
}

func (bck Block) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(bck)
	return
}

func (bck Block) String() string {
	encoded, _ := json.MarshalIndent(bck, "", "  ")
	return string(encoded)
}

func (bck Block) IsEmpty() bool {
	return len(bck.Hash) < 1
}

// NewBlock creates new block; `ptx` represents the
// `ProposerTransaction.GetHash()`.
func NewBlock(proposer string, basis voting.Basis, ptx string, transactions []string, proposedTime string) *Block {
	b := &Block{
		Header:              *NewBlockHeader(basis, getTransactionRoot(append([]string{ptx}, transactions...)), proposedTime),
		Transactions:        transactions,
		ProposerTransaction: ptx,
		Proposer:            proposer,
		Round:               basis.Round,
	}

	b.Hash = base58.Encode(common.MustMakeObjectHash(b))
	return b
}

func getTransactionRoot(txs []string) string {
	return common.MustMakeObjectHashString(txs) // TODO make root
}

func getBlockKey(hash string) string {
	return fmt.Sprintf("%s%s", common.BlockPrefixHash, hash)
}

func getBlockKeyPrefixHeight(height uint64) string {
	return fmt.Sprintf("%s%020d", common.BlockPrefixHeight, height)
}

func (b Block) NewBlockKeyConfirmed() string {
	return fmt.Sprintf(
		"%s%s-%s%s",
		common.BlockPrefixConfirmed, b.ProposedTime,
		common.EncodeUint64ToByteSlice(b.Height),
		common.GetUniqueIDFromUUID(),
	)
}

func (b *Block) Save(st *storage.LevelDBBackend) (err error) {
	key := getBlockKey(b.Hash)
	b.Confirmed = common.NowISO8601()

	var exists bool
	exists, err = st.Has(key)
	if err != nil {
		return
	} else if exists {
		return errors.BlockAlreadyExists
	}

	if err = st.New(key, b); err != nil {
		return
	}

	if err = st.New(b.NewBlockKeyConfirmed(), b.Hash); err != nil {
		return
	}
	if err = st.New(getBlockKeyPrefixHeight(b.Height), b.Hash); err != nil {
		return
	}

	observer.BlockObserver.Trigger(EventBlockPrefix, b)

	return
}

func (b Block) PreviousBlock(st *storage.LevelDBBackend) (blk Block, err error) {
	if b.Height == common.GenesisBlockHeight {
		err = errors.StorageRecordDoesNotExist
		return
	}

	return GetBlockByHeight(st, b.Height-1)
}

func (b Block) NextBlock(st *storage.LevelDBBackend) (Block, error) {
	return GetBlockByHeight(st, b.Height+1)
}

func GetBlock(st *storage.LevelDBBackend, hash string) (bt Block, err error) {
	err = st.Get(getBlockKey(hash), &bt)
	return
}

func GetBlockHeader(st *storage.LevelDBBackend, hash string) (bt Header, err error) {
	err = st.Get(getBlockKey(hash), &bt)
	return
}

func ExistsBlock(st *storage.LevelDBBackend, hash string) (exists bool, err error) {
	exists, err = st.Has(getBlockKey(hash))
	return
}

func ExistsBlockByHeight(st *storage.LevelDBBackend, height uint64) (exists bool, err error) {
	exists, err = st.Has(getBlockKeyPrefixHeight(height))
	return
}

func LoadBlocksInsideIterator(
	st *storage.LevelDBBackend,
	iterFunc func() (storage.IterItem, bool),
	closeFunc func(),
) (
	func() (Block, bool, []byte),
	func(),
) {

	return (func() (Block, bool, []byte) {
			item, hasNext := iterFunc()
			if !hasNext {
				return Block{}, false, item.Key
			}

			var hash string
			json.Unmarshal(item.Value, &hash)

			b, err := GetBlock(st, hash)
			if err != nil {
				return Block{}, false, item.Key
			}

			return b, hasNext, item.Key
		}), (func() {
			closeFunc()
		})
}

func LoadBlockHeadersInsideIterator(
	st *storage.LevelDBBackend,
	iterFunc func() (storage.IterItem, bool),
	closeFunc func(),
) (
	func() (Header, bool, []byte),
	func(),
) {

	return (func() (Header, bool, []byte) {
			item, hasNext := iterFunc()
			if !hasNext {
				return Header{}, false, []byte{}
			}

			var hash string
			json.Unmarshal(item.Value, &hash)

			b, err := GetBlockHeader(st, hash)
			if err != nil {
				return Header{}, false, []byte{}
			}

			return b, hasNext, item.Key
		}), (func() {
			closeFunc()
		})
}

func GetBlocksByConfirmed(st *storage.LevelDBBackend, options storage.ListOptions) (
	func() (Block, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(common.BlockPrefixConfirmed, options)

	return LoadBlocksInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockHeadersByConfirmed(st *storage.LevelDBBackend, options storage.ListOptions) (
	func() (Header, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(common.BlockPrefixConfirmed, options)

	return LoadBlockHeadersInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockByHeight(st *storage.LevelDBBackend, height uint64) (bt Block, err error) {
	var hash string
	if err = st.Get(getBlockKeyPrefixHeight(height), &hash); err != nil {
		return
	}

	return GetBlock(st, hash)
}

func GetBlockHeaderByHeight(st *storage.LevelDBBackend, height uint64) (bt Header, err error) {
	var hash string
	if err = st.Get(getBlockKeyPrefixHeight(height), &hash); err != nil {
		return
	}

	return GetBlockHeader(st, hash)
}

func GetLatestBlock(st *storage.LevelDBBackend) Block {
	// get latest blocks
	iterFunc, closeFunc := GetBlocksByConfirmed(st, storage.NewDefaultListOptions(true, nil, 1))
	b, _, _ := iterFunc()
	closeFunc()

	if b.Hash == "" {
		panic(errors.BlockNotFound)
	}

	return b
}

func WalkBlocks(st *storage.LevelDBBackend, option *storage.WalkOption, walkFunc func(*Block, []byte) (bool, error)) error {
	err := st.Walk(common.BlockPrefixHeight, option, func(key, value []byte) (bool, error) {
		var hash string
		if err := json.Unmarshal(value, &hash); err != nil {
			return false, err
		}

		b, err := GetBlock(st, hash)
		if err != nil {
			return false, err
		}
		return walkFunc(&b, key)
	})
	return err
}

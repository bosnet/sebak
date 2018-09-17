package block

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

type Block struct {
	Header
	Transactions []string `json:"transactions"` /* []Transaction.GetHash() */
	//PrevConsensusResult ConsensusResult

	Hash      string      `json:"hash"`
	Confirmed string      `json:"confirmed"`
	Proposer  string      `json:"proposer"` /* Node.Address() */
	Round     round.Round `json:"round"`
}

func (bck Block) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(bck)
	return
}

func (bck Block) String() string {
	encoded, _ := json.MarshalIndent(bck, "", "  ")
	return string(encoded)
}

func MakeGenesisBlock(st *storage.LevelDBBackend, account BlockAccount) Block {
	proposer := "" // null proposer
	round := round.Round{
		Number:      0,
		BlockHeight: 0,
		BlockHash:   base58.Encode(common.MustMakeObjectHash(account)),
		TotalTxs:    0,
	}
	transactions := []string{}
	confirmed := ""

	b := NewBlock(
		proposer,
		round,
		transactions,
		confirmed,
	)
	b.Save(st)

	return b
}

func NewBlock(proposer string, round round.Round, transactions []string, confirmed string) Block {
	b := &Block{
		Header:       *NewBlockHeader(round, uint64(len(transactions)), getTransactionRoot(transactions)),
		Transactions: transactions,
		Proposer:     proposer,
		Round:        round,
		Confirmed:    confirmed,
	}

	log.Debug("NewBlock created", "PrevTotalTxs", round.TotalTxs, "txs", len(transactions), "TotalTxs", b.Header.TotalTxs)

	b.Hash = base58.Encode(common.MustMakeObjectHash(b))

	return *b
}

func NewBlockFromBallot(b ballot.Ballot) Block {
	return NewBlock(
		b.Proposer(),
		b.Round(),
		b.Transactions(),
		b.ProposerConfirmed(),
	)
}

func getTransactionRoot(txs []string) string {
	return common.MustMakeObjectHashString(txs) // TODO make root
}

func GetBlockKey(hash string) string {
	return fmt.Sprintf("%s%s", common.BlockPrefixHash, hash)
}

func GetBlockKeyPrefixConfirmed(confirmed string) string {
	return fmt.Sprintf("%s%s-", common.BlockPrefixConfirmed, confirmed)
}

func (b Block) NewBlockKeyConfirmed() string {
	return fmt.Sprintf(
		"%s%s%s",
		GetBlockKeyPrefixConfirmed(b.Confirmed),
		common.EncodeUint64ToByteSlice(b.Height),
		common.GetUniqueIDFromUUID(),
	)
}

func (b Block) Save(st *storage.LevelDBBackend) (err error) {
	key := GetBlockKey(b.Hash)

	var exists bool
	exists, err = st.Has(key)
	if err != nil {
		return
	} else if exists {
		return errors.ErrorBlockAlreadyExists
	}

	if err = st.New(key, b); err != nil {
		return
	}

	if err = st.New(b.NewBlockKeyConfirmed(), b.Hash); err != nil {
		return
	}

	return
}

func GetBlock(st *storage.LevelDBBackend, hash string) (bt Block, err error) {
	err = st.Get(GetBlockKey(hash), &bt)
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

func GetBlocksByConfirmed(st *storage.LevelDBBackend, iteratorOptions *storage.IteratorOptions) (
	func() (Block, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(common.BlockPrefixConfirmed, iteratorOptions)

	return LoadBlocksInsideIterator(st, iterFunc, closeFunc)
}

func GetLatestBlock(st *storage.LevelDBBackend) (b Block, err error) {
	// get latest blocks
	iterFunc, closeFunc := GetBlocksByConfirmed(st, &storage.IteratorOptions{Reverse: true})
	b, _, _ = iterFunc()
	closeFunc()

	if b.Hash == "" {
		err = errors.ErrorBlockNotFound
		return
	}

	return
}

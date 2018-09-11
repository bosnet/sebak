package block

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

const (
	BlockPrefixHash      string = "b-hash-"      // b-hash-<Block.Hash>
	BlockPrefixConfirmed string = "b-confirmed-" // b-hash-<Block.Confirmed>
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

func NewBlockFromBallot(ballot Ballot) Block {
	return NewBlock(
		ballot.Proposer(),
		ballot.Round(),
		ballot.Transactions(),
		ballot.ProposerConfirmed(),
	)
}

func getTransactionRoot(txs []string) string {
	return common.MustMakeObjectHashString(txs) // TODO make root
}

func GetBlockKey(hash string) string {
	return fmt.Sprintf("%s%s", BlockPrefixHash, hash)
}

func GetBlockKeyPrefixConfirmed(confirmed string) string {
	return fmt.Sprintf("%s%s-", BlockPrefixConfirmed, confirmed)
}

func (b Block) NewBlockKeyConfirmed() string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockKeyPrefixConfirmed(b.Confirmed),
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
	func() (Block, bool),
	func(),
) {

	return (func() (Block, bool) {
			item, hasNext := iterFunc()
			if !hasNext {
				return Block{}, false
			}

			var hash string
			json.Unmarshal(item.Value, &hash)

			b, err := GetBlock(st, hash)
			if err != nil {
				return Block{}, false
			}

			return b, hasNext
		}), (func() {
			closeFunc()
		})
}

func GetBlocksByConfirmed(st *storage.LevelDBBackend, reverse bool) (
	func() (Block, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(BlockPrefixConfirmed, reverse)

	return LoadBlocksInsideIterator(st, iterFunc, closeFunc)
}

func GetLatestBlock(st *storage.LevelDBBackend) (b Block, err error) {
	// get latest blocks
	iterFunc, closeFunc := GetBlocksByConfirmed(st, true)
	b, _ = iterFunc()
	closeFunc()

	if b.Hash == "" {
		err = errors.ErrorBlockNotFound
		return
	}

	return
}

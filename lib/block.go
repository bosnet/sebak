package sebak

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

const (
	BlockPrefixHash      string = "b-hash-"      // b-hash-<Block.Hash>
	BlockPrefixConfirmed string = "b-confirmed-" // b-hash-<Block.Confirmed>
)

type Block struct {
	Header
	Transactions []string /* []Transaction.GetHash() */
	//PrevConsensusResult ConsensusResult

	Hash      string
	Confirmed string
	Proposer  string /* Node.Address() */
	Round     Round
}

func MakeGenesisBlock(st *sebakstorage.LevelDBBackend, account BlockAccount) Block {
	proposer := "" // null proposer
	round := Round{
		Number:      0,
		BlockHeight: 0,
		BlockHash:   base58.Encode(sebakcommon.MustMakeObjectHash(account)),
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

func NewBlock(proposer string, round Round, transactions []string, confirmed string) Block {
	b := &Block{
		Header:       *NewBlockHeader(round.BlockHeight+1, round.BlockHash, round.TotalTxs, uint64(len(transactions)), getTransactionRoot(transactions)),
		Transactions: transactions,
		//PrevConsensusResult: ConsensusResult{},
		Proposer:  proposer,
		Round:     round,
		Confirmed: confirmed,
	}

	log.Debug("", "PrevTotalTxs", round.TotalTxs, "txs", uint64(len(transactions)), "TotalTxs", b.Header.TotalTxs)

	b.Hash = base58.Encode(sebakcommon.MustMakeObjectHash(b))
	//b.Header.fill()

	return *b
}

// func (b *Block) SetPrevConsensusResult(result *ConsensusResult) {
// 	b.PrevConsensusResult = *result
// }

func NewBlockFromRoundBallot(roundBallot RoundBallot) Block {
	return NewBlock(
		roundBallot.Proposer(),
		roundBallot.Round(),
		roundBallot.ValidTransactions(),
		roundBallot.ProposerConfirmed(),
	)
}

func getTransactionRoot(txs []string) string {
	return sebakcommon.MustMakeObjectHashString(txs) // [TODO] make root
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
		sebakcommon.GetUniqueIDFromUUID(),
	)
}

func (b Block) Save(st *sebakstorage.LevelDBBackend) (err error) {
	key := GetBlockKey(b.Hash)

	var exists bool
	exists, err = st.Has(key)
	if err != nil {
		return
	} else if exists {
		return sebakerror.ErrorBlockAlreadyExists
	}

	if err = st.New(key, b); err != nil {
		return
	}

	if err = st.New(b.NewBlockKeyConfirmed(), b.Hash); err != nil {
		return
	}

	return
}

func GetBlock(st *sebakstorage.LevelDBBackend, hash string) (bt Block, err error) {
	err = st.Get(GetBlockKey(hash), &bt)
	return
}

func LoadBlocksInsideIterator(
	st *sebakstorage.LevelDBBackend,
	iterFunc func() (sebakstorage.IterItem, bool),
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

func GetBlocksByConfirmed(st *sebakstorage.LevelDBBackend, reverse bool) (
	func() (Block, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(BlockPrefixConfirmed, reverse)

	return LoadBlocksInsideIterator(st, iterFunc, closeFunc)
}

func GetLatestBlock(st *sebakstorage.LevelDBBackend) (b Block, err error) {
	// get latest blocks
	iterFunc, closeFunc := GetBlocksByConfirmed(st, true)
	b, _ = iterFunc()
	closeFunc()

	if b.Hash == "" {
		err = sebakerror.ErrorBlockNotFound
		return
	}

	return
}

package api

import (
	"bufio"
	"bytes"
	"encoding/json"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/transaction"
)

type NodeItemDataType string

const (
	NodeItemBlock            NodeItemDataType = "block"
	NodeItemBlockHeader      NodeItemDataType = "block-header"
	NodeItemBlockTransaction NodeItemDataType = "block-transaction"
	NodeItemTransaction      NodeItemDataType = "transaction"
	NodeItemBallot           NodeItemDataType = "ballot"
	NodeItemError            NodeItemDataType = "error"
)

func UnmarshalNodeItemResponse(d []byte) (itemType NodeItemDataType, b interface{}, err error) {
	sc := bufio.NewScanner(bytes.NewReader(d))
	sc.Split(bufio.ScanWords)
	sc.Scan()
	if err = sc.Err(); err != nil {
		return
	}

	unmarshal := func(o interface{}) error {
		if err := json.Unmarshal(d[len(sc.Bytes())+1:], o); err != nil {
			return err
		}
		return nil
	}

	itemType = NodeItemDataType(sc.Text())
	switch itemType {
	case NodeItemBlock:
		var t block.Block
		err = unmarshal(&t)
		b = t
	case NodeItemBlockHeader:
		var t block.Header
		err = unmarshal(&t)
		b = t
	case NodeItemBlockTransaction:
		var t block.BlockTransaction
		err = unmarshal(&t)
		b = t
	case NodeItemTransaction:
		var t transaction.Transaction
		err = unmarshal(&t)
		b = t
	case NodeItemBallot:
		var t ballot.Ballot
		err = unmarshal(&t)
		b = t
	case NodeItemError:
		var t errors.Error
		err = unmarshal(&t)
		b = &t
	default:
		err = errors.InvalidMessage
	}

	return
}

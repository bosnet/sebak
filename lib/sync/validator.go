package sync

import (
	"fmt"
	"time"

	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
)

type BlockValidator struct {
	network network.Network
	storage *storage.LevelDBBackend

	validationTimeout time.Duration

	messages  <-chan *Message
	responses chan *Response

	stop chan chan struct{}
}

type BlockValidatorOption func(*BlockValidator)

var _ Validator = (*BlockValidator)(nil)

func NewBlockValidator(nw network.Network, ldb *storage.LevelDBBackend, opts ...BlockValidatorOption) *BlockValidator {
	v := &BlockValidator{
		network: nw,
		storage: ldb,

		messages:  nil,
		responses: make(chan *Response),
		stop:      make(chan chan struct{}),
	}

	for _, opt := range opts {
		opt(v)
	}

	go v.loop()

	return v
}

func (v *BlockValidator) Stop() error {
	c := make(chan struct{})
	v.stop <- c
	<-c
	return nil
}

func (v *BlockValidator) Consume(msg <-chan *Message) error {
	v.messages = msg
	return nil
}

func (v *BlockValidator) Response() <-chan *Response {
	return v.responses
}

func (v *BlockValidator) loop() {
	for {

		select {
		case msg := <-v.messages:
			//TODO validation steps
			// v.validate()
			// 		v.checkTxs()
			// 		v.block()
			//		v.saveTxs()
			//		v.saveBlock()
			// ...
			if err := v.validate(msg); err != nil {
				v.errorResponse(msg, fmt.Errorf("validation err:"))
				continue
			}
			if err := v.save(msg); err != nil {
				v.errorResponse(msg, fmt.Errorf("storage err:"))
			}
		case c := <-v.stop:
			close(c)
			return
		}
	}
}

func (v *BlockValidator) validate(msg *Message) error {
	//TODO:
	// ctx := context.WithTimeout(...
	return nil
}

func (v *BlockValidator) save(msg *Message) error {
	//TODO: using leveldb.Tx or leveldb.Batch?

	for _, op := range msg.Ops {
		if err := op.Save(v.storage); err != nil {
			return err
		}
	}

	for _, tx := range msg.Txs {
		if err := tx.Save(v.storage); err != nil {
			return err
		}
	}

	blk := msg.Block
	if err := blk.Save(v.storage); err != nil {
		return err
	}
	return nil
}

func (v *BlockValidator) errorResponse(msg *Message, err error) {
	v.responses <- &Response{
		msg: msg,
		err: err,
	}
}

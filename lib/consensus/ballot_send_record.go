package consensus

import (
	"sync"

	logging "github.com/inconshreveable/log15"
)

// Record the ballot sent by ISAACstate
// This is to avoid sending another voting result in the same ISAACState.
type BallotSendRecord struct {
	sync.RWMutex

	record map[ISAACState]bool
	log    logging.Logger
}

func NewBallotSendRecord(nodeAlias string) *BallotSendRecord {
	p := &BallotSendRecord{
		record: make(map[ISAACState]bool),
		log:    log.New(logging.Ctx{"node": nodeAlias}),
	}

	return p
}

func (r *BallotSendRecord) SetSent(state ISAACState) {
	r.Lock()
	defer r.Unlock()
	log.Debug("BallotSendRecord.SetSent()", "state", state)
	r.record[state] = true

	return
}

func (r *BallotSendRecord) Sent(state ISAACState) bool {
	r.RLock()
	defer r.RUnlock()
	log.Debug("BallotSendRecord.Sent()", "state", state)

	return r.record[state]
}

func (r *BallotSendRecord) RemoveLowerThanOrEqualHeight(height uint64) {
	r.Lock()
	defer r.Unlock()
	log.Debug("BallotSendRecord.RemoveLowerThanOrEqualHeight()", "height", height)

	for state := range r.record {
		if state.Height <= height {
			delete(r.record, state)
		}
	}

	return
}

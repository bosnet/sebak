package sebak

import (
	"boscoin.io/sebak/lib/common"
	"time"
)

type BallotBoxExpireMover struct {
	srcBox        *BallotBox
	targetBox     *BallotBox
	prevBoxHashes map[ /* VotingResultHash*/ string] /*Message.GetHash()*/ string
	votingResults map[ /*Message.GetHash()*/ string]*VotingResult

	retain time.Duration
	stop   chan chan struct{}
}

func NewBallotBoxExpireMover(srcBox, targetBox *BallotBox, votingResults map[string]*VotingResult, retain time.Duration) *BallotBoxExpireMover {
	e := &BallotBoxExpireMover{
		srcBox:        srcBox,
		targetBox:     targetBox,
		prevBoxHashes: make(map[string]string),
		votingResults: votingResults,
		retain:        retain,
		stop:          make(chan chan struct{}),
	}

	return e
}

func (e *BallotBoxExpireMover) Run() (err error) {
	ticker := time.NewTicker(e.retain)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.moveToTargetBox()
			e.makePrevHashesFromSrcBox()
		case q := <-e.stop:
			close(q)
			return
		}
	}
}

func (e *BallotBoxExpireMover) Stop() {
	q := make(chan struct{})
	e.stop <- q
	<-q
}

func (e *BallotBoxExpireMover) moveToTargetBox() {
	currBoxHashes := make(map[string]string)

	for hash, _ := range e.srcBox.Hashes {
		if vr, ok := e.votingResults[hash]; ok {
			if byteArr, err := sebakcommon.MakeObjectHash(vr); err == nil {
				currBoxHashes[string(byteArr)] = hash
			} else {
				log.Error("failed to make voting result hash", "VotingResult", vr, "error", err)
			}
		}
	}

	for vrHash, _ := range e.prevBoxHashes {
		if msgHash, ok := currBoxHashes[vrHash]; ok {
			if e.srcBox.HasMessageByHash(msgHash) { // srcBox.HasHash(h)?
				e.srcBox.RemoveHash(msgHash)
				e.targetBox.AddHash(msgHash)
			}
		}
	}
}

func (e *BallotBoxExpireMover) makePrevHashesFromSrcBox() {
	e.prevBoxHashes = make(map[string]string)

	for hash, _ := range e.srcBox.Hashes {
		if vr, ok := e.votingResults[hash]; ok {
			if byteArr, err := sebakcommon.MakeObjectHash(vr); err == nil {
				e.prevBoxHashes[string(byteArr)] = hash
			} else {
				log.Error("failed to make voting result hash", "VotingResult", vr, "error", err)
			}
		}
	}
}

type BallotBoxExpireRemover struct {
	srcBox        *BallotBox
	boxes         *BallotBoxes
	prevBoxHashes map[ /* VotingResultHash*/ string] /*Message.GetHash()*/ string
	votingResults map[string]*VotingResult

	retain time.Duration
	stop   chan chan struct{}
}

func NewBallotBoxExpireRemover(srcBox *BallotBox, boxes *BallotBoxes, votingResults map[string]*VotingResult, retain time.Duration) *BallotBoxExpireRemover {
	e := &BallotBoxExpireRemover{
		srcBox:        srcBox,
		boxes:         boxes,
		prevBoxHashes: make(map[string]string),
		votingResults: votingResults,
		retain:        retain,
		stop:          make(chan chan struct{}),
	}

	return e
}

func (e *BallotBoxExpireRemover) Run() (err error) {
	ticker := time.NewTicker(e.retain)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.removeVotingResults()
			e.makePrevHashesFromSrcBox()
		case q := <-e.stop:
			close(q)
			return
		}
	}
}

func (e *BallotBoxExpireRemover) Stop() {
	q := make(chan struct{})
	e.stop <- q
	<-q
}

func (e *BallotBoxExpireRemover) removeVotingResults() {
	for _, hash := range e.prevBoxHashes {
		if e.srcBox.HasMessageByHash(hash) { // srcBox.HasHash(h)?
			if vr, ok := e.boxes.Results[hash]; ok {
				e.boxes.RemoveVotingResult(vr)
			}
		}
	}
}

func (e *BallotBoxExpireRemover) makePrevHashesFromSrcBox() {
	e.prevBoxHashes = make(map[string]string)

	for hash, _ := range e.srcBox.Hashes {
		if vr, ok := e.votingResults[hash]; ok {
			if byteArr, err := sebakcommon.MakeObjectHash(vr); err == nil {
				e.prevBoxHashes[string(byteArr)] = hash
			} else {
				log.Error("failed to make voting result hash", "VotingResult", vr, "error", err)
			}
		}
	}
}

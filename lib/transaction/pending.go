// Implement a type to hold and process pending operations
//
// Pending operations are recorded blockchain operations that
// have not yet altered the state of the node,
// but will trigger at a specific block height.
package transaction

import (
	"container/list"
	"fmt"
	"sync"
)

// The main type is a simple list with insert/pop primitives
type PendingPool struct {
	sync.RWMutex
	*list.List
}

// Unexported item type
// We construct it from a height/key pair,
// but the user only needs to get the key back
// as popping requires the height
type pendingPoolItem struct {
	// Block height at which this operation will take action
	height uint64
	// Key to look up the operation
	opKey string
}

// Instantiate a new `PendingPool`
func NewPendingPool() *PendingPool {
	return &PendingPool{sync.RWMutex{}, list.New()}
}

// Insert a new element in the pool
func (pp *PendingPool) Insert(height uint64, key string) {
	elem := &pendingPoolItem{height: height, opKey: key}

	pp.Lock()
	defer pp.Unlock()

	// Otherwise we need to insert this item in order
	for e := pp.Front(); e != nil; e = e.Next() {
		casted := e.Value.(*pendingPoolItem)
		// We can have multiple ops at the same height
		// Store them in the order they are inserted
		if casted.height > height {
			pp.InsertBefore(elem, e)
			return
		}
	}
	// Either empty, or we need to insert at the end
	pp.PushBack(elem)
}

// Peek an element from the list
//
// Elements are always stored in order, and should be popped in order.
// This means it's fine to `Insert` out of order
// (e.g. `Insert(42, "...")`, `Insert(25, "...")`,
// but when `Pop`ing, `25` should be `Pop`ed before `42`.
//
// Params:
//   height = The height being processed.
//            If the head of the list is under this height,
//            an empty string is returned.
//   offset = In the event that this list contains more than one
//            entry for a given height, this will skip `offset`
//            entries. As such, it should be 0 on the first call.
func (pp *PendingPool) Peek(height uint64, offset uint64) string {
	pp.RLock()
	defer pp.RUnlock()

	nOffset := offset
	for entry := pp.Front(); entry != nil; entry = entry.Next() {
		casted := entry.Value.(*pendingPoolItem)
		// We should never skip an element
		if casted.height < height {
			// This is a sanity check to ensure we don't attempt to pop
			// from the middle of the queue
			panic(fmt.Errorf("Attempt to get height %d but an item with height %d exists", height, casted.height))
		}

		// The delayed operation is still pending
		if height < casted.height {
			// But if we attempted to skip over it, it's a bug
			if nOffset != 0 {
				panic(fmt.Errorf("Invalid offset (%d/%d) used to seek over %d (expected %d)",
					nOffset, offset, casted.height, height))
			}
			return ""
		}
		// Since it's neither `<` not `>`, `height == casted.height`
		if nOffset == 0 {
			return casted.opKey
		}
		nOffset -= 1
	}
	return ""
}

// Pop any element at `height`
//
// Because we don't want to mutate the pool until all elements
// are processed, `Peek` does not remove any element.
// As a result a caller needs to do it manuall efter peeking
// all the elements from the height.
//
// Params:
//   height = The height being poped.
func (pp *PendingPool) PopHeight(height uint64) {
	pp.Lock()
	defer pp.Unlock()

	for entry := pp.Front(); entry != nil; entry = pp.Front() {
		casted := entry.Value.(*pendingPoolItem)
		if casted.height < height {
			panic(fmt.Errorf("Attempt to pop height %d but an item with height %d exists", height, casted.height))
		} else if height < casted.height {
			break
		}
		pp.Remove(entry)
	}
}

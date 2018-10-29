package sync

import (
	"context"
	"sync"
	"testing"
)

func TestPool(t *testing.T) {
	size := 2
	ctx := context.Background()
	pool := NewPool(uint64(size))
	var wg sync.WaitGroup

	wg.Add(size)

	for i := 0; i < size; i++ {
		pool.Add(ctx, func() {
			wg.Done()
		})
	}

	wg.Wait()
	pool.Finish()
}

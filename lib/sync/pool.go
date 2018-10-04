package sync

import (
	"context"
	"errors"
	"sync"
)

type Pool struct {
	finish chan struct{}
	work   chan<- func()
	wg     sync.WaitGroup
}

var ErrFinished = errors.New("Add: Pool is finished")

func NewPool(n int) *Pool {
	work := make(chan func())
	finish := make(chan struct{})
	p := &Pool{
		work:   work,
		finish: finish,
	}

	for ; n > 0; n-- {
		p.wg.Add(1)
		go func() {
			for {
				select {
				case f := <-work:
					f()
				case <-p.finish:
					p.wg.Done()
					return
				}
			}
		}()
	}
	return p
}

func (p *Pool) Add(ctx context.Context, f func()) error {
	select {
	case <-p.finish:
		return ErrFinished
	default:
	}

	select {
	case <-p.finish:
		return ErrFinished
	case p.work <- f:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) TryAdd(ctx context.Context, f func()) bool {
	select {
	case p.work <- f:
		return true
	default:
		return false
	}
}

func (p *Pool) Finish() {
	select {
	case <-p.finish:
	default:
		close(p.finish)
	}
	p.wg.Wait()
}

package workers

import "sync"

// Pool is a bounded worker pool using a semaphore pattern.
// Submit blocks when all workers are busy, providing backpressure.
type Pool struct {
	sem chan struct{}
	wg  sync.WaitGroup
}

// NewPool creates a worker pool with the given concurrency limit.
func NewPool(size int) *Pool {
	return &Pool{sem: make(chan struct{}, size)}
}

// Submit blocks until a worker slot is available, then runs fn in a goroutine.
func (p *Pool) Submit(fn func()) {
	p.sem <- struct{}{} // blocks if pool is full
	p.wg.Add(1)
	go func() {
		defer func() {
			<-p.sem
			p.wg.Done()
		}()
		fn()
	}()
}

// Shutdown waits for all in-flight work to complete.
func (p *Pool) Shutdown() {
	p.wg.Wait()
}

package workers

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_ConcurrencyLimit(t *testing.T) {
	poolSize := 3
	pool := NewPool(poolSize)

	var running atomic.Int32
	var maxRunning atomic.Int32

	done := make(chan struct{})

	for i := 0; i < 10; i++ {
		pool.Submit(func() {
			cur := running.Add(1)
			// Track max concurrent
			for {
				old := maxRunning.Load()
				if cur <= old || maxRunning.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			running.Add(-1)
		})
	}

	go func() {
		pool.Shutdown()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("pool shutdown timed out")
	}

	if max := maxRunning.Load(); max > int32(poolSize) {
		t.Errorf("max concurrent workers = %d, want <= %d", max, poolSize)
	}
	if max := maxRunning.Load(); max < int32(poolSize) {
		t.Errorf("max concurrent workers = %d, want %d (pool underutilized)", max, poolSize)
	}
}

func TestPool_Shutdown_WaitsForWork(t *testing.T) {
	pool := NewPool(2)

	var completed atomic.Int32

	for i := 0; i < 5; i++ {
		pool.Submit(func() {
			time.Sleep(20 * time.Millisecond)
			completed.Add(1)
		})
	}

	pool.Shutdown()

	if c := completed.Load(); c != 5 {
		t.Errorf("completed = %d, want 5", c)
	}
}

func TestPool_Submit_BlocksWhenFull(t *testing.T) {
	pool := NewPool(1)

	started := make(chan struct{})
	release := make(chan struct{})

	// Fill the pool
	pool.Submit(func() {
		close(started)
		<-release
	})

	<-started

	// This should block because the pool is full
	blocked := make(chan struct{})
	go func() {
		pool.Submit(func() {})
		close(blocked)
	}()

	select {
	case <-blocked:
		t.Fatal("Submit should block when pool is full")
	case <-time.After(100 * time.Millisecond):
		// Expected: Submit is blocking
	}

	// Release the first task
	close(release)

	select {
	case <-blocked:
		// Expected: Submit unblocks
	case <-time.After(time.Second):
		t.Fatal("Submit should unblock after worker completes")
	}

	pool.Shutdown()
}

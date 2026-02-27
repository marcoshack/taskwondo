package workers

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestScheduler_RunsImmediately(t *testing.T) {
	s := NewScheduler(zerolog.Nop())

	var called atomic.Int32

	s.Add(PeriodicTask{
		Name:     "test",
		Interval: time.Hour, // long interval — only immediate run matters
		Fn: func(_ context.Context) error {
			called.Add(1)
			return nil
		},
	})

	s.Start(context.Background())
	time.Sleep(100 * time.Millisecond)
	s.Shutdown()

	if c := called.Load(); c < 1 {
		t.Errorf("task called %d times, want >= 1", c)
	}
}

func TestScheduler_RunsPeriodically(t *testing.T) {
	s := NewScheduler(zerolog.Nop())

	var called atomic.Int32

	s.Add(PeriodicTask{
		Name:     "test",
		Interval: 20 * time.Millisecond,
		Fn: func(_ context.Context) error {
			called.Add(1)
			return nil
		},
	})

	s.Start(context.Background())
	time.Sleep(150 * time.Millisecond)
	s.Shutdown()

	// Should have run at least 3 times: immediate + ~6 ticks
	if c := called.Load(); c < 3 {
		t.Errorf("task called %d times, want >= 3", c)
	}
}

func TestScheduler_ShutdownStopsTasks(t *testing.T) {
	s := NewScheduler(zerolog.Nop())

	var called atomic.Int32

	s.Add(PeriodicTask{
		Name:     "test",
		Interval: 10 * time.Millisecond,
		Fn: func(_ context.Context) error {
			called.Add(1)
			return nil
		},
	})

	s.Start(context.Background())
	time.Sleep(50 * time.Millisecond)
	s.Shutdown()

	countAfterShutdown := called.Load()
	time.Sleep(50 * time.Millisecond)
	countLater := called.Load()

	if countLater != countAfterShutdown {
		t.Errorf("task still running after shutdown: %d -> %d", countAfterShutdown, countLater)
	}
}

func TestScheduler_ErrorDoesNotStop(t *testing.T) {
	s := NewScheduler(zerolog.Nop())

	var called atomic.Int32

	s.Add(PeriodicTask{
		Name:     "failing",
		Interval: 20 * time.Millisecond,
		Fn: func(_ context.Context) error {
			called.Add(1)
			return fmt.Errorf("intentional error")
		},
	})

	s.Start(context.Background())
	time.Sleep(100 * time.Millisecond)
	s.Shutdown()

	if c := called.Load(); c < 3 {
		t.Errorf("task called %d times, want >= 3 (errors should not stop the loop)", c)
	}
}

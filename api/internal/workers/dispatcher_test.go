package workers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

// startTestNATS starts an in-process NATS server with JetStream enabled.
func startTestNATS(t *testing.T) (*server.Server, string) {
	t.Helper()
	opts := &server.Options{
		Host:      "127.0.0.1",
		Port:      -1, // random port
		JetStream: true,
		StoreDir:  t.TempDir(),
		NoLog:     true,
	}
	srv, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("creating nats server: %v", err)
	}
	srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats server not ready")
	}
	return srv, srv.ClientURL()
}

type testTask struct {
	name     string
	executed atomic.Int32
	payload  atomic.Value
	err      error
}

func (t *testTask) Name() string { return t.name }
func (t *testTask) Execute(_ context.Context, payload []byte) error {
	t.executed.Add(1)
	t.payload.Store(payload)
	return t.err
}

func TestDispatcher_RegisterAndProcess(t *testing.T) {
	srv, url := startTestNATS(t)
	defer srv.Shutdown()

	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connecting to nats: %v", err)
	}
	defer nc.Close()

	pool := NewPool(2)
	dispatcher, err := NewDispatcher(nc, pool, zerolog.Nop())
	if err != nil {
		t.Fatalf("creating dispatcher: %v", err)
	}

	task := &testTask{name: "test.echo"}
	dispatcher.Register(task)

	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("starting dispatcher: %v", err)
	}

	// Publish a message
	js, _ := nc.JetStream()
	_, err = js.Publish("taskwondo.test.echo", []byte("hello"))
	if err != nil {
		t.Fatalf("publishing message: %v", err)
	}

	// Wait for processing
	deadline := time.After(3 * time.Second)
	for {
		if task.executed.Load() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("task was not executed within timeout")
		case <-time.After(10 * time.Millisecond):
		}
	}

	if got := task.payload.Load().([]byte); string(got) != "hello" {
		t.Errorf("payload = %q, want %q", string(got), "hello")
	}

	dispatcher.Shutdown(context.Background())
}

func TestDispatcher_AckOnSuccess(t *testing.T) {
	srv, url := startTestNATS(t)
	defer srv.Shutdown()

	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connecting to nats: %v", err)
	}
	defer nc.Close()

	pool := NewPool(2)
	dispatcher, err := NewDispatcher(nc, pool, zerolog.Nop())
	if err != nil {
		t.Fatalf("creating dispatcher: %v", err)
	}

	task := &testTask{name: "test.ack"}
	dispatcher.Register(task)

	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("starting dispatcher: %v", err)
	}

	js, _ := nc.JetStream()
	_, err = js.Publish("taskwondo.test.ack", []byte("data"))
	if err != nil {
		t.Fatalf("publishing message: %v", err)
	}

	// Wait for processing
	deadline := time.After(3 * time.Second)
	for {
		if task.executed.Load() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("task was not executed within timeout")
		case <-time.After(10 * time.Millisecond):
		}
	}

	// Give time for ack to propagate
	time.Sleep(100 * time.Millisecond)

	// Check stream info — pending should be 0 after successful ack
	info, err := js.StreamInfo(streamName)
	if err != nil {
		t.Fatalf("getting stream info: %v", err)
	}
	// With WorkQueue policy, acked messages are removed from the stream
	if info.State.Msgs != 0 {
		t.Errorf("stream messages = %d, want 0 (message should be acked and removed)", info.State.Msgs)
	}

	dispatcher.Shutdown(context.Background())
}

func TestDispatcher_Shutdown(t *testing.T) {
	srv, url := startTestNATS(t)
	defer srv.Shutdown()

	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connecting to nats: %v", err)
	}
	defer nc.Close()

	pool := NewPool(2)
	dispatcher, err := NewDispatcher(nc, pool, zerolog.Nop())
	if err != nil {
		t.Fatalf("creating dispatcher: %v", err)
	}

	task := &testTask{name: "test.shutdown"}
	dispatcher.Register(task)

	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("starting dispatcher: %v", err)
	}

	// Shutdown should complete without hanging
	done := make(chan struct{})
	go func() {
		dispatcher.Shutdown(context.Background())
		close(done)
	}()

	select {
	case <-done:
		// Expected
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown timed out")
	}
}

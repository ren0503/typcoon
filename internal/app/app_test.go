package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/platform/config"
)

type fakeDB struct {
	mu     sync.Mutex
	order  *[]string
	closed bool
}

func (f *fakeDB) Ping(context.Context) error {
	return nil
}

func (f *fakeDB) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	*f.order = append(*f.order, "db")
}

type fakeServer struct {
	mu       sync.Mutex
	order    *[]string
	stop     chan struct{}
	started  chan struct{}
	startOne sync.Once
	startErr error
}

func (s *fakeServer) Start() error {
	s.startOne.Do(func() { close(s.started) })
	if s.startErr != nil {
		return s.startErr
	}
	<-s.stop
	return nil
}

func (s *fakeServer) Shutdown(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	*s.order = append(*s.order, "server")
	close(s.stop)
	return nil
}

func (s *fakeServer) Addr() string {
	return "fake:0"
}

type fakeMonitor struct {
	err error
}

func (m *fakeMonitor) Run(ctx context.Context) error {
	if m.err != nil {
		return m.err
	}
	<-ctx.Done()
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestApp(order *[]string, server *fakeServer, monitor Monitor) *App {
	return New(
		&config.Config{HTTP: config.HTTPConfig{ShutdownTimeout: time.Second}},
		testLogger(),
		&fakeDB{order: order},
		server,
		monitor,
	)
}

func waitForRun(t *testing.T, errCh <-chan error) error {
	t.Helper()
	select {
	case err := <-errCh:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("application did not stop in time")
		return nil
	}
}

func TestRunShutsDownOnContextCancel(t *testing.T) {
	order := []string{}
	server := &fakeServer{order: &order, stop: make(chan struct{}), started: make(chan struct{})}
	application := newTestApp(&order, server, &fakeMonitor{})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- application.Run(ctx) }()

	<-server.started
	cancel()

	if err := waitForRun(t, errCh); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !reflect.DeepEqual(order, []string{"server", "db"}) {
		t.Fatalf("shutdown order = %v", order)
	}
}

func TestRunShutsDownWhenMonitorFails(t *testing.T) {
	order := []string{}
	server := &fakeServer{order: &order, stop: make(chan struct{}), started: make(chan struct{})}
	application := newTestApp(&order, server, &fakeMonitor{err: errors.New("database unavailable")})

	errCh := make(chan error, 1)
	go func() { errCh <- application.Run(context.Background()) }()

	<-server.started
	err := waitForRun(t, errCh)
	if err == nil {
		t.Fatal("Run() expected an error")
	}
	if !strings.Contains(err.Error(), "database health monitor failed") {
		t.Fatalf("error = %v", err)
	}
	if !reflect.DeepEqual(order, []string{"server", "db"}) {
		t.Fatalf("shutdown order = %v", order)
	}
}

func TestRunShutsDownWhenServerFails(t *testing.T) {
	order := []string{}
	server := &fakeServer{
		order:    &order,
		stop:     make(chan struct{}),
		started:  make(chan struct{}),
		startErr: errors.New("listen failed"),
	}
	application := newTestApp(&order, server, nil)

	err := application.Run(context.Background())
	if err == nil {
		t.Fatal("Run() expected an error")
	}
	if !strings.Contains(err.Error(), "http server failed") {
		t.Fatalf("error = %v", err)
	}
	if !reflect.DeepEqual(order, []string{"server", "db"}) {
		t.Fatalf("shutdown order = %v", order)
	}
}

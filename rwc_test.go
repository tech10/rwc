package rwc_test

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/tech10/rwc"
)

// mockRWC wraps a bytes.Buffer to implement io.ReadWriteCloser.
type mockRWC struct {
	buf    *bytes.Buffer
	delay  time.Duration
	closed bool
	mu     sync.Mutex
}

func newMockRWC() *mockRWC {
	return &mockRWC{buf: &bytes.Buffer{}}
}

func (m *mockRWC) Read(p []byte) (int, error) {
	m.waitDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.buf.Read(p)
}

func (m *mockRWC) Write(p []byte) (int, error) {
	m.waitDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.buf.Write(p)
}

func (m *mockRWC) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockRWC) waitDelay() {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
}

func TestNewResReadWriteCloser_PanicOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when passing nil to NewResReadWriteCloser")
		}
	}()
	_ = rwc.NewResReadWriteCloser(nil)
}

func TestReadWriteBasic(t *testing.T) {
	m := newMockRWC()
	r := rwc.NewResReadWriteCloser(m)

	data := []byte("hello")
	n, err := r.Write(data)
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected %d bytes written, got %d", len(data), n)
	}

	readBuf := make([]byte, len(data))
	n, err = r.Read(readBuf)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("unexpected read error: %v", err)
	}
	if n != len(data) || string(readBuf) != "hello" {
		t.Fatalf("unexpected read result: %s", readBuf)
	}
}

func TestResetErrors(t *testing.T) {
	m1 := newMockRWC()
	m2 := newMockRWC()
	r := rwc.NewResReadWriteCloser(m1)

	// Reset with nil
	if err := r.Reset(nil, true); !errors.Is(err, rwc.ErrResetNil) {
		t.Fatalf("expected ErrResetNil, got %v", err)
	}

	// Reset with same RWC
	if err := r.Reset(m1, true); !errors.Is(err, rwc.ErrEqual) {
		t.Fatalf("expected ErrEqual, got %v", err)
	}

	// Reset with same ResReadWriteCloser
	if err := r.Reset(r, true); !errors.Is(err, rwc.ErrEqualToSelf) {
		t.Fatalf("expected ErrEqualToSelf, got %v", err)
	}

	// Reset with new RWC
	if err := r.Reset(m2, true); err != nil {
		t.Fatalf("unexpected error on valid reset: %v", err)
	}

	if r.ResetCount() != 1 {
		t.Fatalf("expected ResetCount 1, got %d", r.ResetCount())
	}
}

func TestResetCloseOld(t *testing.T) {
	m1 := newMockRWC()
	m2 := newMockRWC()
	r := rwc.NewResReadWriteCloser(m1)

	if err := r.Reset(m2, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// m1 should be closed
	data := []byte("test")
	_, err := m1.Write(data)
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected m1 to be closed, got err: %v", err)
	}

	// m2 should work
	n, err := m2.Write(data)
	if err != nil || n != len(data) {
		t.Fatalf("m2 write failed: %v", err)
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	m := newMockRWC()
	r := rwc.NewResReadWriteCloser(m)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			data := []byte{byte(i)}
			_, _ = r.Write(data)
		}(i)
		go func() {
			defer wg.Done()
			buf := make([]byte, 1)
			_, _ = r.Read(buf)
		}()
	}
	wg.Wait()
	t.Log("No deadlock.")
}

func TestClose(t *testing.T) {
	m := newMockRWC()
	r := rwc.NewResReadWriteCloser(m)

	if err := r.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}

	// Underlying RWC should be closed
	// and its error should be passed on
	_, err := r.Write([]byte("x"))
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected underlying RWC closed, got: %v", err)
	}
}

func TestDelayedReadWriteDuringReset(t *testing.T) {
	m := newMockRWC()
	m.delay = 50 * time.Millisecond
	r := rwc.NewResReadWriteCloser(m)

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer goroutine
	go func() {
		defer wg.Done()
		data := []byte("test")
		n, err := r.Write(data)
		if !errors.Is(err, rwc.ErrRWCReset) {
			t.Errorf("expected write error %v, got %v", rwc.ErrRWCReset, err)
		}
		if num := len(data); n != num {
			t.Errorf("Expected written bytes %d, got %d", num, n)
		}
	}()

	// Reader goroutine
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * 4)
		buf := make([]byte, 4)
		n, err := r.Read(buf)
		if !errors.Is(err, rwc.ErrRWCReset) {
			t.Errorf("expected read error %v, got %v", rwc.ErrRWCReset, err)
		}
		if n != 4 {
			t.Errorf("Expected length 4, got %d", n)
		}
		if s := string(buf); s != "test" {
			t.Errorf("Expected from read: \"test\", got \"%s\"", s)
		}
	}()

	// Ensure goroutines have started and hit delay
	time.Sleep(20 * time.Millisecond)

	// Reset while operations are still pending, do not close the previous ReadWriteCloser
	// so full functionality can be tested properly.
	if err := r.Reset(newMockRWC(), false); err != nil {
		t.Errorf("unexpected reset error: %v", err)
	}

	wg.Wait()
}

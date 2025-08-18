// Package rwc provides a resettable ReadWriteCloser, which can change its ReadWriteCloser via a call to the Reset function.
// Ideally, this should only be done when there are no reads or writes occurring,
// but it should be thread safe with proper use of a sync.Mutex and an atomic counter.
package rwc

import (
	"io"
	"sync"

	"github.com/tech10/rwc/atomic"
)

// ResReadWriteCloser is a ReadWriteCloser who's io.ReadWriteCloser can be safely reset.
// It implements the io.ReadWriteCloser interface.
//
// If its io.ReadWriteCloser is reset during a read or write operation,
// the read or write operation will complete on the old ReadWriteCloser unless the old ReadWriteCloser is closed.
//
// If the io.ReadWriteCloser is reset during a read or write call,
// ErrRWCReset is returned along with the number of bytes read or written by the old reader.
// Callers of Read or Write should check for the presence of ErrRWCReset,
// and if desired, try their calls again.
// Callers should also check the number of bytes read or written
// on the old io.ReadWriteCloser after the reset occurred.
type ResReadWriteCloser struct {
	mu    sync.RWMutex
	rwc   io.ReadWriteCloser
	count atomic.Uint64 // increments on every reset
}

// NewResReadWriteCloser creates a resettable ReadWriteCloser implementing the io.ReadWriteCloser interface.
// If you attempt to create it with a nil value, a runtime panic will occur.
// This is done as soon as possible, as the io.ReadWriteCloser used should never be nil,
// or it will panic during any calls to Read, Write, or Close.
func NewResReadWriteCloser(rwc io.ReadWriteCloser) *ResReadWriteCloser {
	if rwc == nil {
		panic("ResReadWriteCloser: nil io.ReadWriteCloser not permitted")
	}
	return &ResReadWriteCloser{rwc: rwc}
}

// Read implements the io.Reader interface.
// If the ReadWriteCloser is reset during a read,
// ErrRWCReset is returned along with the number of bytes read from the previous ReadWriteCloser.
// Any error returned by the io.ReadWriteCloser after the reset
// is replaced with ErrRWCReset.
// If it is reset before the read takes place,
// 0 is returned along with ErrRWCReset.
func (r *ResReadWriteCloser) Read(p []byte) (int, error) {
	startCount := r.count.Load()

	r.mu.RLock()
	reader := r.rwc
	r.mu.RUnlock()

	// Detect reset before starting
	if startCount != r.count.Load() {
		return 0, ErrRWCReset
	}

	n, err := reader.Read(p)

	// Detect reset after read
	if startCount != r.count.Load() {
		return n, ErrRWCReset
	}

	return n, err
}

// Write implements the io.Writer interface.
// If the ReadWriteCloser is reset during a write,
// ErrRWCReset is returned along with the number of bytes written to the previous ReadWriteCloser.
// Any error returned by the io.ReadWriteCloser after the reset
// is replaced with ErrRWCReset.
// If it is reset before the write takes place,
// 0 is returned along with ErrRWCReset.
func (r *ResReadWriteCloser) Write(p []byte) (int, error) {
	startCount := r.count.Load()

	r.mu.RLock()
	writer := r.rwc
	r.mu.RUnlock()

	// Detect reset before starting
	if startCount != r.count.Load() {
		return 0, ErrRWCReset
	}

	n, err := writer.Write(p)

	// Detect reset after write
	if startCount != r.count.Load() {
		return n, ErrRWCReset
	}

	return n, err
}

// Close implements the io.Closer interface.
// It closes the io.ReadWriteCloser assigned under a read mutex lock
// to ensure a reset cannot occur until all read locks are released.
// Whether or not to use the ResReadWriteCloser after close is left up to the caller.
// It can be reset after Close is called.
func (r *ResReadWriteCloser) Close() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rwc.Close()
}

// Reset will allow you to reset the current io.ReadWriteCloser
// with a new io.ReadWriteCloser.
// Ideally, this should only be done when there are no reads or writes occurring.
//
// You cannot reset it with nil, ErrResetNil will be returned.
// You cannot reset it with a value equal to the existing io.ReadWriteCloser interface,
// ErrEqual will be returned.
// You cannot reset it to its own ResReadWriteCloser,
// ErrEqualToSelf will be returned.
//
// If you set closeOld to true, the old io.ReadWriteCloser will be closed during the reset.
// Setting closeOld to false could prove useful in certain situations,
// such as resetting the ResReadWriteCloser with a custom ReadWriteCloser implementation
// wrapping the one you originally used on creation.
func (r *ResReadWriteCloser) Reset(newRWC io.ReadWriteCloser, closeOld bool) error {
	switch newRWC {
	case nil:
		return ErrResetNil
	case r:
		return ErrEqualToSelf
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.rwc == newRWC {
		return ErrEqual
	}

	old := r.rwc
	r.rwc = newRWC
	r.count.Add(1) // increment generation

	if closeOld {
		_ = old.Close()
	}

	return nil
}

// ResetCount returns the number of times the ResReadWriteCloser has been reset.
// This could be useful for debugging, testing, or if you chose to set up your own limits for resets.
func (r *ResReadWriteCloser) ResetCount() uint64 {
	return r.count.Load()
}

// RWC returns the underlying io.ReadWriteCloser.
// If you initialized the ResReadWriteCloser with something like a net.Conn or os.File,
// you can retrieve the original value via type assertion.
func (r *ResReadWriteCloser) RWC() io.ReadWriteCloser {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rwc
}

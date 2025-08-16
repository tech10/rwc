package rwc

import "errors"

var (
	// ErrEqual is returned if attempting to reset ResReadWriteCloser
	// with a value equal to that which already exists.
	ErrEqual = errors.New("new ReadWriteCloser equal to old ReadWriteCloser")
	// ErrEqualToSelf is returned if attempting to reset ResReadWriteCloser
	// with a value equal to itself.
	ErrEqualToSelf = errors.New("new ReadWriteCloser equal to current ResReadWriteCloser")
	// ErrResetNil is returned if attempting to reset ResReadWriteCloser with a nil value.
	ErrResetNil = errors.New("nil reset not permitted")
	// ErrRWCReset is returned by Read and Write functions on ResReadWriteCloser
	// if it is reset during a read or write.
	ErrRWCReset = errors.New("ReadWriteCloser reset")
)

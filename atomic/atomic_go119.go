//go:build go1.19
// +build go1.19

package atomic

import "sync/atomic"

// Uint64 is re-exported from sync/atomic for Go 1.19+.
type Uint64 = atomic.Uint64

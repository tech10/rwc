//go:build !go1.19
// +build !go1.19

package atomic

import "sync/atomic"

// Uint64 is a drop-in replacement for Go < 1.19
type Uint64 struct {
	v uint64
}

func (u *Uint64) Load() uint64 {
	return atomic.LoadUint64(&u.v)
}

func (u *Uint64) Store(val uint64) {
	atomic.StoreUint64(&u.v, val)
}

func (u *Uint64) Add(delta uint64) uint64 {
	return atomic.AddUint64(&u.v, delta)
}

func (u *Uint64) Swap(new uint64) uint64 {
	return atomic.SwapUint64(&u.v, new)
}

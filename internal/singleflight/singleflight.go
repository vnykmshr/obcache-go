package singleflight

import (
	"context"
	"sync"
)

// Group represents a class of work and forms a namespace in which
// units of work can be executed with duplicate suppression.
type Group[K comparable, V any] struct {
	mu sync.Mutex
	m  map[K]*call[V]
}

// call is an in-flight or completed singleflight.Do call
type call[V any] struct {
	wg sync.WaitGroup

	// These fields are written once before the WaitGroup is done
	// and are only read after the WaitGroup is done.
	val V
	err error

	// These fields are read and written with the Group's mutex held.
	dups  int
	chans []chan<- Result[V]
}

// Result holds the results of Do, so they can be passed
// on a channel.
type Result[V any] struct {
	Val    V
	Err    error
	Shared bool
}

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
// The return value shared indicates whether v was given to multiple callers.
func (g *Group[K, V]) Do(key K, fn func() (V, error)) (v V, err error, shared bool) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[K]*call[V])
	}
	if c, ok := g.m[key]; ok {
		c.dups++
		g.mu.Unlock()
		c.wg.Wait()

		return c.val, c.err, true
	}
	c := new(call[V])
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	g.doCall(c, key, fn)
	return c.val, c.err, c.dups > 0
}

// DoChan is like Do but returns a channel that will receive the
// results when they are ready.
//
// The returned channel will not be closed.
func (g *Group[K, V]) DoChan(key K, fn func() (V, error)) <-chan Result[V] {
	ch := make(chan Result[V], 1)
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[K]*call[V])
	}
	if c, ok := g.m[key]; ok {
		c.dups++
		c.chans = append(c.chans, ch)
		g.mu.Unlock()
		return ch
	}
	c := &call[V]{chans: []chan<- Result[V]{ch}}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	go g.doCall(c, key, fn)

	return ch
}

// doCall handles the single call for a key.
func (g *Group[K, V]) doCall(c *call[V], key K, fn func() (V, error)) {
	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	for _, ch := range c.chans {
		ch <- Result[V]{c.val, c.err, c.dups > 0}
	}
	g.mu.Unlock()
}

// Forget tells the singleflight to forget about a key.  Future calls
// to Do for this key will call the function rather than waiting for
// an earlier call to complete.
func (g *Group[K, V]) Forget(key K) {
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
}

// DoContext is like Do but respects context cancellation.
// If the context is cancelled, it returns the context error immediately.
// The underlying function will still complete, but this caller won't wait for it.
func (g *Group[K, V]) DoContext(ctx context.Context, key K, fn func() (V, error)) (v V, err error, shared bool) {
	if err := ctx.Err(); err != nil {
		return v, err, false
	}

	ch := g.DoChan(key, fn)
	select {
	case <-ctx.Done():
		return v, ctx.Err(), false
	case result := <-ch:
		return result.Val, result.Err, result.Shared
	}
}

// InFlight returns the number of keys currently being processed.
func (g *Group[K, V]) InFlight() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.m)
}

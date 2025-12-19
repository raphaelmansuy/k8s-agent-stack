/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package coalesce provides request coalescing to deduplicate concurrent identical requests.
package coalesce

import (
	"context"
	"sync"
	"time"
)

// Coalescer deduplicates concurrent identical requests.
// When multiple goroutines request the same key simultaneously,
// only one will execute the function and all will receive the same result.
type Coalescer[K comparable, V any] struct {
	mu       sync.Mutex
	inflight map[K]*call[V]
	timeout  time.Duration
}

type call[V any] struct {
	wg     sync.WaitGroup
	result V
	err    error
}

// New creates a new Coalescer with the specified cleanup timeout.
func New[K comparable, V any](timeout time.Duration) *Coalescer[K, V] {
	if timeout == 0 {
		timeout = 100 * time.Millisecond
	}
	return &Coalescer[K, V]{
		inflight: make(map[K]*call[V]),
		timeout:  timeout,
	}
}

// Do executes the function, coalescing concurrent calls with the same key.
// If there's already an inflight request for the key, it waits for that result.
func (c *Coalescer[K, V]) Do(ctx context.Context, key K, fn func() (V, error)) (V, error) {
	c.mu.Lock()

	// Check if there's already an inflight request
	if existing, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		existing.wg.Wait()
		return existing.result, existing.err
	}

	// Create new call
	call := &call[V]{}
	call.wg.Add(1)
	c.inflight[key] = call
	c.mu.Unlock()

	// Execute the function
	call.result, call.err = fn()
	call.wg.Done()

	// Cleanup after timeout to catch stragglers
	go func() {
		time.Sleep(c.timeout)
		c.mu.Lock()
		delete(c.inflight, key)
		c.mu.Unlock()
	}()

	return call.result, call.err
}

// DoWithContext executes with context cancellation support.
func (c *Coalescer[K, V]) DoWithContext(ctx context.Context, key K, fn func(context.Context) (V, error)) (V, error) {
	c.mu.Lock()

	// Check if there's already an inflight request
	if existing, ok := c.inflight[key]; ok {
		c.mu.Unlock()

		// Wait with context cancellation
		done := make(chan struct{})
		go func() {
			existing.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			return existing.result, existing.err
		case <-ctx.Done():
			var zero V
			return zero, ctx.Err()
		}
	}

	// Create new call
	call := &call[V]{}
	call.wg.Add(1)
	c.inflight[key] = call
	c.mu.Unlock()

	// Execute with context
	call.result, call.err = fn(ctx)
	call.wg.Done()

	// Cleanup after timeout
	go func() {
		time.Sleep(c.timeout)
		c.mu.Lock()
		delete(c.inflight, key)
		c.mu.Unlock()
	}()

	return call.result, call.err
}

// BatchCoalescer batches multiple requests together.
type BatchCoalescer[K comparable, V any] struct {
	mu        sync.Mutex
	pending   map[K][]chan result[V]
	batchSize int
	delay     time.Duration
	executor  func(keys []K) (map[K]V, error)
	timer     *time.Timer
}

type result[V any] struct {
	value V
	err   error
}

// NewBatchCoalescer creates a new batch coalescer.
func NewBatchCoalescer[K comparable, V any](
	batchSize int,
	delay time.Duration,
	executor func(keys []K) (map[K]V, error),
) *BatchCoalescer[K, V] {
	return &BatchCoalescer[K, V]{
		pending:   make(map[K][]chan result[V]),
		batchSize: batchSize,
		delay:     delay,
		executor:  executor,
	}
}

// Get retrieves a value, potentially batching with other concurrent requests.
func (b *BatchCoalescer[K, V]) Get(ctx context.Context, key K) (V, error) {
	ch := make(chan result[V], 1)

	b.mu.Lock()
	b.pending[key] = append(b.pending[key], ch)
	pendingCount := len(b.pending)

	// Execute batch if size threshold reached
	if pendingCount >= b.batchSize {
		b.executeBatch()
	} else if b.timer == nil {
		// Start timer for delay-based execution
		b.timer = time.AfterFunc(b.delay, func() {
			b.mu.Lock()
			b.executeBatch()
			b.mu.Unlock()
		})
	}
	b.mu.Unlock()

	// Wait for result with context cancellation
	select {
	case r := <-ch:
		return r.value, r.err
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	}
}

func (b *BatchCoalescer[K, V]) executeBatch() {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	if len(b.pending) == 0 {
		return
	}

	// Collect keys
	keys := make([]K, 0, len(b.pending))
	pending := make(map[K][]chan result[V])
	for k, chs := range b.pending {
		keys = append(keys, k)
		pending[k] = chs
	}
	b.pending = make(map[K][]chan result[V])

	// Execute in goroutine to not block
	go func() {
		results, err := b.executor(keys)

		for k, chs := range pending {
			r := result[V]{err: err}
			if err == nil {
				if v, ok := results[k]; ok {
					r.value = v
				} else {
					r.err = ErrNotFound
				}
			}
			for _, ch := range chs {
				ch <- r
				close(ch)
			}
		}
	}()
}

// ErrNotFound is returned when a key is not found in batch results.
var ErrNotFound = &NotFoundError{}

// NotFoundError indicates a key was not found.
type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "key not found in batch results"
}

// SingleFlight provides a simpler interface for single-flight deduplication.
type SingleFlight struct {
	mu sync.Mutex
	m  map[string]*singleFlightCall
}

type singleFlightCall struct {
	wg  sync.WaitGroup
	val any
	err error
}

// NewSingleFlight creates a new single flight instance.
func NewSingleFlight() *SingleFlight {
	return &SingleFlight{
		m: make(map[string]*singleFlightCall),
	}
}

// Do executes and returns the results of the given function, deduplicating
// concurrent calls with the same key.
func (g *SingleFlight) Do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(singleFlightCall)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}

// Forget removes a key from the in-flight map.
func (g *SingleFlight) Forget(key string) {
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
}

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

package coalesce

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCoalescer_SingleRequest(t *testing.T) {
	callCount := int32(0)
	c := New[string, string](100 * time.Millisecond)

	result, err := c.Do(context.Background(), "key1", func() (string, error) {
		atomic.AddInt32(&callCount, 1)
		return "result-key1", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result-key1" {
		t.Errorf("expected 'result-key1', got '%s'", result)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestCoalescer_ConcurrentSameKey(t *testing.T) {
	callCount := int32(0)
	c := New[string, string](100 * time.Millisecond)

	var wg sync.WaitGroup
	results := make([]string, 10)
	errs := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = c.Do(context.Background(), "same-key", func() (string, error) {
				atomic.AddInt32(&callCount, 1)
				time.Sleep(50 * time.Millisecond)
				return "result-same-key", nil
			})
		}(i)
	}

	wg.Wait()

	// All should succeed with same result
	for i := 0; i < 10; i++ {
		if errs[i] != nil {
			t.Errorf("request %d failed: %v", i, errs[i])
		}
		if results[i] != "result-same-key" {
			t.Errorf("request %d got wrong result: %s", i, results[i])
		}
	}

	// Only one function call should have been made
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestCoalescer_DifferentKeys(t *testing.T) {
	callCount := int32(0)
	c := New[string, string](100 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := string(rune('a' + idx))
			_, _ = c.Do(context.Background(), key, func() (string, error) {
				atomic.AddInt32(&callCount, 1)
				time.Sleep(20 * time.Millisecond)
				return "result", nil
			})
		}(i)
	}

	wg.Wait()

	// Each different key should trigger a call
	if callCount != 5 {
		t.Errorf("expected 5 calls, got %d", callCount)
	}
}

func TestCoalescer_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("test error")
	c := New[string, string](100 * time.Millisecond)

	var wg sync.WaitGroup
	errs := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = c.Do(context.Background(), "key", func() (string, error) {
				time.Sleep(20 * time.Millisecond)
				return "", expectedErr
			})
		}(i)
	}

	wg.Wait()

	// All should get the same error
	for i := 0; i < 5; i++ {
		if !errors.Is(errs[i], expectedErr) {
			t.Errorf("request %d: expected error %v, got %v", i, expectedErr, errs[i])
		}
	}
}

func TestCoalescer_ContextCancellation(t *testing.T) {
	started := make(chan struct{})
	continueExec := make(chan struct{})
	c := New[string, string](100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	var err error
	go func() {
		defer wg.Done()
		_, err = c.DoWithContext(ctx, "key", func(ctx context.Context) (string, error) {
			close(started)
			// Wait until told to continue or context is cancelled
			select {
			case <-continueExec:
				return "result", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		})
	}()

	// Wait for the function to start
	<-started

	// Cancel the context
	cancel()

	wg.Wait()

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestBatchCoalescer(t *testing.T) {
	callCount := int32(0)
	var receivedKeys []string
	var mu sync.Mutex

	bc := NewBatchCoalescer[string, string](5, 10*time.Millisecond, func(keys []string) (map[string]string, error) {
		atomic.AddInt32(&callCount, 1)
		mu.Lock()
		receivedKeys = append(receivedKeys, keys...)
		mu.Unlock()

		result := make(map[string]string)
		for _, k := range keys {
			result[k] = "result-" + k
		}
		return result, nil
	})

	var wg sync.WaitGroup
	results := make([]string, 10)
	errs := make([]error, 10)

	// Submit 10 requests quickly
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := string(rune('a' + idx))
			results[idx], errs[idx] = bc.Get(context.Background(), key)
		}(i)
	}

	wg.Wait()

	// Check all succeeded
	for i := 0; i < 10; i++ {
		if errs[i] != nil {
			t.Errorf("request %d failed: %v", i, errs[i])
		}
		expected := "result-" + string(rune('a'+i))
		if results[i] != expected {
			t.Errorf("request %d: expected '%s', got '%s'", i, expected, results[i])
		}
	}

	// Should have batched into 2 calls (batch size 5)
	if callCount != 2 {
		t.Errorf("expected 2 batch calls, got %d", callCount)
	}
}

func TestBatchCoalescer_Timeout(t *testing.T) {
	callCount := int32(0)
	bc := NewBatchCoalescer[string, string](100, 50*time.Millisecond, func(keys []string) (map[string]string, error) {
		atomic.AddInt32(&callCount, 1)
		result := make(map[string]string)
		for _, k := range keys {
			result[k] = "result-" + k
		}
		return result, nil
	}) // Large batch size, short timeout

	// Submit just one request
	result, err := bc.Get(context.Background(), "key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result-key1" {
		t.Errorf("expected 'result-key1', got '%s'", result)
	}

	// Should have triggered due to timeout
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestSingleFlight(t *testing.T) {
	callCount := int32(0)
	sf := NewSingleFlight()

	var wg sync.WaitGroup
	results := make([]any, 10)
	errs := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = sf.Do("key", func() (any, error) {
				atomic.AddInt32(&callCount, 1)
				time.Sleep(50 * time.Millisecond)
				return "result", nil
			})
		}(i)
	}

	wg.Wait()

	// All should succeed
	for i := 0; i < 10; i++ {
		if errs[i] != nil {
			t.Errorf("request %d failed: %v", i, errs[i])
		}
		if results[i] != "result" {
			t.Errorf("request %d: expected 'result', got '%v'", i, results[i])
		}
	}

	// Only one call
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

package xsnowflake

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestGenerateUniqueIDsConcurrently(t *testing.T) {
	generator, err := New(7)
	if err != nil {
		t.Fatal(err)
	}

	const count = 1000
	ids := make(chan int64, count)
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, generateErr := generator.Generate()
			if generateErr != nil {
				t.Errorf("generate ID: %v", generateErr)
				return
			}
			ids <- id
		}()
	}
	wg.Wait()
	close(ids)

	seen := make(map[int64]struct{}, count)
	for id := range ids {
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate ID: %d", id)
		}
		seen[id] = struct{}{}
		if info := Parse(id); info.NodeID != 7 {
			t.Fatalf("unexpected node ID: %d", info.NodeID)
		}
	}
}

func TestClockMovedBackwards(t *testing.T) {
	generator, err := New(1)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	generator.now = func() time.Time { return now }
	if _, err = generator.Generate(); err != nil {
		t.Fatal(err)
	}

	generator.now = func() time.Time { return now.Add(-time.Millisecond) }
	if _, err = generator.Generate(); !errors.Is(err, ErrClockMovedBackwards) {
		t.Fatalf("expected clock rollback error, got %v", err)
	}
}

func TestDefaultGenerator(t *testing.T) {
	defaultGenerator.Lock()
	defaultGenerator.generator = nil
	defaultGenerator.Unlock()
	t.Cleanup(func() {
		defaultGenerator.Lock()
		defaultGenerator.generator = nil
		defaultGenerator.Unlock()
	})

	if _, err := ID(); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("expected not initialized error, got %v", err)
	}
	if err := Init(9); err != nil {
		t.Fatal(err)
	}
	if err := Init(10); !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("expected already initialized error, got %v", err)
	}

	id, err := ID()
	if err != nil {
		t.Fatal(err)
	}
	if info := Parse(id); info.NodeID != 9 {
		t.Fatalf("unexpected node ID: %d", info.NodeID)
	}
}

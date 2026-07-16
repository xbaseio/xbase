package xsnowflake

import (
	"errors"
	"sync"
)

var (
	ErrNotInitialized     = errors.New("snowflake: default generator is not initialized")
	ErrAlreadyInitialized = errors.New("snowflake: default generator is already initialized")
)

var defaultGenerator struct {
	sync.RWMutex
	generator *Generator
}

// Init initializes the process-wide generator. Every server instance in the
// cluster must pass a different node ID. Init may only be called once.
func Init(nodeID int64) error {
	generator, err := New(nodeID)
	if err != nil {
		return err
	}

	defaultGenerator.Lock()
	defer defaultGenerator.Unlock()
	if defaultGenerator.generator != nil {
		return ErrAlreadyInitialized
	}
	defaultGenerator.generator = generator
	return nil
}

// ID generates an ID using the process-wide generator.
func ID() (int64, error) {
	defaultGenerator.RLock()
	generator := defaultGenerator.generator
	defaultGenerator.RUnlock()
	if generator == nil {
		return 0, ErrNotInitialized
	}
	return generator.Generate()
}

// MustID is like ID but panics when the generator is unavailable or the clock
// moves backwards. Prefer ID in request-handling code where errors can be returned.
func MustID() int64 {
	id, err := ID()
	if err != nil {
		panic(err)
	}
	return id
}

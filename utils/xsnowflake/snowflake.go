// Package xsnowflake provides a small, concurrency-safe Snowflake ID generator.
package xsnowflake

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	nodeBits     = 10
	sequenceBits = 12

	maxNodeID   = int64(1<<nodeBits - 1)
	maxSequence = int64(1<<sequenceBits - 1)

	nodeShift      = sequenceBits
	timestampShift = nodeBits + sequenceBits

	// 2024-01-01 00:00:00 UTC. IDs can be generated for about 69 years from it.
	defaultEpoch = int64(1704067200000)
)

var ErrClockMovedBackwards = errors.New("snowflake: clock moved backwards")

// Generator generates 64-bit IDs ordered by generation time.
// A node ID must be unique among all concurrently running generators.
type Generator struct {
	mu            sync.Mutex
	nodeID        int64
	sequence      int64
	lastTimestamp int64
	epoch         int64
	now           func() time.Time
}

// New creates a generator for nodeID. Valid node IDs range from 0 to 1023.
func New(nodeID int64) (*Generator, error) {
	if nodeID < 0 || nodeID > maxNodeID {
		return nil, fmt.Errorf("snowflake: node ID must be between 0 and %d", maxNodeID)
	}

	return &Generator{
		nodeID: nodeID,
		epoch:  defaultEpoch,
		now:    time.Now,
	}, nil
}

// Generate returns a new Snowflake ID.
func (g *Generator) Generate() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	timestamp := g.now().UnixMilli()
	if timestamp < g.lastTimestamp {
		return 0, fmt.Errorf("%w: last=%d current=%d", ErrClockMovedBackwards, g.lastTimestamp, timestamp)
	}

	if timestamp == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			timestamp = g.waitNextMillis(timestamp)
		}
	} else {
		g.sequence = 0
	}

	g.lastTimestamp = timestamp
	return ((timestamp - g.epoch) << timestampShift) |
		(g.nodeID << nodeShift) |
		g.sequence, nil
}

func (g *Generator) waitNextMillis(timestamp int64) int64 {
	for timestamp <= g.lastTimestamp {
		timestamp = g.now().UnixMilli()
	}
	return timestamp
}

// Info contains the components encoded in an ID.
type Info struct {
	Time     time.Time
	NodeID   int64
	Sequence int64
}

// Parse decodes an ID generated with the default epoch.
func Parse(id int64) Info {
	timestamp := (id >> timestampShift) + defaultEpoch
	return Info{
		Time:     time.UnixMilli(timestamp),
		NodeID:   (id >> nodeShift) & maxNodeID,
		Sequence: id & maxSequence,
	}
}

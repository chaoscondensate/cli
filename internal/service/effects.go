package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

// ObservationClock captures caller-observed write and verification times.
// Domain builders must capture a default once and reuse the exact value.
type ObservationClock interface {
	Now() time.Time
}

// CSPRNG is the only source of salts, keys, nonces, and persisted random IDs.
type CSPRNG interface {
	ReadFull(context.Context, []byte) error
}

type Effects struct {
	Clock  ObservationClock
	Random CSPRNG
}

// ProductionEffects returns the only production effect implementations. Tests
// inject local deterministic fakes; no CLI, MCP, config, or environment input
// can select deterministic time or entropy.
func ProductionEffects() Effects {
	return Effects{Clock: systemClock{}, Random: systemCSPRNG{reader: rand.Reader}}
}

func (e Effects) Validate() error {
	if e.Clock == nil {
		return fmt.Errorf("observation clock is not configured")
	}
	if e.Random == nil {
		return fmt.Errorf("CSPRNG is not configured")
	}
	return nil
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

type systemCSPRNG struct {
	reader io.Reader
}

func (s systemCSPRNG) ReadFull(ctx context.Context, destination []byte) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if len(destination) == 0 {
		return nil
	}
	if _, err := io.ReadFull(s.reader, destination); err != nil {
		return fmt.Errorf("read operating-system random source: %w", err)
	}
	if ctx != nil && ctx.Err() != nil {
		for index := range destination {
			destination[index] = 0
		}
		return ctx.Err()
	}
	return nil
}

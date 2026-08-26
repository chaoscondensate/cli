package service

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

type fixedTestClock struct{ value time.Time }

func (clock fixedTestClock) Now() time.Time { return clock.value }

type deterministicTestRandom struct{ reader io.Reader }

func (random deterministicTestRandom) ReadFull(_ context.Context, destination []byte) error {
	_, err := io.ReadFull(random.reader, destination)
	return err
}

func TestEffectsCanBeInjectedWithoutProductionDeterministicMode(t *testing.T) {
	wantTime := time.Date(2026, 8, 26, 12, 0, 0, 0, time.FixedZone("BST", 3600))
	effects := Effects{
		Clock:  fixedTestClock{value: wantTime},
		Random: deterministicTestRandom{reader: bytes.NewReader([]byte{1, 2, 3, 4})},
	}
	if err := effects.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := effects.Clock.Now(); !got.Equal(wantTime) || got.Format(time.RFC3339) != "2026-08-26T12:00:00+01:00" {
		t.Fatalf("captured time = %s", got)
	}
	buffer := make([]byte, 4)
	if err := effects.Random.ReadFull(context.Background(), buffer); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buffer, []byte{1, 2, 3, 4}) {
		t.Fatalf("random bytes = %v", buffer)
	}
}

func TestProductionEffectsAreCompleteAndCancellable(t *testing.T) {
	effects := ProductionEffects()
	if err := effects.Validate(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	buffer := []byte{7, 7, 7}
	if err := effects.Random.ReadFull(ctx, buffer); err == nil {
		t.Fatal("canceled entropy read succeeded")
	}
	if !bytes.Equal(buffer, []byte{7, 7, 7}) {
		t.Fatalf("pre-canceled read changed destination: %v", buffer)
	}
}

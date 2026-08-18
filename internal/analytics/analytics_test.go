package analytics

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/art"
)

// TestOpenOnAFreshBoxStartsAtZero is the ordinary first-run case: nothing has
// been recorded yet, so there is no file, and Open must not treat that as an
// error - the server should never refuse to start because analytics has
// nothing to load.
func TestOpenOnAFreshBoxStartsAtZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open on a missing file returned an error: %v", err)
	}
	got := s.Snapshot()
	if got.Sessions != 0 {
		t.Errorf("Sessions = %d, want 0", got.Sessions)
	}
	for _, tier := range art.Tiers {
		if got.Tiers[tier.String()] != 0 {
			t.Errorf("Tiers[%s] = %d, want 0", tier, got.Tiers[tier.String()])
		}
	}
}

// TestOpenOnACorruptFileStillReturnsAUsableStore guards the failure mode this
// package cares most about: a visitor's session must never break because the
// analytics file on disk got corrupted somehow. Open reports the error so it
// can be logged, but hands back a Store that works.
func TestOpenOnACorruptFileStillReturnsAUsableStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")
	if err := os.WriteFile(path, []byte("not json"), 0o640); err != nil {
		t.Fatalf("could not seed a corrupt file: %v", err)
	}

	s, err := Open(path)
	if err == nil {
		t.Fatal("Open on a corrupt file returned no error")
	}
	if s == nil {
		t.Fatal("Open on a corrupt file returned a nil Store")
	}
	if err := s.RecordSession(); err != nil {
		t.Fatalf("RecordSession on a Store opened from a corrupt file: %v", err)
	}
	if got := s.Snapshot().Sessions; got != 1 {
		t.Errorf("Sessions = %d, want 1", got)
	}
}

// TestRecordTierCountsBothTheSessionAndTheRung is the acceptance criterion
// "a session counter and a render-tier histogram are recorded": a session
// that reaches a render tier is one session and one point in the histogram,
// not either-or.
func TestRecordTierCountsBothTheSessionAndTheRung(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := s.RecordTier(art.Sextant); err != nil {
		t.Fatalf("RecordTier: %v", err)
	}
	if err := s.RecordTier(art.Quad); err != nil {
		t.Fatalf("RecordTier: %v", err)
	}
	if err := s.RecordTier(art.Quad); err != nil {
		t.Fatalf("RecordTier: %v", err)
	}

	got := s.Snapshot()
	if got.Sessions != 3 {
		t.Errorf("Sessions = %d, want 3", got.Sessions)
	}
	want := map[art.Tier]uint64{art.Sextant: 1, art.Quad: 2, art.VHalf: 0, art.Colorless: 0}
	for tier, count := range want {
		if got.Tiers[tier.String()] != count {
			t.Errorf("Tiers[%s] = %d, want %d", tier, got.Tiers[tier.String()], count)
		}
	}
}

// TestRecordSessionCountsTheVisitButNoTier is D2's piped path: `ssh ... |
// cat` never allocates a PTY and never picks a rung of the render ladder, but
// it is still a visit, so the session counter should count it even though the
// histogram has nowhere to put it.
func TestRecordSessionCountsTheVisitButNoTier(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := s.RecordSession(); err != nil {
		t.Fatalf("RecordSession: %v", err)
	}

	got := s.Snapshot()
	if got.Sessions != 1 {
		t.Errorf("Sessions = %d, want 1", got.Sessions)
	}
	for _, tier := range art.Tiers {
		if got.Tiers[tier.String()] != 0 {
			t.Errorf("a piped session bumped Tiers[%s] to %d, want 0", tier, got.Tiers[tier.String()])
		}
	}
}

// TestCountersSurviveAServiceRestart is the acceptance criterion of the same
// name: a second Store opened from the same path - standing in for the
// process restart a systemd unit does on every deploy - has to pick up
// exactly what the first one left behind.
func TestCountersSurviveAServiceRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")

	first, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := first.RecordTier(art.Sextant); err != nil {
		t.Fatalf("RecordTier: %v", err)
	}
	if err := first.RecordTier(art.Colorless); err != nil {
		t.Fatalf("RecordTier: %v", err)
	}
	if err := first.RecordSession(); err != nil {
		t.Fatalf("RecordSession: %v", err)
	}

	second, err := Open(path)
	if err != nil {
		t.Fatalf("Open after restart: %v", err)
	}
	got := second.Snapshot()
	if got.Sessions != 3 {
		t.Errorf("after restart, Sessions = %d, want 3", got.Sessions)
	}
	if got.Tiers[art.Sextant.String()] != 1 {
		t.Errorf("after restart, Tiers[sextant] = %d, want 1", got.Tiers[art.Sextant.String()])
	}
	if got.Tiers[art.Colorless.String()] != 1 {
		t.Errorf("after restart, Tiers[colorless] = %d, want 1", got.Tiers[art.Colorless.String()])
	}
}

// TestConcurrentSessionsAllCount is Wish serving more than one session at
// once: every visitor's goroutine calls into the same Store, and none of
// their counts should be lost to a race.
func TestConcurrentSessionsAllCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var err error
			if i%2 == 0 {
				err = s.RecordTier(art.Quad)
			} else {
				err = s.RecordSession()
			}
			if err != nil {
				t.Errorf("record: %v", err)
			}
		}(i)
	}
	wg.Wait()

	got := s.Snapshot()
	if got.Sessions != n {
		t.Errorf("Sessions = %d, want %d", got.Sessions, n)
	}
	if got.Tiers[art.Quad.String()] != n/2 {
		t.Errorf("Tiers[quad] = %d, want %d", got.Tiers[art.Quad.String()], n/2)
	}
}

// TestSnapshotIsACopy guards against a caller mutating the live state through
// the map Snapshot hands back.
func TestSnapshotIsACopy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.RecordTier(art.Quad); err != nil {
		t.Fatalf("RecordTier: %v", err)
	}

	got := s.Snapshot()
	got.Tiers[art.Quad.String()] = 999
	got.Sessions = 999

	fresh := s.Snapshot()
	if fresh.Sessions != 1 {
		t.Errorf("Sessions = %d after mutating a snapshot, want 1", fresh.Sessions)
	}
	if fresh.Tiers[art.Quad.String()] != 1 {
		t.Errorf("Tiers[quad] = %d after mutating a snapshot, want 1", fresh.Tiers[art.Quad.String()])
	}
}

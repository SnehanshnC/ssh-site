// Package analytics is D7: enough to know the site is being visited, and
// nothing more.
//
// A Store holds two numbers - a session count and a render-tier histogram -
// and persists them to a plain JSON file so they survive a service restart.
// There is no third-party service, no IP retention beyond what journald
// already keeps, and nothing here identifies a visitor: a Store never sees
// more than "a session happened" and, for a session that reached a render
// tier, which rung of the ladder it was.
//
// Nothing in this package is reachable from a visitor session. The app never
// serves these counters back over port 22; reading them is a matter of
// logging into the box over the admin sshd and reading the file directly
// (sudo is required, since the app runs as the unprivileged ssh-site user).
package analytics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/SnehanshnC/ssh-site/internal/art"
)

// Counters is the whole of what this package records: how many sessions have
// been served, and, of the sessions that reached a render tier, how many
// landed on each rung of the ladder. D2's piped document path never picks a
// tier, so Sessions can run ahead of the sum of Tiers.
type Counters struct {
	Sessions uint64            `json:"sessions"`
	Tiers    map[string]uint64 `json:"tiers"`
}

// Store is Counters plus the file it is persisted to. Every Record call
// writes through to disk immediately: at this site's traffic, a synchronous
// write per session is simpler than a background flush loop and never risks
// losing a count to a crash between them.
type Store struct {
	path string

	mu       sync.Mutex
	counters Counters
}

// Open loads counters from path if a file is already there, and returns a
// Store ready to record more. A missing file is the ordinary first-run case
// and starts from zero with no error. A file that exists but cannot be read
// or parsed also starts from zero, but is returned alongside the error that
// explains why - the caller can log it, but the server should never refuse
// to start, and should never refuse a visitor, over a corrupt analytics
// file. The next successful Record overwrites it with a well-formed one.
func Open(path string) (*Store, error) {
	s := &Store{path: path, counters: zeroCounters()}

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, fmt.Errorf("read analytics state %s: %w", path, err)
	}

	var c Counters
	if err := json.Unmarshal(b, &c); err != nil {
		return s, fmt.Errorf("parse analytics state %s: %w", path, err)
	}
	if c.Tiers == nil {
		c.Tiers = map[string]uint64{}
	}
	for _, t := range art.Tiers {
		if _, ok := c.Tiers[t.String()]; !ok {
			c.Tiers[t.String()] = 0
		}
	}
	s.counters = c
	return s, nil
}

// RecordSession counts one visit with no render tier: D2's piped document
// path, where a session never has a PTY to pick a rung of the ladder from.
func (s *Store) RecordSession() error {
	return s.record(nil)
}

// RecordTier counts one visit that reached the given render tier.
func (s *Store) RecordTier(tier art.Tier) error {
	return s.record(&tier)
}

// Snapshot returns a copy of the current counters, safe for a caller to read
// without racing a concurrent Record. Nothing in cmd/ssh-site calls this - it
// exists for tests and for any future admin-side reader - since the whole
// point of D7 is that these numbers are read off the box, not through the
// app.
func (s *Store) Snapshot() Counters {
	s.mu.Lock()
	defer s.mu.Unlock()
	tiers := make(map[string]uint64, len(s.counters.Tiers))
	for k, v := range s.counters.Tiers {
		tiers[k] = v
	}
	return Counters{Sessions: s.counters.Sessions, Tiers: tiers}
}

func (s *Store) record(tier *art.Tier) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counters.Sessions++
	if tier != nil {
		s.counters.Tiers[tier.String()]++
	}
	return s.save()
}

// save writes the counters to path by writing a temp file in the same
// directory and renaming it into place, so a reader - or a crash - never
// sees a half-written file. Must be called with mu held.
func (s *Store) save() error {
	b, err := json.MarshalIndent(s.counters, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal analytics state: %w", err)
	}
	b = append(b, '\n')

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".analytics-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp analytics file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op once the rename below succeeds

	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp analytics file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp analytics file: %w", err)
	}
	// Readable only by its owner and group - the box runs this as the
	// unprivileged ssh-site user in a 750 directory only root can also
	// enter, which is what makes "readable only over the admin sshd"
	// (sudo, on the box) true without this package doing anything special.
	if err := os.Chmod(tmpPath, 0o640); err != nil {
		return fmt.Errorf("chmod temp analytics file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("rename temp analytics file into place: %w", err)
	}
	return nil
}

func zeroCounters() Counters {
	tiers := make(map[string]uint64, len(art.Tiers))
	for _, t := range art.Tiers {
		tiers[t.String()] = 0
	}
	return Counters{Tiers: tiers}
}

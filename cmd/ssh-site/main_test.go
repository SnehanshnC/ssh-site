package main

import (
	"testing"
	"time"
)

// TestServerTimeoutsStayAsHardeningSet guards D4's constraint: the app-level
// ten-minute idle timer in internal/ui is new, but the server's own
// IdleTimeout and MaxTimeout are backstops that research 09 already set and
// this build must not move - a well-behaved session should never reach them.
func TestServerTimeoutsStayAsHardeningSet(t *testing.T) {
	if idleTimeout != 15*time.Minute {
		t.Errorf("the server's IdleTimeout is %s, want 15m", idleTimeout)
	}
	if maxTimeout != 1*time.Hour {
		t.Errorf("the server's MaxTimeout is %s, want 1h", maxTimeout)
	}
}

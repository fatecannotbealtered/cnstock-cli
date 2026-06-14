package cmd

import (
	"testing"
	"time"
)

// isolateHome points os.UserHomeDir at a throwaway directory so the consumed
// store under ~/.cnstock-cli never touches the real home. USERPROFILE covers
// Windows, HOME covers Unix.
func isolateHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOME", dir)
}

// A confirm token may drive exactly one update; replaying it (e.g. an agent
// retrying an update that timed out) must be detectable so the operation cannot
// be duplicated.
func TestConfirmTokenSingleUse(t *testing.T) {
	isolateHome(t)

	now := time.Now()
	token := "ct_eyJtZXRob2QiOiJucG0ifQ.deadbeef"

	if isConfirmTokenConsumed(token, now) {
		t.Fatal("fresh token should not be consumed yet")
	}

	markConfirmTokenConsumed(token, now.Add(15*time.Minute).Unix(), now)

	if !isConfirmTokenConsumed(token, now) {
		t.Fatal("token should be consumed after marking (replay must be detected)")
	}

	// A different token sharing the store is unaffected.
	if isConfirmTokenConsumed("ct_other.cafef00d", now) {
		t.Fatal("unrelated token should not be reported as consumed")
	}
}

// Entries past their expiry are pruned on access, so a token whose window has
// closed no longer blocks and the store cannot grow without bound.
func TestConfirmTokenExpiryPruned(t *testing.T) {
	isolateHome(t)

	issued := time.Now()
	token := "ct_eyJtZXRob2QiOiJnbyJ9.0badf00d"
	markConfirmTokenConsumed(token, issued.Add(15*time.Minute).Unix(), issued)

	later := issued.Add(16 * time.Minute)
	if isConfirmTokenConsumed(token, later) {
		t.Fatal("expired token should be pruned, not reported as consumed")
	}
}

func TestTokenFingerprintStable(t *testing.T) {
	a := tokenFingerprint("ct_123_abc")
	b := tokenFingerprint("ct_123_abc")
	c := tokenFingerprint("ct_123_abd")
	if a != b {
		t.Fatal("fingerprint not stable for the same token")
	}
	if a == c {
		t.Fatal("fingerprint collided for different tokens")
	}
	if len(a) != 16 {
		t.Fatalf("fingerprint length = %d, want 16", len(a))
	}
}

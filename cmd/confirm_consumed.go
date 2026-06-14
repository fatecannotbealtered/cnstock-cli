package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Single-use confirm tokens: once a confirm token has been accepted to perform
// the update, its fingerprint is recorded so the SAME token cannot drive a
// second update. This gives agents safe-retry semantics — a confirmed update
// that times out cannot be blindly replayed; the retry is rejected with
// E_CONFLICT and the agent must re-run --dry-run (which reveals the now-current
// state, e.g. that the version is already installed). The store lives at
// ~/.cnstock-cli/confirm-consumed.json (0600) and is pruned of expired entries
// on every access so it cannot grow without bound.
//
// Unlike a ct_<unix>_<hex> token whose expiry can be read from the prefix,
// cnstock's confirm token carries its expiry inside the signed payload
// (updateConfirmPayload.ExpiresAt). The caller therefore supplies the expiry it
// has already validated, rather than the store re-parsing the token.
var consumedTokensMu sync.Mutex

func consumedTokensPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", err
	}
	return filepath.Join(home, ".cnstock-cli", "confirm-consumed.json"), nil
}

// tokenFingerprint is a short, non-reversible id for a confirm token.
func tokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])[:16]
}

func loadConsumedTokens(path string, now time.Time) map[string]int64 {
	out := map[string]int64{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var stored map[string]int64
	if json.Unmarshal(data, &stored) != nil {
		return out
	}
	// Drop expired entries so the file cannot grow without bound.
	for fp, exp := range stored {
		if exp > now.Unix() {
			out[fp] = exp
		}
	}
	return out
}

// isConfirmTokenConsumed reports whether this token has already been used.
func isConfirmTokenConsumed(token string, now time.Time) bool {
	path, err := consumedTokensPath()
	if err != nil {
		return false // cannot check; do not block the operation
	}
	consumedTokensMu.Lock()
	defer consumedTokensMu.Unlock()
	tokens := loadConsumedTokens(path, now)
	_, ok := tokens[tokenFingerprint(token)]
	return ok
}

// markConfirmTokenConsumed records the token as used until its expiry. Best
// effort: a storage failure does not block the update (warned, single-use simply
// cannot be guaranteed on that host).
func markConfirmTokenConsumed(token string, expiresUnix int64, now time.Time) {
	path, err := consumedTokensPath()
	if err != nil {
		return
	}
	consumedTokensMu.Lock()
	defer consumedTokensMu.Unlock()
	tokens := loadConsumedTokens(path, now)
	tokens[tokenFingerprint(token)] = expiresUnix
	data, err := json.Marshal(tokens)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}

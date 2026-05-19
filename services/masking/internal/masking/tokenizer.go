package masking

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Tokenizer produces deterministic, opaque tokens for PII field values.
//
// Determinism is the key property: the same input always produces the same
// token for a given key. This allows downstream consumers to JOIN on
// tokenized fields across messages without ever seeing the raw value.
//
// We use HMAC-SHA256 truncated to 16 bytes (32 hex chars). HMAC binds the
// token to the secret key, so tokens are not reversible without the key.
// The namespace parameter scopes tokens to a field type (e.g. "account_id")
// so the same raw value in two different field namespaces produces different
// tokens. This prevents cross-field token correlation attacks.
//
// Why HMAC rather than a random UUID per value?
// Random UUIDs require a lookup table to maintain consistency across messages.
// That lookup table is a database, which means a network call on the hot path.
// HMAC is stateless: the same input always produces the same output.
type Tokenizer struct {
	key    []byte
	prefix string
}

func NewTokenizer(key, prefix string) *Tokenizer {
	return &Tokenizer{
		key:    []byte(key),
		prefix: prefix,
	}
}

func (t *Tokenizer) Tokenize(namespace, value string) string {
	mac := hmac.New(sha256.New, t.key)
	// Separator between namespace and value prevents namespace="ac" value="count_id:foo"
	// from colliding with namespace="account_id" value="foo".
	fmt.Fprintf(mac, "%s\x00%s", namespace, value)
	return t.prefix + hex.EncodeToString(mac.Sum(nil))[:32]
}

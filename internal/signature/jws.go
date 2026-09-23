// Package signature implements the bounded M0 JWS envelope experiment.
// It signs exact bytes, not report files, JSON schemas, or signer identities.
package signature

import (
	"bytes"
	"crypto/ed25519"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json/v2"
	"errors"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jws"
)

const (
	MaxPayloadBytes  = 1 << 20
	maxHeaderBytes   = 1024
	maxEnvelopeBytes = (MaxPayloadBytes+2)/3*4 + (maxHeaderBytes+2)/3*4 + 88 + 2
	MediaType        = "webfence-manifest+jws"
)

var (
	ErrKey     = errors.New("signature_invalid_key")
	ErrPayload = errors.New("signature_invalid_payload")
	ErrInvalid = errors.New("signature_invalid_envelope")
)

type header struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Type      string `json:"typ"`
}

// Sign embeds a nonempty payload in a compact Ed25519 JWS. The caller owns
// canonicalization and key storage. No key or payload is persisted or logged.
func Sign(payload []byte, keyID string, key ed25519.PrivateKey) ([]byte, error) {
	if !validKeyID(keyID) || len(key) != ed25519.PrivateKeySize {
		return nil, ErrKey
	}
	if subtle.ConstantTimeCompare(key, ed25519.NewKeyFromSeed(key[:ed25519.SeedSize])) != 1 {
		return nil, ErrKey
	}
	if len(payload) == 0 || len(payload) > MaxPayloadBytes {
		return nil, ErrPayload
	}
	h := jws.NewHeaders()
	if err := h.Set(jws.KeyIDKey, keyID); err != nil {
		return nil, ErrKey
	}
	if err := h.Set(jws.TypeKey, MediaType); err != nil {
		return nil, ErrInvalid
	}
	result, err := jws.Sign(payload, jws.WithKey(jwa.EdDSAEd25519(), key, jws.WithProtectedHeaders(h)), jws.WithCompact())
	if err != nil {
		return nil, ErrInvalid
	}
	return result, nil
}

// Verify requires a public key and its ID selected by the caller from a trusted
// source outside the envelope. It never follows URLs or imports embedded keys.
// Success authenticates only the returned bytes with respect to that key; it
// does not establish identity, revocation status, or report validity.
func Verify(envelope []byte, expectedKeyID string, trustedKey ed25519.PublicKey) ([]byte, error) {
	if !validKeyID(expectedKeyID) || len(trustedKey) != ed25519.PublicKeySize {
		return nil, ErrKey
	}
	if len(envelope) == 0 || len(envelope) > maxEnvelopeBytes || bytes.Count(envelope, []byte(".")) != 2 {
		return nil, ErrInvalid
	}
	parts := bytes.Split(envelope, []byte("."))
	headerBytes, ok := decodeSegment(parts[0], maxHeaderBytes)
	if !ok {
		return nil, ErrInvalid
	}
	var h header
	// json/v2 rejects duplicate names and invalid UTF-8 by default. Unknown
	// fields (including crit/b64/jku/jwk) are outside this deliberately narrow profile.
	if json.Unmarshal(headerBytes, &h, json.RejectUnknownMembers(true)) != nil ||
		h.Algorithm != "Ed25519" || h.Type != MediaType || h.KeyID != expectedKeyID {
		return nil, ErrInvalid
	}
	if _, ok := decodeSegment(parts[1], MaxPayloadBytes); !ok {
		return nil, ErrInvalid
	}
	sig, ok := decodeSegment(parts[2], ed25519.SignatureSize)
	if !ok || len(sig) != ed25519.SignatureSize {
		return nil, ErrInvalid
	}
	payload, err := jws.Verify(envelope, jws.WithKey(jwa.EdDSAEd25519(), trustedKey), jws.WithCompact())
	if err != nil {
		return nil, ErrInvalid
	}
	return payload, nil
}

func validKeyID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, c := range []byte(id) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func decodeSegment(src []byte, limit int) ([]byte, bool) {
	if len(src) == 0 || len(src) > (limit+2)/3*4 {
		return nil, false
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(string(src))
	if err != nil || len(decoded) > limit || base64.RawURLEncoding.EncodeToString(decoded) != string(src) {
		return nil, false
	}
	return decoded, true
}

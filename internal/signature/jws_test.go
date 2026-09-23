package signature

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"
)

func keys(t testing.TB) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

// Independent standard-library signing creates valid signatures over bad
// profiles, so rejection cannot be explained just by a signature mismatch.
func rawEnvelope(priv ed25519.PrivateKey, h, payload []byte) []byte {
	input := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(payload)
	return []byte(input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, []byte(input))))
}

func TestRoundTripAndIndependentVerification(t *testing.T) {
	pub, priv := keys(t)
	for _, payload := range [][]byte{[]byte("{\"evidence\":\"caffè ☕\"}"), {0, 255, 1}, bytes.Repeat([]byte("x"), MaxPayloadBytes)} {
		envelope, err := Sign(payload, "operator-1", priv)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Verify(envelope, "operator-1", pub)
		if err != nil || !bytes.Equal(payload, got) {
			t.Fatalf("round trip: %v", err)
		}
		parts := bytes.Split(envelope, []byte("."))
		sig, err := base64.RawURLEncoding.DecodeString(string(parts[2]))
		if err != nil || !ed25519.Verify(pub, bytes.Join(parts[:2], []byte(".")), sig) {
			t.Fatal("independent verification failed")
		}
	}
	h := []byte(`{"typ":"webfence-manifest+jws","kid":"operator-1","alg":"Ed25519"}`)
	if got, err := Verify(rawEnvelope(priv, h, []byte("independent")), "operator-1", pub); err != nil || string(got) != "independent" {
		t.Fatalf("independent signer: %v", err)
	}
}

func TestRejectValidSignaturesOutsideProfile(t *testing.T) {
	pub, priv := keys(t)
	for name, h := range map[string]string{
		"legacy":           `{"alg":"EdDSA","kid":"operator-1","typ":"webfence-manifest+jws"}`,
		"none":             `{"alg":"none","kid":"operator-1","typ":"webfence-manifest+jws"}`,
		"wrong algorithm":  `{"alg":"HS256","kid":"operator-1","typ":"webfence-manifest+jws"}`,
		"wrong kid":        `{"alg":"Ed25519","kid":"operator-2","typ":"webfence-manifest+jws"}`,
		"wrong type":       `{"alg":"Ed25519","kid":"operator-1","typ":"JWT"}`,
		"missing type":     `{"alg":"Ed25519","kid":"operator-1"}`,
		"duplicate":        `{"alg":"Ed25519","kid":"operator-1","kid":"operator-1","typ":"webfence-manifest+jws"}`,
		"case variant":     `{"ALG":"Ed25519","kid":"operator-1","typ":"webfence-manifest+jws"}`,
		"URL":              `{"alg":"Ed25519","kid":"operator-1","typ":"webfence-manifest+jws","jku":"https://example.invalid/keys"}`,
		"embedded key":     `{"alg":"Ed25519","kid":"operator-1","typ":"webfence-manifest+jws","jwk":{}}`,
		"crit":             `{"alg":"Ed25519","kid":"operator-1","typ":"webfence-manifest+jws","crit":["other"]}`,
		"b64":              `{"alg":"Ed25519","kid":"operator-1","typ":"webfence-manifest+jws","b64":true}`,
		"null":             `null`,
		"invalid utf8":     "{\"alg\":\"Ed25519\",\"kid\":\"operator-1\",\"typ\":\"\xff\"}",
		"oversized header": strings.Repeat(" ", maxHeaderBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := Verify(rawEnvelope(priv, []byte(h), []byte("fixture")), "operator-1", pub)
			if !errors.Is(err, ErrInvalid) || got != nil {
				t.Fatalf("accepted invalid profile: %v", err)
			}
		})
	}
}

func TestTamperingAndInvalidInputs(t *testing.T) {
	pub, priv := keys(t)
	envelope, err := Sign([]byte("fixture"), "operator-1", priv)
	if err != nil {
		t.Fatal(err)
	}
	parts := bytes.Split(envelope, []byte("."))
	for i := range parts {
		altered := bytes.Clone(envelope)
		offset := 0
		for j := 0; j < i; j++ {
			offset += len(parts[j]) + 1
		}
		if altered[offset] == 'A' {
			altered[offset] = 'B'
		} else {
			altered[offset] = 'A'
		}
		if got, err := Verify(altered, "operator-1", pub); !errors.Is(err, ErrInvalid) || got != nil {
			t.Fatalf("tampering part %d: %v", i, err)
		}
	}
	other, _ := keys(t)
	if _, err := Verify(envelope, "operator-1", other); !errors.Is(err, ErrInvalid) {
		t.Fatal("accepted untrusted key")
	}
	for _, token := range [][]byte{nil, []byte("a.b.c.d"), []byte(`{"payload":"x"}`), append(bytes.Clone(envelope), '\n'), bytes.Replace(envelope, []byte("."), []byte("=."), 1), bytes.Repeat([]byte("a"), maxEnvelopeBytes+1)} {
		if _, err := Verify(token, "operator-1", pub); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid token: %v", err)
		}
	}
	for _, id := range []string{"", "a:b", "é", "a\n", strings.Repeat("a", 65)} {
		if _, err := Sign([]byte("x"), id, priv); !errors.Is(err, ErrKey) {
			t.Fatal("accepted bad ID")
		}
		if _, err := Verify(envelope, id, pub); !errors.Is(err, ErrKey) {
			t.Fatal("accepted bad expected ID")
		}
	}
	corrupt := bytes.Clone(priv)
	corrupt[len(corrupt)-1] ^= 1
	for _, k := range []ed25519.PrivateKey{nil, {1}, corrupt} {
		if _, err := Sign([]byte("x"), "operator-1", k); !errors.Is(err, ErrKey) {
			t.Fatal("accepted bad private key")
		}
	}
	if _, err := Verify(envelope, "operator-1", nil); !errors.Is(err, ErrKey) {
		t.Fatal("accepted nil public key")
	}
	for _, payload := range [][]byte{nil, bytes.Repeat([]byte("x"), MaxPayloadBytes+1)} {
		if _, err := Sign(payload, "operator-1", priv); !errors.Is(err, ErrPayload) {
			t.Fatal("accepted invalid payload")
		}
		h := []byte(`{"alg":"Ed25519","kid":"operator-1","typ":"webfence-manifest+jws"}`)
		if _, err := Verify(rawEnvelope(priv, h, payload), "operator-1", pub); !errors.Is(err, ErrInvalid) {
			t.Fatal("accepted invalid embedded payload")
		}
	}
}

func TestConcurrentSigning(t *testing.T) {
	pub, priv := keys(t)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Go(func() {
			envelope, err := Sign([]byte("fixture"), "operator-1", priv)
			if err != nil {
				t.Error(err)
				return
			}
			got, err := Verify(envelope, "operator-1", pub)
			if err != nil || string(got) != "fixture" {
				t.Errorf("concurrent roundtrip: %v", err)
			}
		})
	}
	wg.Wait()
}

func FuzzVerify(f *testing.F) {
	pub, priv := keys(f)
	valid, err := Sign([]byte("fixture"), "operator-1", priv)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	f.Add([]byte("a.b.c"))
	f.Add([]byte(`{"signatures":[]}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		got, err := Verify(input, "operator-1", pub)
		if err != nil {
			if got != nil {
				t.Fatal("unverified bytes returned")
			}
			return
		}
		if len(got) == 0 || len(got) > MaxPayloadBytes {
			t.Fatal("payload bounds violated")
		}
		parts := bytes.Split(input, []byte("."))
		sig, err := base64.RawURLEncoding.DecodeString(string(parts[2]))
		if err != nil || !ed25519.Verify(pub, bytes.Join(parts[:2], []byte(".")), sig) {
			t.Fatal("verification disagreement")
		}
	})
}

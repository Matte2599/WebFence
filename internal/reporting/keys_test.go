package reporting

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

type memorySecrets struct {
	mu          sync.Mutex
	values      map[string][]byte
	unavailable bool
}

func (m *memorySecrets) Get(_ context.Context, id string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.unavailable {
		return nil, ErrUnavailable
	}
	v, ok := m.values[id]
	if !ok {
		return nil, ErrUnavailable
	}
	return append([]byte(nil), v...), nil
}
func (m *memorySecrets) Set(_ context.Context, id string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.unavailable {
		return ErrUnavailable
	}
	if m.values == nil {
		m.values = map[string][]byte{}
	}
	m.values[id] = append([]byte(nil), value...)
	return nil
}
func (m *memorySecrets) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, id)
	return nil
}

func TestNativeKeyContractRotationRevocationAndImport(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	trust, err := OpenTrustStore(filepath.Join(dir, "trust.json"))
	if err != nil {
		t.Fatal(err)
	}
	secrets := &memorySecrets{}
	keys, err := NewKeyring(trust, secrets)
	if err != nil {
		t.Fatal(err)
	}
	release, err := trust.mutationLock()
	if err != nil {
		t.Fatal(err)
	}
	otherWriter, err := OpenTrustStore(filepath.Join(dir, "trust.json"))
	if err != nil {
		t.Fatal(err)
	}
	if unlock, e := otherWriter.mutationLock(); e == nil {
		unlock()
		t.Fatal("concurrent trust writer accepted")
	}
	release()
	first, err := keys.Generate(ctx, "Synthetic operator")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := secrets.Get(ctx, secretID(first.KeyID)); err != nil {
		t.Fatal(err)
	}
	pubPath := filepath.Join(dir, "operator-public.json")
	if err := trust.ExportPublic(first.KeyID, pubPath); err != nil {
		t.Fatal(err)
	}
	signed := filepath.Join(dir, "signed.wfr")
	if _, err := keys.ExportSigned(ctx, fixtureSnapshot(), signed, first.KeyID, ExportOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(signed, trust); err != nil {
		t.Fatal(err)
	}
	second, err := keys.Rotate(ctx, first.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	if second.KeyID == first.KeyID {
		t.Fatal("rotation reused key")
	}
	if _, err := keys.ExportSigned(ctx, fixtureSnapshot(), filepath.Join(dir, "old.wfr"), first.KeyID, ExportOptions{}); !errors.Is(err, ErrUntrustedKey) {
		t.Fatalf("retired key signed: %v", err)
	}
	if result, err := Verify(signed, trust); err != nil || result.TrustStatus != "retired" {
		t.Fatalf("retired verification: %+v %v", result, err)
	}
	if err := keys.Revoke(ctx, first.KeyID); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(signed, trust); !errors.Is(err, ErrRevokedKey) {
		t.Fatalf("revocation not applied: %v", err)
	}
	if _, err := keys.ExportSigned(ctx, fixtureSnapshot(), filepath.Join(dir, "new.wfr"), second.KeyID, ExportOptions{}); err != nil {
		t.Fatal(err)
	}
	other, err := OpenTrustStore(filepath.Join(dir, "other-trust.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.ImportTrusted(pubPath, "invalid"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unverified import: %v", err)
	}
	if _, err := other.ImportTrusted(pubPath, first.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(signed, other); err != nil {
		t.Fatalf("out-of-band trust: %v", err)
	}
	secrets.unavailable = true
	if _, err := keys.ExportSigned(ctx, fixtureSnapshot(), filepath.Join(dir, "locked.wfr"), second.KeyID, ExportOptions{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("keychain unavailable fallback: %v", err)
	}
}

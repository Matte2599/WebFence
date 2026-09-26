package reporting

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	jsonv2 "encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/Matte2599/WebFence/internal/credentials"
)

// SecretStore is implemented by credentials.Store. Tests inject an isolated
// synthetic implementation; production never stores private keys in files.
type SecretStore interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte) error
	Delete(context.Context, string) error
}

type KeyRecord struct {
	KeyID        string `json:"key_id"`
	PublicKey    string `json:"public_key"`
	Fingerprint  string `json:"fingerprint_sha256"`
	Identity     string `json:"identity"`
	Status       string `json:"status"`
	CreatedAtUTC string `json:"created_at_utc"`
	UpdatedAtUTC string `json:"updated_at_utc"`
}
type trustDocument struct {
	Schema string      `json:"schema"`
	Keys   []KeyRecord `json:"keys"`
}
type TrustStore struct {
	path string
	mu   sync.Mutex
}

func OpenTrustStore(path string) (*TrustStore, error) {
	if !filepath.IsAbs(path) || filepath.Base(path) == "." {
		return nil, ErrInvalid
	}
	info, err := os.Lstat(path)
	if err == nil && (!info.Mode().IsRegular() || runtime.GOOS != "windows" && info.Mode().Perm()&0o022 != 0) {
		return nil, ErrInvalid
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, ErrUnavailable
	}
	return &TrustStore{path: path}, nil
}

func validKeyID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
func validIdentity(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
func validateKeyRecord(k KeyRecord) bool {
	if !validKeyID(k.KeyID) || !validIdentity(k.Identity) || (k.Status != "active" && k.Status != "retired" && k.Status != "revoked") {
		return false
	}
	pub, err := DecodePublicKeyText(k.PublicKey)
	if err != nil {
		return false
	}
	h := sha256.Sum256(pub)
	if k.Fingerprint != hex.EncodeToString(h[:]) {
		return false
	}
	created, e1 := time.Parse(time.RFC3339Nano, k.CreatedAtUTC)
	updated, e2 := time.Parse(time.RFC3339Nano, k.UpdatedAtUTC)
	return e1 == nil && e2 == nil && !updated.Before(created)
}

func (t *TrustStore) read() (trustDocument, error) {
	info, statErr := os.Lstat(t.path)
	if errors.Is(statErr, os.ErrNotExist) {
		return trustDocument{Schema: "webfence-trust-v1", Keys: []KeyRecord{}}, nil
	}
	if statErr != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 || runtime.GOOS != "windows" && info.Mode().Perm()&0o022 != 0 {
		return trustDocument{}, ErrInvalid
	}
	data, err := os.ReadFile(t.path)
	if err != nil || len(data) > 1<<20 {
		return trustDocument{}, ErrUnavailable
	}
	var doc trustDocument
	if jsonv2.Unmarshal(data, &doc, jsonv2.RejectUnknownMembers(true)) != nil || doc.Schema != "webfence-trust-v1" || len(doc.Keys) > 256 {
		return trustDocument{}, ErrInvalid
	}
	seen := map[string]bool{}
	for _, k := range doc.Keys {
		if !validateKeyRecord(k) || seen[k.KeyID] {
			return trustDocument{}, ErrInvalid
		}
		seen[k.KeyID] = true
	}
	return doc, nil
}

// mutationLock serializes trust decisions across processes on the same local
// filesystem. The persistent lock file contains no key material.
func (t *TrustStore) mutationLock() (func(), error) {
	t.mu.Lock()
	path := t.path + ".lock"
	info, err := os.Lstat(path)
	if err == nil && (!info.Mode().IsRegular() || runtime.GOOS != "windows" && info.Mode().Perm()&0o022 != 0) {
		t.mu.Unlock()
		return nil, ErrInvalid
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.mu.Unlock()
		return nil, ErrUnavailable
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.mu.Unlock()
		return nil, ErrUnavailable
	}
	if err := lockTrustFile(f); err != nil {
		_ = f.Close()
		t.mu.Unlock()
		return nil, ErrUnavailable
	}
	return func() { _ = unlockTrustFile(f); _ = f.Close(); t.mu.Unlock() }, nil
}

func (t *TrustStore) save(doc trustDocument) error {
	data, err := jsonv2.Marshal(doc)
	if err != nil || len(data) > 1<<20 {
		return ErrInvalid
	}
	f, err := os.CreateTemp(filepath.Dir(t.path), ".webfence-trust-*.tmp")
	if err != nil {
		return ErrUnavailable
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if f.Chmod(0o600) != nil {
		_ = f.Close()
		return ErrUnavailable
	}
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return ErrUnavailable
	}
	if f.Sync() != nil {
		_ = f.Close()
		return ErrUnavailable
	}
	if f.Close() != nil {
		return ErrUnavailable
	}
	if os.Rename(tmp, t.path) != nil {
		return ErrUnavailable
	}
	return nil
}

func (t *TrustStore) Lookup(id string) (TrustedKey, error) {
	if t == nil || !validKeyID(id) {
		return TrustedKey{}, ErrInvalid
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	doc, err := t.read()
	if err != nil {
		return TrustedKey{}, err
	}
	for _, k := range doc.Keys {
		if k.KeyID == id {
			pub, _ := DecodePublicKeyText(k.PublicKey)
			updated, _ := time.Parse(time.RFC3339Nano, k.UpdatedAtUTC)
			return TrustedKey{PublicKey: pub, Identity: k.Identity, Status: k.Status, UpdatedAt: updated}, nil
		}
	}
	return TrustedKey{}, ErrUntrustedKey
}

type Keyring struct {
	Trust   *TrustStore
	Secrets SecretStore
}

func NewKeyring(trust *TrustStore, secrets SecretStore) (*Keyring, error) {
	if trust == nil || secrets == nil {
		return nil, ErrInvalid
	}
	return &Keyring{Trust: trust, Secrets: secrets}, nil
}
func NativeKeyring(trust *TrustStore) (*Keyring, error) { return NewKeyring(trust, credentials.New()) }

func newKey() (KeyRecord, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyRecord{}, nil, ErrUnavailable
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		clear(priv)
		return KeyRecord{}, nil, ErrUnavailable
	}
	now := utc(time.Now())
	h := sha256.Sum256(pub)
	return KeyRecord{KeyID: hex.EncodeToString(idBytes), PublicKey: PublicKeyText(pub), Fingerprint: hex.EncodeToString(h[:]), Status: "active", CreatedAtUTC: now, UpdatedAtUTC: now}, priv, nil
}
func secretID(id string) string { return "report-" + id }

// Generate creates an operator key in the native secret store and publishes
// only its public key in the local trust file. A failed trust write rolls back
// the secret-store item when possible.
func (k *Keyring) Generate(ctx context.Context, identity string) (KeyRecord, error) {
	if k == nil || k.Trust == nil || k.Secrets == nil || ctx == nil || !validIdentity(identity) {
		return KeyRecord{}, ErrInvalid
	}
	release, err := k.Trust.mutationLock()
	if err != nil {
		return KeyRecord{}, err
	}
	defer release()
	doc, err := k.Trust.read()
	if err != nil {
		return KeyRecord{}, err
	}
	record, priv, err := newKey()
	if err != nil {
		return KeyRecord{}, err
	}
	defer clear(priv)
	for _, existing := range doc.Keys {
		if existing.KeyID == record.KeyID {
			return KeyRecord{}, ErrUnavailable
		}
	}
	record.Identity = identity
	if err := k.Secrets.Set(ctx, secretID(record.KeyID), priv); err != nil {
		return KeyRecord{}, err
	}
	doc.Keys = append(doc.Keys, record)
	if err := k.Trust.save(doc); err != nil {
		_ = k.Secrets.Delete(ctx, secretID(record.KeyID))
		return KeyRecord{}, err
	}
	return record, nil
}

// Rotate retires the old public key for historical verification and creates
// a new signing key. Revoked keys can never be rotated back to active.
func (k *Keyring) Rotate(ctx context.Context, oldID string) (KeyRecord, error) {
	if k == nil || k.Trust == nil || k.Secrets == nil || ctx == nil || !validKeyID(oldID) {
		return KeyRecord{}, ErrInvalid
	}
	release, err := k.Trust.mutationLock()
	if err != nil {
		return KeyRecord{}, err
	}
	defer release()
	doc, err := k.Trust.read()
	if err != nil {
		return KeyRecord{}, err
	}
	index := -1
	for i, v := range doc.Keys {
		if v.KeyID == oldID {
			index = i
			break
		}
	}
	if index < 0 || doc.Keys[index].Status != "active" {
		return KeyRecord{}, ErrInvalid
	}
	record, priv, err := newKey()
	if err != nil {
		return KeyRecord{}, err
	}
	defer clear(priv)
	for _, existing := range doc.Keys {
		if existing.KeyID == record.KeyID {
			return KeyRecord{}, ErrUnavailable
		}
	}
	record.Identity = doc.Keys[index].Identity
	if err := k.Secrets.Set(ctx, secretID(record.KeyID), priv); err != nil {
		return KeyRecord{}, err
	}
	doc.Keys[index].Status = "retired"
	doc.Keys[index].UpdatedAtUTC = utc(time.Now())
	doc.Keys = append(doc.Keys, record)
	if err := k.Trust.save(doc); err != nil {
		_ = k.Secrets.Delete(ctx, secretID(record.KeyID))
		return KeyRecord{}, err
	}
	_ = k.Secrets.Delete(ctx, secretID(oldID))
	return record, nil
}

func (k *Keyring) Revoke(ctx context.Context, id string) error {
	if k == nil || k.Trust == nil || k.Secrets == nil || ctx == nil || !validKeyID(id) {
		return ErrInvalid
	}
	release, err := k.Trust.mutationLock()
	if err != nil {
		return err
	}
	defer release()
	doc, err := k.Trust.read()
	if err != nil {
		return err
	}
	index := -1
	for i, v := range doc.Keys {
		if v.KeyID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return ErrUntrustedKey
	}
	if doc.Keys[index].Status == "revoked" {
		return nil
	}
	doc.Keys[index].Status = "revoked"
	doc.Keys[index].UpdatedAtUTC = utc(time.Now())
	if err := k.Trust.save(doc); err != nil {
		return err
	}
	_ = k.Secrets.Delete(ctx, secretID(id))
	return nil
}

// ExportSigned retrieves a private key only for this operation, checks that it
// corresponds to the trusted public key and clears the returned copy afterward.
func (k *Keyring) ExportSigned(ctx context.Context, s Snapshot, destination, id string, opts ExportOptions) (Manifest, error) {
	if k == nil || k.Trust == nil || k.Secrets == nil || ctx == nil {
		return Manifest{}, ErrInvalid
	}
	release, err := k.Trust.mutationLock()
	if err != nil {
		return Manifest{}, err
	}
	defer release()
	doc, err := k.Trust.read()
	if err != nil {
		return Manifest{}, err
	}
	var trusted TrustedKey
	for _, entry := range doc.Keys {
		if entry.KeyID == id {
			pub, _ := DecodePublicKeyText(entry.PublicKey)
			trusted = TrustedKey{PublicKey: pub, Identity: entry.Identity, Status: entry.Status}
			break
		}
	}
	if trusted.Status != "active" {
		return Manifest{}, ErrUntrustedKey
	}
	priv, err := k.Secrets.Get(ctx, secretID(id))
	if err != nil {
		return Manifest{}, err
	}
	defer clear(priv)
	if len(priv) != ed25519.PrivateKeySize || !bytes.Equal(priv[32:], trusted.PublicKey) {
		return Manifest{}, ErrUntrustedKey
	}
	opts.Signer = &Signer{KeyID: id, Name: trusted.Identity, PrivateKey: priv}
	return Export(ctx, s, destination, opts)
}

type PublicDescriptor struct {
	KeyID       string `json:"key_id"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint_sha256"`
	Identity    string `json:"identity"`
	Status      string `json:"status"`
}

func (t *TrustStore) ExportPublic(id, path string) error {
	if t == nil || !filepath.IsAbs(path) || !validKeyID(id) {
		return ErrInvalid
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	doc, err := t.read()
	if err != nil {
		return err
	}
	for _, k := range doc.Keys {
		if k.KeyID == id {
			data, e := jsonv2.Marshal(PublicDescriptor{KeyID: k.KeyID, PublicKey: k.PublicKey, Fingerprint: k.Fingerprint, Identity: k.Identity, Status: k.Status})
			if e != nil {
				return ErrInvalid
			}
			return writeNewPrivateFile(path, data)
		}
	}
	return ErrUntrustedKey
}

// ImportTrusted requires an independently supplied SHA-256 fingerprint. Merely
// receiving a public-key file or bundle is not a trust decision.
func (t *TrustStore) ImportTrusted(path, expectedFingerprint string) (KeyRecord, error) {
	if t == nil || !filepath.IsAbs(path) || len(expectedFingerprint) != 64 {
		return KeyRecord{}, ErrInvalid
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 4096 {
		return KeyRecord{}, ErrInvalid
	}
	var p PublicDescriptor
	if jsonv2.Unmarshal(data, &p, jsonv2.RejectUnknownMembers(true)) != nil || p.Fingerprint != expectedFingerprint || !validKeyID(p.KeyID) || !validIdentity(p.Identity) || (p.Status != "active" && p.Status != "retired") {
		return KeyRecord{}, ErrInvalid
	}
	pub, err := DecodePublicKeyText(p.PublicKey)
	if err != nil {
		return KeyRecord{}, err
	}
	h := sha256.Sum256(pub)
	if hex.EncodeToString(h[:]) != expectedFingerprint {
		return KeyRecord{}, ErrInvalid
	}
	release, err := t.mutationLock()
	if err != nil {
		return KeyRecord{}, err
	}
	defer release()
	doc, err := t.read()
	if err != nil {
		return KeyRecord{}, err
	}
	for _, k := range doc.Keys {
		if k.KeyID == p.KeyID {
			return KeyRecord{}, ErrInvalid
		}
	}
	now := utc(time.Now())
	record := KeyRecord{KeyID: p.KeyID, PublicKey: p.PublicKey, Fingerprint: p.Fingerprint, Identity: p.Identity, Status: p.Status, CreatedAtUTC: now, UpdatedAtUTC: now}
	doc.Keys = append(doc.Keys, record)
	if err := t.save(doc); err != nil {
		return KeyRecord{}, err
	}
	return record, nil
}

func writeNewPrivateFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return ErrUnavailable
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return ErrUnavailable
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return ErrUnavailable
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return ErrUnavailable
	}
	return nil
}

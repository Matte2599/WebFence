package reporting

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Matte2599/WebFence/internal/signature"
)

const maxArtifactBytes = 4 << 20
const maxBundleBytes = 20 << 20

type Signer struct {
	KeyID, Name string
	PrivateKey  ed25519.PrivateKey
}
type ExportOptions struct {
	Signer       *Signer
	Offline      bool
	FreshnessTTL time.Duration
}

type Manifest struct {
	Schema         string         `json:"schema"`
	ReportID       string         `json:"report_id"`
	ProjectID      string         `json:"project_id"`
	RunID          string         `json:"run_id"`
	GeneratedAtUTC string         `json:"generated_at_utc"`
	SignerName     string         `json:"signer_name"`
	KeyID          string         `json:"key_id"`
	Languages      []string       `json:"languages"`
	Files          []ManifestFile `json:"files"`
}
type ManifestFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func canonicalManifest(m Manifest) ([]byte, error) {
	raw, err := jsonv2.Marshal(m)
	if err != nil {
		return nil, ErrInvalid
	}
	value := jsontext.Value(raw)
	if err := value.Canonicalize(); err != nil {
		return nil, ErrInvalid
	}
	if len(value) > signature.MaxPayloadBytes {
		return nil, ErrInvalid
	}
	return []byte(value), nil
}

func newReportID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", ErrUnavailable
	}
	return hex.EncodeToString(b), nil
}

// Export creates a new, private ZIP bundle. It refuses to replace an existing
// path. Signed and unsigned bundles are visibly different in both reports.
func Export(ctx context.Context, s Snapshot, destination string, opts ExportOptions) (Manifest, error) {
	if ctx == nil || !filepath.IsAbs(destination) || filepath.Base(destination) == "." {
		return Manifest{}, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return Manifest{}, err
	}
	if opts.FreshnessTTL <= 0 {
		opts.FreshnessTTL = 24 * time.Hour
	}
	signed := opts.Signer != nil
	name, kid := "", ""
	if signed {
		name, kid = opts.Signer.Name, opts.Signer.KeyID
		if !validIdentity(name) || len(opts.Signer.PrivateKey) != ed25519.PrivateKeySize || !validKeyID(kid) {
			return Manifest{}, ErrInvalid
		}
	}
	id, err := newReportID()
	if err != nil {
		return Manifest{}, err
	}
	now := time.Now().UTC()
	m := Manifest{Schema: "webfence-manifest-v1", ReportID: id, ProjectID: s.ProjectID, RunID: s.Run.ID,
		GeneratedAtUTC: utc(now), SignerName: name, KeyID: kid, Languages: []string{"it", "en"}}
	files := make(map[string][]byte, 6)
	for _, lang := range m.Languages {
		report, e := buildReport(s, id, lang, now, signed, name, opts.Offline, opts.FreshnessTTL)
		if e != nil {
			return Manifest{}, e
		}
		jsonData, e := renderJSON(report)
		if e != nil {
			return Manifest{}, ErrInvalid
		}
		htmlData, e := renderHTML(report)
		if e != nil {
			return Manifest{}, ErrInvalid
		}
		files["report-"+lang+".json"] = jsonData
		files["report-"+lang+".html"] = htmlData
	}
	for _, name := range []string{"report-it.json", "report-it.html", "report-en.json", "report-en.html"} {
		data := files[name]
		if len(data) < 1 || len(data) > maxArtifactBytes {
			return Manifest{}, ErrInvalid
		}
		h := sha256.Sum256(data)
		m.Files = append(m.Files, ManifestFile{Path: name, Size: int64(len(data)), SHA256: hex.EncodeToString(h[:])})
	}
	canonical, err := canonicalManifest(m)
	if err != nil {
		return Manifest{}, err
	}
	files["manifest.json"] = canonical
	if signed {
		envelope, e := signature.Sign(canonical, kid, opts.Signer.PrivateKey)
		if e != nil {
			return Manifest{}, ErrInvalid
		}
		files["manifest.jws"] = envelope
	}
	if err := writeBundle(ctx, destination, files); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func writeBundle(ctx context.Context, destination string, files map[string][]byte) error {
	_, err := os.Lstat(destination)
	if err == nil {
		return ErrInvalid
	}
	if !errors.Is(err, os.ErrNotExist) {
		return ErrUnavailable
	}
	parent := filepath.Dir(destination)
	tmp, err := os.CreateTemp(parent, ".webfence-report-*.tmp")
	if err != nil {
		return ErrUnavailable
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return ErrUnavailable
	}
	zw := zip.NewWriter(tmp)
	for _, name := range []string{"report-it.json", "report-it.html", "report-en.json", "report-en.html", "manifest.json", "manifest.jws"} {
		data, ok := files[name]
		if !ok {
			continue
		}
		if err := ctx.Err(); err != nil {
			_ = zw.Close()
			_ = tmp.Close()
			return err
		}
		h := &zip.FileHeader{Name: name, Method: zip.Store}
		h.SetMode(0o600)
		h.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
		w, e := zw.CreateHeader(h)
		if e != nil {
			_ = zw.Close()
			_ = tmp.Close()
			return ErrUnavailable
		}
		if _, e = w.Write(data); e != nil {
			_ = zw.Close()
			_ = tmp.Close()
			return ErrUnavailable
		}
	}
	if err := zw.Close(); err != nil {
		_ = tmp.Close()
		return ErrUnavailable
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return ErrUnavailable
	}
	if err := tmp.Close(); err != nil {
		return ErrUnavailable
	}
	info, err := os.Stat(tmpPath)
	if err != nil || info.Size() > maxBundleBytes {
		return ErrInvalid
	}
	// Same-directory hard link is an atomic no-clobber publication on local
	// filesystems. Do not use rename here: it can replace an existing report.
	if err := os.Link(tmpPath, destination); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrInvalid
		}
		return ErrUnavailable
	}
	return nil
}

type TrustedKey struct {
	PublicKey        ed25519.PublicKey
	Identity, Status string
	UpdatedAt        time.Time
}
type TrustLookup interface {
	Lookup(string) (TrustedKey, error)
}
type Verification struct {
	ReportID, ProjectID, RunID, KeyID, DeclaredSigner, TrustedIdentity string
	TrustStatus                                                        string
	TrustUpdatedAt                                                     time.Time
	Languages                                                          []string
	Files                                                              int
}

// Verify accepts trust only from the supplied out-of-bundle lookup. It checks
// the exact JWS profile, canonical manifest, every listed artifact and rejects
// additions. It does not claim trusted time or globally current revocation.
func Verify(path string, trust TrustLookup) (Verification, error) {
	if !filepath.IsAbs(path) || trust == nil {
		return Verification{}, ErrInvalid
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxBundleBytes || info.Size() < 1 {
		return Verification{}, ErrInvalidBundle
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return Verification{}, ErrInvalidBundle
	}
	defer zr.Close()
	if len(zr.File) < 5 || len(zr.File) > 6 {
		return Verification{}, ErrInvalidBundle
	}
	files := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		_, duplicate := files[f.Name]
		if !allowedBundleName(f.Name) || duplicate || f.UncompressedSize64 > maxArtifactBytes || f.CompressedSize64 > maxArtifactBytes || !f.Mode().IsRegular() {
			return Verification{}, ErrInvalidBundle
		}
		r, e := f.Open()
		if e != nil {
			return Verification{}, ErrInvalidBundle
		}
		data, e := io.ReadAll(io.LimitReader(r, maxArtifactBytes+1))
		closeErr := r.Close()
		if e != nil || closeErr != nil || len(data) > maxArtifactBytes || uint64(len(data)) != f.UncompressedSize64 {
			return Verification{}, ErrInvalidBundle
		}
		files[f.Name] = data
	}
	manifestBytes, ok := files["manifest.json"]
	if !ok {
		return Verification{}, ErrInvalidBundle
	}
	envelope, ok := files["manifest.jws"]
	if !ok {
		return Verification{}, ErrUnsigned
	}
	var m Manifest
	if jsonv2.Unmarshal(manifestBytes, &m, jsonv2.RejectUnknownMembers(true)) != nil || !validManifest(m) {
		return Verification{}, ErrInvalidBundle
	}
	canonical, e := canonicalManifest(m)
	if e != nil || !bytes.Equal(canonical, manifestBytes) {
		return Verification{}, ErrInvalidBundle
	}
	key, e := trust.Lookup(m.KeyID)
	if e != nil || key.Status == "unknown" || len(key.PublicKey) != ed25519.PublicKeySize {
		return Verification{}, ErrUntrustedKey
	}
	if key.Status != "active" && key.Status != "retired" && key.Status != "revoked" {
		return Verification{}, ErrUntrustedKey
	}
	payload, e := signature.Verify(envelope, m.KeyID, key.PublicKey)
	if e != nil || !bytes.Equal(payload, manifestBytes) {
		return Verification{}, ErrInvalidSignature
	}
	if key.Status == "revoked" {
		return Verification{}, ErrRevokedKey
	}
	expected := map[string]bool{"manifest.json": true, "manifest.jws": true}
	for _, entry := range m.Files {
		data, ok := files[entry.Path]
		if !ok || expected[entry.Path] || entry.Size != int64(len(data)) {
			return Verification{}, ErrInvalidBundle
		}
		h := sha256.Sum256(data)
		if hex.EncodeToString(h[:]) != entry.SHA256 {
			return Verification{}, ErrInvalidBundle
		}
		expected[entry.Path] = true
	}
	if len(files) != len(expected) {
		return Verification{}, ErrInvalidBundle
	}
	for _, lang := range []string{"it", "en"} {
		var report Report
		if jsonv2.Unmarshal(files["report-"+lang+".json"], &report, jsonv2.RejectUnknownMembers(true)) != nil ||
			report.Schema != "webfence-report-v1" || report.ReportID != m.ReportID || report.Project.ID != m.ProjectID || report.Run.ID != m.RunID ||
			report.Language != lang || report.SignatureStatus != "signed" || report.GeneratedAt != m.GeneratedAtUTC || report.SignerName != m.SignerName {
			return Verification{}, ErrInvalidBundle
		}
	}
	return Verification{ReportID: m.ReportID, ProjectID: m.ProjectID, RunID: m.RunID, KeyID: m.KeyID, DeclaredSigner: m.SignerName,
		TrustedIdentity: key.Identity, TrustStatus: key.Status, TrustUpdatedAt: key.UpdatedAt, Languages: append([]string(nil), m.Languages...), Files: len(m.Files)}, nil
}

func allowedBundleName(name string) bool {
	switch name {
	case "report-it.json", "report-it.html", "report-en.json", "report-en.html", "manifest.json", "manifest.jws":
		return true
	}
	return false
}

func validManifest(m Manifest) bool {
	if m.Schema != "webfence-manifest-v1" || len(m.ReportID) != 32 || len(m.ProjectID) == 0 || len(m.RunID) != 32 ||
		len(m.KeyID) < 1 || len(m.KeyID) > 64 || len(m.SignerName) < 1 || len(m.SignerName) > 128 ||
		len(m.Languages) != 2 || m.Languages[0] != "it" || m.Languages[1] != "en" || len(m.Files) != 4 {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, m.GeneratedAtUTC); err != nil {
		return false
	}
	for i, f := range m.Files {
		if f.Path != []string{"report-it.json", "report-it.html", "report-en.json", "report-en.html"}[i] || f.Size < 1 || f.Size > maxArtifactBytes || len(f.SHA256) != 64 {
			return false
		}
		if _, err := hex.DecodeString(f.SHA256); err != nil {
			return false
		}
	}
	return true
}

// PublicKeyText is a portable representation for an out-of-band trust file.
func PublicKeyText(key ed25519.PublicKey) string { return base64.RawURLEncoding.EncodeToString(key) }
func DecodePublicKeyText(value string) (ed25519.PublicKey, error) {
	b, e := base64.RawURLEncoding.Strict().DecodeString(value)
	if e != nil || len(b) != ed25519.PublicKeySize || base64.RawURLEncoding.EncodeToString(b) != value {
		return nil, ErrInvalid
	}
	return ed25519.PublicKey(b), nil
}

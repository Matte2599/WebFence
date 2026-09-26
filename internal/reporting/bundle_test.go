package reporting

import (
	"archive/zip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/intelligence"
	"github.com/Matte2599/WebFence/internal/storage"
)

type fixedTrust struct {
	id  string
	key TrustedKey
}

func (t fixedTrust) Lookup(id string) (TrustedKey, error) {
	if id != t.id {
		return TrustedKey{}, ErrUntrustedKey
	}
	return t.key, nil
}

func fixtureSnapshot() Snapshot {
	now := time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)
	return Snapshot{ProjectID: "synthetic-report", ProjectName: "<script>alert(1)</script>", TargetOwner: "Synthetic owner",
		Origins: []string{"https://owned.invalid:443"}, AuthorizationRevision: 1, AuthorizationExpiresAt: now.Add(24 * time.Hour),
		Run: storage.ScanRun{ID: "0123456789abcdef0123456789abcdef", ProjectID: "synthetic-report", AuthorizationRevision: 1,
			Mode: "loopback", State: "complete", StartedAt: now, FinishedAt: now.Add(time.Minute), PlannedSeeds: 1, CompletedVisits: 1, RequestsUsed: 1,
			Visits: []storage.ScanVisit{{VisitIndex: 0, Depth: 0, StatusCode: 200, RuleID: "HTTP-XCTO-001", RuleRevision: 1,
				Outcome: "observed", EvidenceCode: "nosniff_absent", DiscoveryStatus: "observed", DiscoveryReason: "html_observed", Links: 1}}}}
}

func readZip(t *testing.T, path string) map[string][]byte {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	files := map[string][]byte{}
	for _, f := range r.File {
		opened, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(opened)
		_ = opened.Close()
		if e != nil {
			t.Fatal(e)
		}
		files[f.Name] = data
	}
	return files
}

func repack(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for name, data := range files {
		h := &zip.FileHeader{Name: name, Method: zip.Store}
		h.SetMode(0o600)
		entry, e := w.CreateHeader(h)
		if e != nil {
			t.Fatal(e)
		}
		if _, e := entry.Write(data); e != nil {
			t.Fatal(e)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSignedBundleBilingualIntegrityAndTrust(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "signed.wfr")
	m, err := Export(context.Background(), fixtureSnapshot(), path, ExportOptions{Signer: &Signer{KeyID: "operator-1", Name: "Test operator", PrivateKey: priv}})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 4 {
		t.Fatal(m)
	}
	files := readZip(t, path)
	if len(files) != 6 || !strings.Contains(string(files["report-it.html"]), "Report WebFence") || !strings.Contains(string(files["report-en.html"]), "WebFence report") {
		t.Fatal("missing bilingual report")
	}
	if strings.Contains(string(files["report-en.html"]), "<script>") || !strings.Contains(string(files["report-en.html"]), "&lt;script&gt;") {
		t.Fatal("HTML escaped content missing")
	}
	trust := fixedTrust{"operator-1", TrustedKey{PublicKey: pub, Identity: "Trusted operator", Status: "active", UpdatedAt: time.Now()}}
	result, err := Verify(path, trust)
	if err != nil || result.Files != 4 || result.TrustedIdentity != "Trusted operator" {
		t.Fatalf("verify: %+v %v", result, err)
	}
	if _, err := Export(context.Background(), fixtureSnapshot(), path, ExportOptions{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("overwrite accepted: %v", err)
	}
	if _, err := Verify(path, fixedTrust{"other", trust.key}); !errors.Is(err, ErrUntrustedKey) {
		t.Fatalf("unknown key: %v", err)
	}
	badKey, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := Verify(path, fixedTrust{"operator-1", TrustedKey{PublicKey: badKey, Status: "active"}}); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("wrong key: %v", err)
	}
	if _, err := Verify(path, fixedTrust{"operator-1", TrustedKey{PublicKey: pub, Status: "revoked"}}); !errors.Is(err, ErrRevokedKey) {
		t.Fatalf("revoked key: %v", err)
	}
	files["report-en.html"] = []byte("tampered")
	tampered := filepath.Join(dir, "tampered.wfr")
	repack(t, tampered, files)
	if _, err := Verify(tampered, trust); !errors.Is(err, ErrInvalidBundle) {
		t.Fatalf("tampered artifact: %v", err)
	}
	files = readZip(t, path)
	files["extra.txt"] = []byte("unlisted")
	extra := filepath.Join(dir, "extra.wfr")
	repack(t, extra, files)
	if _, err := Verify(extra, trust); !errors.Is(err, ErrInvalidBundle) {
		t.Fatalf("extra artifact: %v", err)
	}
	files = readZip(t, path)
	files["manifest.json"] = append([]byte(nil), files["manifest.json"]...)
	files["manifest.json"][0] = '['
	manifestTamper := filepath.Join(dir, "manifest-tamper.wfr")
	repack(t, manifestTamper, files)
	if _, err := Verify(manifestTamper, trust); !errors.Is(err, ErrInvalidBundle) {
		t.Fatalf("manifest tamper: %v", err)
	}
	files = readZip(t, path)
	files["manifest.jws"] = append([]byte(nil), files["manifest.jws"]...)
	files["manifest.jws"][len(files["manifest.jws"])-1] ^= 1
	signatureTamper := filepath.Join(dir, "signature-tamper.wfr")
	repack(t, signatureTamper, files)
	if _, err := Verify(signatureTamper, trust); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("signature tamper: %v", err)
	}
}

func TestUnsignedBundleIsExplicit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unsigned.wfr")
	if _, err := Export(context.Background(), fixtureSnapshot(), path, ExportOptions{}); err != nil {
		t.Fatal(err)
	}
	files := readZip(t, path)
	if len(files) != 5 || strings.Contains(string(files["report-en.json"]), `"signature_status":"signed"`) || !strings.Contains(string(files["report-en.json"]), `"signature_status":"unsigned"`) {
		t.Fatal("unsigned label absent")
	}
	if _, err := Verify(path, fixedTrust{}); !errors.Is(err, ErrUnsigned) {
		t.Fatalf("unsigned accepted: %v", err)
	}
}

func TestOfflineReportPreservesStaleFeedState(t *testing.T) {
	s := fixtureSnapshot()
	s.Intelligence = []intelligence.Snapshot{{Source: "nvd", LastSuccess: time.Now().Add(-72 * time.Hour), Records: 4}}
	r, err := buildReport(s, "synthetic", "en", time.Now(), false, "", true, 24*time.Hour)
	if err != nil || len(r.Intelligence) != 2 || r.Intelligence[0].Status != "offline" || r.Intelligence[0].Freshness != "stale" || r.Intelligence[1].Status != "unavailable" {
		t.Fatalf("feed state: %+v %v", r.Intelligence, err)
	}
}

func TestUnexecutedSeedsExcludeDiscoveredVisits(t *testing.T) {
	s := fixtureSnapshot()
	s.Run.PlannedSeeds = 2
	discovered := s.Run.Visits[0]
	discovered.VisitIndex = 1
	discovered.Depth = 1
	s.Run.Visits = append(s.Run.Visits, discovered)
	s.Run.CompletedVisits = 2
	r, err := buildReport(s, "synthetic", "en", time.Now(), false, "", true, time.Hour)
	if err != nil || r.Coverage.UnexecutedSeedMinimum != 1 {
		t.Fatalf("seed coverage: %+v %v", r.Coverage, err)
	}
}

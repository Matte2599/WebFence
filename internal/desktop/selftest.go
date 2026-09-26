package desktop

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Matte2599/WebFence/internal/demo"
	"github.com/Matte2599/WebFence/internal/intelligence"
	"github.com/Matte2599/WebFence/internal/preferences"
	"github.com/Matte2599/WebFence/internal/reporting"
	qt "github.com/mappu/miqt/qt6"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type selfTestSecrets struct {
	mu   sync.Mutex
	data map[string][]byte
}

func (s *selfTestSecrets) Get(_ context.Context, id string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	if !ok {
		return nil, reporting.ErrUntrustedKey
	}
	return append([]byte(nil), v...), nil
}
func (s *selfTestSecrets) Set(_ context.Context, id string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = append([]byte(nil), value...)
	return nil
}
func (s *selfTestSecrets) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

// selfTest exercises the real Qt model/widgets on their owner OS thread.
// It is not a screen-reader test. It never writes to the system clipboard.
func selfTest(w *workspace) int {
	// A headless platform has no screen reader to activate Qt accessibility.
	// Activation is required for model-reset cache invalidation.
	wasActive := qt.QAccessible_IsActive()
	qt.QAccessible_SetActive(true)
	defer qt.QAccessible_SetActive(wasActive)
	fmt.Printf("qt_accessibility_active=%t\n", qt.QAccessible_IsActive())
	failures := 0
	check := func(ok bool, name string) {
		if !ok {
			fmt.Println("FAIL", name)
			failures++
		} else {
			fmt.Println("PASS", name)
		}
	}
	pressTab := func(backward bool) {
		key, modifiers := qt.Key_Tab, qt.NoModifier
		if backward {
			key, modifiers = qt.Key_Backtab, qt.ShiftModifier
		}
		for _, kind := range []qt.QEvent__Type{qt.QEvent__KeyPress, qt.QEvent__KeyRelease} {
			if focused := qt.QApplication_FocusWidget(); focused != nil {
				event := qt.NewQKeyEvent(kind, int(key), modifiers)
				qt.QCoreApplication_SendEvent(focused.QObject, event.QEvent)
				event.Delete()
			}
		}
	}
	// These are Qt interfaces, not the OS accessibility bridge or a screen reader.
	checkAccessible := func(stage string) {
		qt.QCoreApplication_ProcessEvents()
		iface := qt.QAccessible_QueryAccessibleInterface(w.table.QObject)
		if iface == nil || iface.TableInterface() == nil {
			check(false, "Qt accessible table: "+stage)
			return
		}
		table := iface.TableInterface()
		if !qt.QAccessible_IsActive() {
			// Qt 6.4 offscreen has no active platform bridge: SetActive only
			// notifies observers. Exercise current interface data with an explicit
			// cache reset, never claim to test automatic OS notifications here.
			event := qt.NewQAccessibleTableModelChangeEvent(w.table.QObject, qt.QAccessibleTableModelChangeEvent__ModelReset)
			table.ModelChange(event)
			event.Delete()
			fmt.Println("NOTE explicit accessible cache reset (no active bridge):", stage)
		}
		check(table.RowCount() == len(w.visible) && table.ColumnCount() == 4, "Qt accessible dimensions: "+stage)
		if len(w.visible) > 0 {
			first := table.CellAt(0, 0)
			check(first != nil && first.Text(qt.QAccessible__Name) == w.visible[0].ID, "Qt accessible first ID: "+stage)
			row := len(w.visible) - 1
			cell := table.CellAt(row, 0)
			check(cell != nil && cell.Text(qt.QAccessible__Name) == w.visible[row].ID, "Qt accessible last ID: "+stage)
		}
		// Interfaces/cells are owned by Qt's accessibility cache, never delete them here.
	}
	check(len(w.visible) == 0, "initial empty state")
	check(w.window.IsActiveWindow(), "test window activated before keyboard checks")
	checkAccessible("empty")
	start := time.Now()
	w.load.Click()
	qt.QCoreApplication_ProcessEvents()
	fmt.Printf("load_10000_ms=%.3f\n", float64(time.Since(start).Microseconds())/1000)
	check(len(w.visible) == 10000, "10000 rows loaded")
	checkAccessible("loaded")
	proxyResets, removedRows, insertedRows := 0, 0, 0
	w.proxy.OnModelReset(func() { proxyResets++ })
	w.proxy.OnRowsRemoved(func(parent *qt.QModelIndex, first, last int) { removedRows += last - first + 1 })
	w.proxy.OnRowsInserted(func(parent *qt.QModelIndex, first, last int) { insertedRows += last - first + 1 })
	parent := qt.NewQModelIndex()
	defer parent.Delete()
	// Rendering/accessibility can request the same model data repeatedly.
	for i := 0; i < 10000; i++ {
		w.model.Data(w.model.Index(i, 0, parent), int(qt.DisplayRole))
	}
	cached := len(w.variants)
	for i := 0; i < 10000; i++ {
		w.model.Data(w.model.Index(i, 0, parent), int(qt.DisplayRole))
	}
	check(len(w.variants) == cached, "repeated model reads do not grow owned variant cache")
	w.table.SelectRow(9999)
	original := w.evidence.ToPlainText()
	check(w.selectedID == "DEMO-10000" && strings.Contains(original, "<script>"), "last row and inert evidence")
	w.evidenceAction.Trigger()
	check(w.evidence.IsVisible() && w.evidence.HasFocus(), "evidence action reveals and focuses details")
	cursor := w.evidence.TextCursor()
	cursor.SetPosition(0)
	cursor.SetPosition2(10, qt.QTextCursor__KeepAnchor)
	w.evidence.SetTextCursor(cursor)
	selectedText := cursor.SelectedText()
	runtime.SetFinalizer(cursor, nil) // Release QTextCursor on its owner thread.
	cursor.Delete()
	resets := 0
	w.model.OnModelReset(func() { resets++ })
	w.englishAction.Trigger()
	cursor = w.evidence.TextCursor()
	check(resets == 0 && cursor.SelectedText() == selectedText, "translation preserves model and evidence text selection")
	runtime.SetFinalizer(cursor, nil) // Release QTextCursor on its owner thread.
	cursor.Delete()
	checkAccessible("translated")
	saved, err := preferences.Load(w.preferencePath, "it")
	check(err == nil && saved == "en", "language action persists English")
	w.italianAction.Trigger()
	saved, err = preferences.Load(w.preferencePath, "en")
	check(err == nil && saved == "it", "language action persists Italian")
	check(w.selectedID == "DEMO-10000" && original == w.evidence.ToPlainText(), "language preserves identity and evidence")
	w.advancedAction.Trigger()
	check(!w.evidence.IsVisible() && w.advanced.HasFocus() && !w.advanced.IsChecked(), "hiding details returns focus to advanced control")
	w.advancedAction.Trigger()
	check(w.evidence.IsVisible() && w.advanced.IsChecked(), "advanced menu and checkbox stay synchronized")
	// A 200% desktop can leave roughly this much logical workspace. Check the
	// actual viewport, including space consumed by native non-overlay scrollbars.
	originalWidth, originalHeight := w.window.Width(), w.window.Height()
	w.window.Resize(756, 430)
	qt.QCoreApplication_ProcessEvents()
	fmt.Printf("compact_table_height=%d minimum=%d viewport=%d rows=%d header=%d scrollbar=%d frame=%d\n", w.table.Height(), w.table.MinimumHeight(), w.table.Viewport().Height(), w.table.RowHeight(0)+w.table.RowHeight(1), w.table.HorizontalHeader().Height(), w.table.HorizontalScrollBar().Height(), w.table.FrameWidth())
	check(w.table.Viewport().Height() >= w.table.RowHeight(0)+w.table.RowHeight(1), "compact window retains two complete result rows")
	// A taller native header must not steal the space reserved for result rows.
	// This also exercises geometry changes after the initial style polish.
	header := w.table.HorizontalHeader()
	originalHeaderMinimum := header.MinimumHeight()
	header.SetMinimumHeight(header.Height() + 24)
	qt.QCoreApplication_ProcessEvents()
	check(w.table.Viewport().Height() >= w.table.RowHeight(0)+w.table.RowHeight(1), "header geometry change preserves two complete rows")
	header.SetMinimumHeight(originalHeaderMinimum)
	qt.QCoreApplication_ProcessEvents()
	metrics := w.evidence.FontMetrics()
	runtime.SetFinalizer(metrics, nil)
	check(w.evidence.Viewport().Height() >= 2*metrics.Height(), "compact window retains readable evidence viewport")
	metrics.Delete()
	w.window.Resize(originalWidth, originalHeight)
	qt.QCoreApplication_ProcessEvents()
	copied := ""
	w.copyEvidence = func(value string) { copied = value }
	w.copy.Click()
	check(copied == original, "explicit copy preserves evidence")
	path := w.preferencePath
	w.preferencePath = filepath.Dir(path)
	w.englishAction.Trigger()
	check(w.locale == "en" && w.notice.IsVisible(), "save error remains visible without blocking language change")
	w.preferencePath = path
	w.italianAction.Trigger()
	check(!w.notice.IsVisible(), "successful save clears preference warning")
	w.severity.SetCurrentIndex(1)
	check(len(w.visible) == 3334 && w.selectedID == "DEMO-10000", "severity preserves matching selection")
	start = time.Now()
	w.search.SetText("DEMO-10000")
	qt.QCoreApplication_ProcessEvents()
	fmt.Printf("filter_last_ms=%.3f\n", float64(time.Since(start).Microseconds())/1000)
	check(len(w.visible) == 1 && w.selectedID == "DEMO-10000", "filter last row")
	checkAccessible("filtered")
	w.searchAction.Trigger()
	check(w.search.HasFocus() && w.search.SelectedText() == "DEMO-10000", "search action focuses and selects query")
	w.resultsAction.Trigger()
	check(w.table.HasFocus() && w.selectedID == "DEMO-10000", "results action preserves matching selection")
	w.search.SetText("missing")
	check(len(w.visible) == 0 && w.selectedID == "" && w.evidence.ToPlainText() == "" && !w.copy.IsEnabled(), "no stale evidence on empty results")
	w.resultsAction.Trigger()
	check(w.selectedID == "" && w.table.HasFocus(), "empty results focus does not select stale data")
	checkAccessible("no matches")
	w.search.SetText("")
	w.severity.SetCurrentIndex(0)
	w.resultsAction.Trigger()
	check(w.selectedID == "DEMO-00001", "results action selects first row when none selected")
	checkAccessible("restored")
	check(proxyResets == 0 && removedRows > 0 && insertedRows > 0, "filter changes emit row deltas without model reset")
	pressTab(false)
	check(w.summary.HasFocus(), "Tab leaves results for summary")
	pressTab(false)
	check(w.evidence.HasFocus(), "Tab reaches visible evidence")
	pressTab(false)
	check(w.copy.HasFocus(), "Tab leaves evidence for copy")
	pressTab(true)
	check(w.evidence.HasFocus(), "Shift+Tab returns to evidence")
	w.advancedAction.Trigger()
	check(w.advanced.HasFocus() && !w.evidence.IsVisible(), "hide details after keyboard navigation")
	w.resultsAction.Trigger()
	pressTab(false)
	pressTab(false)
	check(w.load.HasFocus(), "Tab skips hidden evidence and copy")
	long := strings.Repeat(demo.Records()[0].Evidence(), 2048)
	w.evidence.SetPlainText(long)
	qt.QCoreApplication_ProcessEvents()
	check(w.evidence.ToPlainText() == long, "long evidence exact text roundtrip")
	fmt.Printf("long_evidence_bytes=%d\n", len(long))
	w.clear.Click()
	check(len(w.visible) == 0 && w.evidence.ToPlainText() == "", "clear dataset")
	checkAccessible("cleared")
	// The native M1 dialog creates a declared project, scans an owned loopback
	// fixture, and renders only persisted redacted observations.
	if w.scan != nil && w.scan.store != nil {
		var hits atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			hits.Add(1)
			writer.Header().Set("Content-Type", "text/html")
			_, _ = writer.Write([]byte("<html><body>synthetic</body></html>"))
		}))
		defer server.Close()
		u := w.scan
		u.show()
		u.id.SetText("m1-native-selftest")
		u.name.SetText("Synthetic owned lab")
		u.owner.SetText("Test fixture")
		u.reference.SetText("local self-test")
		u.origin.SetText(server.URL)
		u.seed.SetText(server.URL + "/")
		u.confirmed.SetChecked(true)
		u.create.Click()
		check(u.selectedProjectID() == "m1-native-selftest", "M1 native project creation")
		u.start.Click()
		deadline := time.Now().Add(12 * time.Second)
		for u.done != nil && time.Now().Before(deadline) {
			qt.QCoreApplication_ProcessEvents()
			time.Sleep(10 * time.Millisecond)
		}
		qt.QCoreApplication_ProcessEvents()
		check(u.done == nil && hits.Load() == 1 && strings.Contains(u.results.ToPlainText(), "nosniff_absent"), "M1 native owned-lab scan and result")
		check(!strings.Contains(u.results.ToPlainText(), server.URL), "M1 native result redaction")
		m := u.m2
		m.show()
		m.keys, _ = reporting.NewKeyring(m.trust, &selfTestSecrets{data: map[string][]byte{}})
		m.confirmKeyChange = func(string, string) bool { return true }
		m.refreshKeys()
		waitM2 := func() {
			deadline := time.Now().Add(12 * time.Second)
			for m.result != nil && time.Now().Before(deadline) {
				qt.QCoreApplication_ProcessEvents()
				time.Sleep(10 * time.Millisecond)
			}
			qt.QCoreApplication_ProcessEvents()
			check(m.result == nil, "M2 background operation finished")
		}
		captureM2 := func(label string) {
			folder := os.Getenv("WEBFENCE_UI_SNAPSHOT_DIR")
			if folder == "" {
				return
			}
			if err := os.MkdirAll(folder, 0o700); err != nil {
				check(false, "M2 snapshot directory: "+label)
				return
			}
			save := func(suffix string) {
				qt.QCoreApplication_ProcessEvents()
				pixmap := m.dialog.Grab()
				ok := pixmap.Save(filepath.Join(folder, label+suffix+".png"))
				runtime.SetFinalizer(pixmap, nil)
				pixmap.Delete()
				check(ok, "M2 visual snapshot: "+label+suffix)
			}
			save("")
			bar := m.pages[m.tabs.CurrentIndex()].VerticalScrollBar()
			if bar.Maximum() > 0 {
				old := bar.Value()
				bar.SetValue(bar.Maximum())
				save("-bottom")
				bar.SetValue(old)
			}
		}
		check(m.tabs.Count() == 3 && m.assessmentView.Height() >= 100, "M2 three-tab layout and assessment space")
		feed := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"cveMetadata":{"cveId":"CVE-2026-1000","state":"PUBLISHED","datePublished":"2026-09-01T00:00:00Z","dateUpdated":"2026-09-25T00:00:00Z"},"containers":{"cna":{"affected":[{"vendor":"example","product":"widget","versions":[{"version":"1.0.0","lessThan":"2.0.0","versionType":"semver","status":"affected"}]}]}}}`))
		}))
		_, feedErr := m.cache.RefreshCVE(context.Background(), &intelligence.CVEClient{Endpoint: feed.URL + "/"}, []string{"CVE-2026-1000"})
		feed.Close()
		m.refreshCache()
		m.cveID.SetText("CVE-2026-1000")
		m.source.SetCurrentIndex(1)
		m.vendor.SetText("example")
		m.product.SetText("widget")
		m.version.SetText("1.5.0")
		m.evidence.SetText("inventory:synthetic")
		m.match.Click()
		waitM2()
		assessment, assessed := m.assessments["cve:CVE-2026-1000"]
		check(feedErr == nil && assessed && assessment.Status == "applicable", "M2 native explicit cached CVE assessment")
		m.verificationConfirmed.SetChecked(true)
		m.verificationEvidence.SetText("review:synthetic")
		m.match.Click()
		waitM2()
		check(m.assessments["cve:CVE-2026-1000"].Status == "verified", "M2 native verified attestation")
		m.verificationConfirmed.SetChecked(false)
		m.verificationEvidence.SetText("")
		m.backportConfirmed.SetChecked(true)
		m.backportAdvisory.SetText("https://vendor.example.invalid/advisory/1")
		m.backportEvidence.SetText("advisory:synthetic")
		m.match.Click()
		waitM2()
		check(m.assessments["cve:CVE-2026-1000"].Status == "not_applicable", "M2 native backport attestation")
		m.verificationConfirmed.SetChecked(true)
		m.verificationEvidence.SetText("review:synthetic")
		m.match.Click()
		check(m.result == nil && m.status.Text() == m.tr("m2_attestation_conflict"), "M2 conflicting attestations blocked")
		m.backportConfirmed.SetChecked(false)
		m.backportAdvisory.SetText("")
		m.backportEvidence.SetText("")
		m.match.Click()
		waitM2()
		check(m.assessments["cve:CVE-2026-1000"].Status == "verified", "M2 verified status restored")
		m.tabs.SetCurrentIndex(1)
		captureM2("it-attested")
		m.source.SetCurrentIndex(0)
		check(!m.verificationConfirmed.IsChecked() && m.verificationEvidence.Text() == "", "M2 source change clears stale attestation")
		m.part.SetCurrentIndex(0)
		m.match.Click()
		check(m.result == nil && m.status.Text() == m.tr("m2_part_required"), "M2 NVD requires explicit CPE part")
		nvdFeed := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"format":"NVD_CVE","version":"2.0","resultsPerPage":1,"startIndex":0,"totalResults":1,"vulnerabilities":[{"cve":{"id":"CVE-2026-1001","vulnStatus":"Analyzed","published":"2026-09-01T00:00:00Z","lastModified":"2026-09-25T00:00:00Z","configurations":[{"nodes":[{"operator":"OR","cpeMatch":[{"vulnerable":true,"criteria":"cpe:2.3:a:example:widget:*:*:*:*:*:*:*:*","versionStartIncluding":"1.0","versionEndExcluding":"2.0"}]}]}]}}]}`))
		}))
		now := time.Now().UTC()
		_, nvdErr := m.cache.SyncNVD(context.Background(), &intelligence.NVDClient{Endpoint: nvdFeed.URL}, now.Add(-time.Hour), now)
		nvdFeed.Close()
		m.refreshCache()
		m.cveID.SetText("CVE-2026-1001")
		check(!m.backportConfirmed.IsChecked() && !m.verificationConfirmed.IsChecked(), "M2 CVE change keeps attestations clear")
		m.version.SetText("1.5")
		m.part.SetCurrentIndex(1)
		m.match.Click()
		waitM2()
		check(nvdErr == nil && m.assessments["nvd:CVE-2026-1001"].Status == "applicable", "M2 native NVD CPE-part assessment")
		m.cpe23.SetText("cpe:2.3:a:example:widget:1.5:*:*:*:*:*:*:*")
		check(!m.vendor.IsEnabled() && !m.part.IsEnabled(), "M2 full CPE disables separate identity fields")
		m.match.Click()
		waitM2()
		check(m.assessments["nvd:CVE-2026-1001"].Status == "applicable", "M2 native full-CPE assessment")
		m.cpe23.SetText("")
		check(m.vendor.IsEnabled() && m.part.IsEnabled(), "M2 separate identity fields restored")
		m.backportConfirmed.SetChecked(true)
		m.backportAdvisory.SetText("https://vendor.example.invalid/advisory/2")
		m.backportEvidence.SetText("advisory:version")
		m.version.SetText("1.5.0")
		check(!m.backportConfirmed.IsChecked() && m.backportAdvisory.Text() == "" && m.backportEvidence.Text() == "", "M2 version change clears stale backport")
		m.source.SetCurrentIndex(1)
		m.cveID.SetText("CVE-2026-1000")
		m.tabs.SetCurrentIndex(1)
		captureM2("it-assessment")
		m.unsigned.SetChecked(true)
		bundlePath := filepath.Join(filepath.Dir(w.preferencePath), "m2-selftest.wfr")
		m.destination.SetText(bundlePath)
		m.export.Click()
		waitM2()
		bundle, bundleErr := zip.OpenReader(bundlePath)
		if bundleErr == nil {
			defer bundle.Close()
		}
		assessmentInReport := false
		if bundleErr == nil {
			for _, file := range bundle.File {
				if file.Name != "report-en.json" {
					continue
				}
				reader, readErr := file.Open()
				if readErr != nil {
					break
				}
				data, readErr := io.ReadAll(reader)
				_ = reader.Close()
				assessmentInReport = readErr == nil && strings.Contains(string(data), "CVE-2026-1000")
			}
		}
		_, verificationErr := reporting.Verify(bundlePath, m.trust)
		check(m.result == nil && bundleErr == nil && len(bundle.File) == 5 && assessmentInReport && errors.Is(verificationErr, reporting.ErrUnsigned), "M2 native bilingual unsigned export")
		m.name.SetText("Synthetic signer")
		m.generate.Click()
		waitM2()
		check(len(m.keyIDs) == 1 && len(m.manageKeyIDs) == 1, "M2 native signing key generation")
		oldKey := m.keyIDs[0]
		m.unsigned.SetChecked(false)
		signedPath := filepath.Join(filepath.Dir(w.preferencePath), "m2-signed-selftest.wfr")
		m.destination.SetText(signedPath)
		m.export.Click()
		waitM2()
		m.verifyBundle.SetText(signedPath)
		m.verify.Click()
		waitM2()
		check(strings.Contains(m.status.Text(), "Synthetic signer"), "M2 native signed export and offline verification")
		m.rotate.Click()
		waitM2()
		check(len(m.keyIDs) == 1 && m.keyIDs[0] != oldKey && len(m.manageKeyIDs) == 2, "M2 native key rotation")
		newKey := m.keyIDs[0]
		rotatedPath := filepath.Join(filepath.Dir(w.preferencePath), "m2-rotated-selftest.wfr")
		m.destination.SetText(rotatedPath)
		m.export.Click()
		waitM2()
		m.verifyBundle.SetText(rotatedPath)
		m.verify.Click()
		waitM2()
		check(strings.Contains(m.status.Text(), "Synthetic signer"), "M2 rotated key signs and verifies")
		publicPath := filepath.Join(filepath.Dir(w.preferencePath), "m2-public-selftest.json")
		m.publicDestination.SetText(publicPath)
		m.manageKey.SetCurrentIndex(1)
		m.exportPublic.Click()
		waitM2()
		_, publicErr := os.Stat(publicPath)
		check(publicErr == nil, "M2 native public key export")
		m.revoke.Click()
		waitM2()
		check(len(m.keyIDs) == 0 && m.manageKeyStatus[1] == "revoked", "M2 native key revocation")
		m.verify.Click()
		waitM2()
		_, revokedVerifyErr := reporting.Verify(rotatedPath, m.trust)
		check(revokedVerifyErr != nil && strings.Contains(m.status.Text(), revokedVerifyErr.Error()), "M2 revoked key rejected by GUI verifier")
		otherTrust, otherTrustErr := reporting.OpenTrustStore(filepath.Join(filepath.Dir(w.preferencePath), "other-trust.json"))
		if otherTrustErr == nil {
			otherKeys, _ := reporting.NewKeyring(otherTrust, &selfTestSecrets{data: map[string][]byte{}})
			foreign, foreignErr := otherKeys.Generate(context.Background(), "Foreign synthetic signer")
			foreignPath := filepath.Join(filepath.Dir(w.preferencePath), "foreign-public.json")
			if foreignErr == nil {
				foreignErr = otherTrust.ExportPublic(foreign.KeyID, foreignPath)
			}
			m.publicImportPath.SetText(foreignPath)
			m.importFingerprint.SetText(strings.Repeat("0", 64))
			m.importPublic.Click()
			waitM2()
			_, badImportErr := m.trust.Lookup(foreign.KeyID)
			check(foreignErr == nil && badImportErr != nil, "M2 rejects unconfirmed public fingerprint")
			m.importFingerprint.SetText(foreign.Fingerprint)
			m.importPublic.Click()
			waitM2()
			_, importedErr := m.trust.Lookup(foreign.KeyID)
			check(importedErr == nil && len(m.manageKeyIDs) == 3, "M2 imports independently confirmed public key")
			shortPath := filepath.Join(filepath.Dir(w.preferencePath), "short-public.json")
			data, readErr := os.ReadFile(foreignPath)
			var descriptor reporting.PublicDescriptor
			if readErr == nil {
				readErr = json.Unmarshal(data, &descriptor)
			}
			descriptor.KeyID = "x"
			if readErr == nil {
				data, readErr = json.Marshal(descriptor)
			}
			if readErr == nil {
				readErr = os.WriteFile(shortPath, data, 0o600)
			}
			if readErr == nil {
				m.publicImportPath.SetText(shortPath)
				m.importPublic.Click()
				waitM2()
			}
			_, shortErr := m.trust.Lookup("x")
			check(readErr == nil && shortErr == nil && len(m.manageKeyIDs) == 4, "M2 short imported key ID displays safely")
		}
		check(m.manageKeyIDs[1] == newKey, "M2 selected key identity remains stable")
		m.name.SetText("Second local signer")
		m.generate.Click()
		waitM2()
		check(len(m.keyIDs) == 3, "M2 second active key for selection regression")
		m.signer.SetCurrentIndex(2)
		selectedSigner := m.selectedSignerKey()
		w.englishAction.Trigger()
		check(selectedSigner != "" && m.selectedSignerKey() == selectedSigner, "M2 signer selection survives language change")
		w.italianAction.Trigger()
		m.tabs.SetCurrentIndex(2)
		captureM2("it-reports")
		m.tabs.SetCurrentIndex(0)
		captureM2("it-sources")
		m.dialog.Resize(620, 560)
		qt.QCoreApplication_ProcessEvents()
		check(m.dialog.Width() <= 620 && m.dialog.Height() <= 560, "M2 compact dialog accepts requested size")
		for i, label := range []string{"compact-it-sources", "compact-it-assessment", "compact-it-reports"} {
			m.tabs.SetCurrentIndex(i)
			captureM2(label)
		}
		m.dialog.Resize(820, 740)
		w.englishAction.Trigger()
		check(u.start.Text() == "Start scan" && strings.Contains(u.coverage.Text(), "queue"), "M1 native English translation")
		check(m.match.Text() == "Assess selected run" && m.unsigned.Text() == "Export explicitly unsigned", "M2 native English translation")
		check(m.tabs.TabText(2) == "Reports and keys" && m.rotate.Text() == "Rotate selected key", "M2 advanced English translation")
		for i, label := range []string{"en-sources", "en-assessment", "en-reports"} {
			m.tabs.SetCurrentIndex(i)
			captureM2(label)
		}
		w.italianAction.Trigger()
		u.confirmDelete = func() bool { return true }
		u.deleteProject.Click()
		check(u.selectedProjectID() == "" && u.runs.Count() == 0, "M1 native project deletion")
	}
	runtime.GC() // Exercise automatic lifetime management of returned Qt values.
	qt.QCoreApplication_ProcessEvents()
	if failures > 0 {
		return 1
	}
	return 0
}

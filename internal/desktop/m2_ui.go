package desktop

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Matte2599/WebFence/internal/intelligence"
	"github.com/Matte2599/WebFence/internal/reporting"
	qt "github.com/mappu/miqt/qt6"
)

// m2UI keeps CVE signals operator-supplied. A scan header never becomes a
// product identity. Network/feed and keychain work runs off the Qt thread.
type m2UI struct {
	parent                                         *scannerUI
	cache                                          *intelligence.Cache
	trust                                          *reporting.TrustStore
	keys                                           *reporting.Keyring
	cacheErr, trustErr                             error
	dialog                                         *qt.QDialog
	timer                                          *qt.QTimer
	tabs                                           *qt.QTabWidget
	pages                                          []*qt.QScrollArea
	labels                                         map[string]*qt.QLabel
	intro, status, cacheState, signerHelp, cpeHelp *qt.QLabel
	startDate, endDate                             *qt.QLineEdit
	cveID, cpe23, vendor, product                  *qt.QLineEdit
	version, evidence, name                        *qt.QLineEdit
	destination, publicDestination                 *qt.QLineEdit
	publicImportPath, importFingerprint            *qt.QLineEdit
	verifyBundle                                   *qt.QLineEdit
	source, part, method, confidence               *qt.QComboBox
	verificationMethod, signer, manageKey          *qt.QComboBox
	unsigned, backportConfirmed                    *qt.QCheckBox
	verificationConfirmed                          *qt.QCheckBox
	backportAdvisory, backportEvidence             *qt.QLineEdit
	verificationEvidence                           *qt.QLineEdit
	syncNVD, fetchCVE                              *qt.QPushButton
	match, generate, browse                        *qt.QPushButton
	export, verify, cancel                         *qt.QPushButton
	rotate, revoke, exportPublic                   *qt.QPushButton
	importPublic, browsePublic                     *qt.QPushButton
	browseImport, browseVerify                     *qt.QPushButton
	assessmentView                                 *qt.QPlainTextEdit
	keyIDs                                         []string
	manageKeyIDs, manageKeyStatus                  []string
	confirmKeyChange                               func(string, string) bool
	assessments                                    map[string]intelligence.Assessment
	assessmentRun                                  string
	result                                         chan m2Completion
	cancelWork                                     context.CancelFunc
}

type m2Completion struct {
	kind         string
	key          string
	fingerprint  string
	runID        string
	assessment   intelligence.Assessment
	verification reporting.Verification
	err          error
}

func newM2UI(parent *scannerUI, cache *intelligence.Cache, cacheErr error, trust *reporting.TrustStore, trustErr error) *m2UI {
	m := &m2UI{parent: parent, cache: cache, cacheErr: cacheErr, trust: trust, trustErr: trustErr,
		labels: make(map[string]*qt.QLabel), assessments: make(map[string]intelligence.Assessment)}
	if trust != nil {
		m.keys, trustErr = reporting.NativeKeyring(trust)
		m.trustErr = trustErr
	}
	m.dialog = qt.NewQDialog(parent.dialog.QWidget)
	m.dialog.Resize(820, 740)
	outer := qt.NewQVBoxLayout(m.dialog.QWidget)
	m.intro = qt.NewQLabel2()
	m.intro.SetWordWrap(true)
	outer.AddWidget(m.intro.QWidget)
	m.tabs = qt.NewQTabWidget2()
	page := func() *qt.QFormLayout {
		scroll := qt.NewQScrollArea2()
		scroll.SetWidgetResizable(true)
		body := qt.NewQWidget(nil)
		form := qt.NewQFormLayout(body)
		form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
		form.SetRowWrapPolicy(qt.QFormLayout__WrapLongRows)
		scroll.SetWidget(body)
		m.pages = append(m.pages, scroll)
		m.tabs.AddTab(scroll.QWidget, "")
		return form
	}
	row := func(form *qt.QFormLayout, key string, widget *qt.QWidget) {
		label := qt.NewQLabel2()
		label.SetBuddy(widget)
		m.labels[key] = label
		form.AddRow(label.QWidget, widget)
	}
	sources := page()
	m.startDate, m.endDate = qt.NewQLineEdit2(), qt.NewQLineEdit2()
	m.startDate.SetText(time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02"))
	m.endDate.SetText(time.Now().UTC().Format("2006-01-02"))
	row(sources, "m2_start", m.startDate.QWidget)
	row(sources, "m2_end", m.endDate.QWidget)
	m.syncNVD = qt.NewQPushButton2()
	sources.AddRowWithWidget(m.syncNVD.QWidget)
	m.cveID = qt.NewQLineEdit2()
	m.cveID.SetMaxLength(32)
	row(sources, "m2_cve_id", m.cveID.QWidget)
	m.fetchCVE = qt.NewQPushButton2()
	sources.AddRowWithWidget(m.fetchCVE.QWidget)
	matching := page()
	m.source = qt.NewQComboBox2()
	m.source.AddItems([]string{"NVD", "CVE"})
	row(matching, "m2_source", m.source.QWidget)
	m.cpe23 = qt.NewQLineEdit2()
	m.cpe23.SetMaxLength(2048)
	row(matching, "m2_cpe23", m.cpe23.QWidget)
	m.cpeHelp = qt.NewQLabel2()
	m.cpeHelp.SetWordWrap(true)
	matching.AddRowWithWidget(m.cpeHelp.QWidget)
	m.part = qt.NewQComboBox2()
	m.part.AddItems([]string{"", "", "", ""})
	row(matching, "m2_part", m.part.QWidget)
	m.vendor, m.product, m.version, m.evidence = qt.NewQLineEdit2(), qt.NewQLineEdit2(), qt.NewQLineEdit2(), qt.NewQLineEdit2()
	for _, field := range []*qt.QLineEdit{m.vendor, m.product, m.version, m.evidence} {
		field.SetMaxLength(128)
	}
	row(matching, "m2_vendor", m.vendor.QWidget)
	row(matching, "m2_product", m.product.QWidget)
	row(matching, "m2_version", m.version.QWidget)
	row(matching, "m2_evidence", m.evidence.QWidget)
	m.method, m.confidence = qt.NewQComboBox2(), qt.NewQComboBox2()
	m.method.AddItems([]string{"", "", ""})
	m.confidence.AddItems([]string{"", "", ""})
	row(matching, "m2_method", m.method.QWidget)
	row(matching, "m2_confidence", m.confidence.QWidget)
	m.backportConfirmed, m.verificationConfirmed = qt.NewQCheckBox2(), qt.NewQCheckBox2()
	m.backportAdvisory, m.backportEvidence, m.verificationEvidence = qt.NewQLineEdit2(), qt.NewQLineEdit2(), qt.NewQLineEdit2()
	m.backportAdvisory.SetMaxLength(2048)
	m.backportEvidence.SetMaxLength(128)
	m.verificationEvidence.SetMaxLength(128)
	matching.AddRowWithWidget(m.backportConfirmed.QWidget)
	row(matching, "m2_backport_advisory", m.backportAdvisory.QWidget)
	row(matching, "m2_backport_evidence", m.backportEvidence.QWidget)
	matching.AddRowWithWidget(m.verificationConfirmed.QWidget)
	m.verificationMethod = qt.NewQComboBox2()
	m.verificationMethod.AddItems([]string{"", ""})
	row(matching, "m2_verification_method", m.verificationMethod.QWidget)
	row(matching, "m2_verification_evidence", m.verificationEvidence.QWidget)
	m.match = qt.NewQPushButton2()
	matching.AddRowWithWidget(m.match.QWidget)
	reports := page()
	m.name = qt.NewQLineEdit2()
	m.name.SetMaxLength(128)
	row(reports, "m2_operator", m.name.QWidget)
	m.generate = qt.NewQPushButton2()
	reports.AddRowWithWidget(m.generate.QWidget)
	m.signer = qt.NewQComboBox2()
	row(reports, "m2_signer", m.signer.QWidget)
	m.signerHelp = qt.NewQLabel2()
	m.signerHelp.SetWordWrap(true)
	reports.AddRowWithWidget(m.signerHelp.QWidget)
	m.manageKey = qt.NewQComboBox2()
	row(reports, "m2_manage_key", m.manageKey.QWidget)
	m.rotate, m.revoke = qt.NewQPushButton2(), qt.NewQPushButton2()
	reports.AddRowWithWidget(m.rotate.QWidget)
	reports.AddRowWithWidget(m.revoke.QWidget)
	m.publicDestination = qt.NewQLineEdit2()
	row(reports, "m2_public_destination", m.publicDestination.QWidget)
	m.browsePublic, m.exportPublic = qt.NewQPushButton2(), qt.NewQPushButton2()
	reports.AddRowWithWidget(m.browsePublic.QWidget)
	reports.AddRowWithWidget(m.exportPublic.QWidget)
	m.publicImportPath, m.importFingerprint = qt.NewQLineEdit2(), qt.NewQLineEdit2()
	m.importFingerprint.SetMaxLength(64)
	row(reports, "m2_public_import_path", m.publicImportPath.QWidget)
	m.browseImport, m.importPublic = qt.NewQPushButton2(), qt.NewQPushButton2()
	reports.AddRowWithWidget(m.browseImport.QWidget)
	row(reports, "m2_import_fingerprint", m.importFingerprint.QWidget)
	reports.AddRowWithWidget(m.importPublic.QWidget)
	m.unsigned = qt.NewQCheckBox2()
	reports.AddRowWithWidget(m.unsigned.QWidget)
	m.destination = qt.NewQLineEdit2()
	row(reports, "m2_destination", m.destination.QWidget)
	m.browse, m.export, m.verify = qt.NewQPushButton2(), qt.NewQPushButton2(), qt.NewQPushButton2()
	reports.AddRowWithWidget(m.browse.QWidget)
	reports.AddRowWithWidget(m.export.QWidget)
	m.verifyBundle = qt.NewQLineEdit2()
	row(reports, "m2_verify_bundle", m.verifyBundle.QWidget)
	m.browseVerify = qt.NewQPushButton2()
	reports.AddRowWithWidget(m.browseVerify.QWidget)
	reports.AddRowWithWidget(m.verify.QWidget)
	outer.AddWidget(m.tabs.QWidget)
	m.cacheState, m.status = qt.NewQLabel2(), qt.NewQLabel2()
	m.cacheState.SetWordWrap(true)
	m.status.SetWordWrap(true)
	outer.AddWidget(m.cacheState.QWidget)
	outer.AddWidget(m.status.QWidget)
	m.assessmentView = qt.NewQPlainTextEdit2()
	m.assessmentView.SetReadOnly(true)
	m.assessmentView.SetMinimumHeight(100)
	m.assessmentView.SetMaximumHeight(140)
	outer.AddWidget(m.assessmentView.QWidget)
	m.cancel = qt.NewQPushButton2()
	m.cancel.SetEnabled(false)
	outer.AddWidget(m.cancel.QWidget)
	m.timer = qt.NewQTimer2(m.dialog.QObject)
	m.timer.OnTimeout(m.poll)
	m.timer.Start(100)
	m.syncNVD.OnClicked(m.onSyncNVD)
	m.fetchCVE.OnClicked(m.onFetchCVE)
	m.match.OnClicked(m.onMatch)
	m.generate.OnClicked(m.onGenerate)
	m.rotate.OnClicked(m.onRotate)
	m.revoke.OnClicked(m.onRevoke)
	m.manageKey.OnCurrentIndexChanged(func(int) { m.refreshControls() })
	m.source.OnCurrentIndexChanged(func(int) {
		m.part.SetCurrentIndex(0)
		m.updateIdentityControls()
		m.clearAttestations()
	})
	m.cpe23.OnTextChanged(func(string) {
		m.updateIdentityControls()
		m.clearAttestations()
	})
	m.part.OnCurrentIndexChanged(func(int) { m.clearAttestations() })
	m.method.OnCurrentIndexChanged(func(int) { m.clearAttestations() })
	m.confidence.OnCurrentIndexChanged(func(int) { m.clearAttestations() })
	for _, field := range []*qt.QLineEdit{m.cveID, m.vendor, m.product, m.version, m.evidence} {
		field.OnTextChanged(func(string) { m.clearAttestations() })
	}
	m.browse.OnClicked(func() {
		path := qt.QFileDialog_GetSaveFileName4(m.dialog.QWidget, m.tr("m2_destination"), "", "WebFence (*.wfr)")
		if path != "" {
			m.destination.SetText(path)
		}
	})
	m.browsePublic.OnClicked(func() {
		path := qt.QFileDialog_GetSaveFileName4(m.dialog.QWidget, m.tr("m2_public_destination"), "", "JSON (*.json)")
		if path != "" {
			m.publicDestination.SetText(path)
		}
	})
	m.exportPublic.OnClicked(m.onExportPublic)
	m.browseImport.OnClicked(func() {
		path := qt.QFileDialog_GetOpenFileName4(m.dialog.QWidget, m.tr("m2_public_import_path"), "", "JSON (*.json)")
		if path != "" {
			m.publicImportPath.SetText(path)
		}
	})
	m.importPublic.OnClicked(m.onImportPublic)
	m.export.OnClicked(m.onExport)
	m.browseVerify.OnClicked(func() {
		path := qt.QFileDialog_GetOpenFileName4(m.dialog.QWidget, m.tr("m2_verify_bundle"), "", "WebFence (*.wfr)")
		if path != "" {
			m.verifyBundle.SetText(path)
		}
	})
	m.verify.OnClicked(m.onVerify)
	m.cancel.OnClicked(func() {
		if m.cancelWork != nil {
			m.cancelWork()
		}
	})
	m.unsigned.OnToggled(func(bool) { m.refreshControls() })
	m.confirmKeyChange = func(title, message string) bool {
		return qt.QMessageBox_Question6(m.dialog.QWidget, title, message,
			qt.QMessageBox__Yes|qt.QMessageBox__No, qt.QMessageBox__No) == qt.QMessageBox__Yes
	}
	m.translate()
	m.refreshKeys()
	m.refreshCache()
	if m.trust == nil {
		m.status.SetText(m.tr("m2_trust_unavailable"))
	}
	return m
}

func (m *m2UI) tr(key string, args ...any) string { return m.parent.tr(key, args...) }
func (m *m2UI) translate() {
	m.dialog.SetWindowTitle(m.tr("m2_title"))
	m.intro.SetText(m.tr("m2_intro"))
	m.signerHelp.SetText(m.tr("m2_signer_help"))
	m.cpeHelp.SetText(m.tr("m2_cpe_help"))
	for i, key := range []string{"m2_tab_sources", "m2_tab_matching", "m2_tab_reports"} {
		m.tabs.SetTabText(i, m.tr(key))
	}
	for key, label := range m.labels {
		label.SetText(m.tr(key))
	}
	m.syncNVD.SetText(m.tr("m2_sync_nvd"))
	m.fetchCVE.SetText(m.tr("m2_fetch_cve"))
	m.match.SetText(m.tr("m2_match"))
	m.generate.SetText(m.tr("m2_generate"))
	m.rotate.SetText(m.tr("m2_rotate"))
	m.revoke.SetText(m.tr("m2_revoke"))
	m.exportPublic.SetText(m.tr("m2_export_public"))
	m.importPublic.SetText(m.tr("m2_import_public"))
	m.browsePublic.SetText(m.tr("m2_browse_public"))
	m.browseImport.SetText(m.tr("m2_browse_import"))
	m.browseVerify.SetText(m.tr("m2_browse_verify"))
	m.browse.SetText(m.tr("m2_browse"))
	m.export.SetText(m.tr("m2_export"))
	m.verify.SetText(m.tr("m2_verify"))
	m.cancel.SetText(m.tr("m2_cancel"))
	m.unsigned.SetText(m.tr("m2_unsigned"))
	m.backportConfirmed.SetText(m.tr("m2_backport_confirmed"))
	m.verificationConfirmed.SetText(m.tr("m2_verification_confirmed"))
	for i, key := range []string{"m2_part_select", "m2_part_app", "m2_part_os", "m2_part_hardware"} {
		m.part.SetItemText(i, m.tr(key))
	}
	for i, key := range []string{"m2_verification_manual", "m2_verification_safe"} {
		m.verificationMethod.SetItemText(i, m.tr(key))
	}
	for i, key := range []string{"m2_inventory", "m2_manual", "m2_banner"} {
		m.method.SetItemText(i, m.tr(key))
	}
	for i, key := range []string{"m2_high", "m2_medium", "m2_low"} {
		m.confidence.SetItemText(i, m.tr(key))
	}
	m.refreshKeys()
	m.refreshCache()
	m.showAssessments()
}

func (m *m2UI) show() {
	m.updateRun()
	m.refreshKeys()
	m.refreshCache()
	m.dialog.Show()
	m.dialog.Raise()
	m.dialog.ActivateWindow()
}

func (m *m2UI) selectedRun() string {
	i := m.parent.runs.CurrentIndex()
	if i < 0 || i >= len(m.parent.runIDs) {
		return ""
	}
	return m.parent.runIDs[i]
}

func (m *m2UI) clearAttestations() {
	m.backportConfirmed.SetChecked(false)
	m.backportAdvisory.SetText("")
	m.backportEvidence.SetText("")
	m.verificationConfirmed.SetChecked(false)
	m.verificationEvidence.SetText("")
}

func (m *m2UI) updateIdentityControls() {
	manual := strings.TrimSpace(m.cpe23.Text()) == ""
	m.vendor.SetEnabled(manual)
	m.product.SetEnabled(manual)
	m.part.SetEnabled(manual && m.source.CurrentIndex() == 0)
}

func (m *m2UI) updateRun() {
	id := m.selectedRun()
	if id == m.assessmentRun {
		return
	}
	m.assessmentRun = id
	m.assessments = make(map[string]intelligence.Assessment)
	m.clearAttestations()
	m.showAssessments()
}

func (m *m2UI) showAssessments() {
	if m.assessmentView == nil {
		return
	}
	lines := []string{m.tr("m2_session_note")}
	keys := make([]string, 0, len(m.assessments))
	for key := range m.assessments {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		a := m.assessments[key]
		lines = append(lines, fmt.Sprintf("%s · %s · %s · %s · %s", a.CVEID, a.Source, m.tr("report_"+a.Status), m.tr("m2_"+a.Confidence), m.tr("m2_reason_"+a.Reason)))
	}
	m.assessmentView.SetPlainText(strings.Join(lines, "\n"))
}

func (m *m2UI) refreshControls() {
	busy := m.result != nil
	m.syncNVD.SetEnabled(!busy && m.cache != nil)
	m.fetchCVE.SetEnabled(!busy && m.cache != nil)
	m.match.SetEnabled(!busy && m.cache != nil && m.selectedRun() != "")
	m.generate.SetEnabled(!busy && m.keys != nil)
	index := m.manageKey.CurrentIndex()
	selected := index >= 0 && index < len(m.manageKeyIDs)
	active := selected && m.manageKeyStatus[index] == "active"
	m.rotate.SetEnabled(!busy && m.keys != nil && active)
	m.revoke.SetEnabled(!busy && m.keys != nil && selected && m.manageKeyStatus[index] != "revoked")
	m.exportPublic.SetEnabled(!busy && m.trust != nil && selected)
	m.importPublic.SetEnabled(!busy && m.trust != nil)
	m.export.SetEnabled(!busy && m.parent.store != nil && m.selectedRun() != "" && (m.unsigned.IsChecked() || len(m.keyIDs) > 0))
	m.verify.SetEnabled(!busy && m.trust != nil)
	m.cancel.SetEnabled(busy)
}

func (m *m2UI) refreshCache() {
	if m.cacheState == nil {
		return
	}
	if m.cache == nil {
		m.cacheState.SetText(m.tr("m2_cache_unavailable"))
		m.refreshControls()
		return
	}
	lines := []string{}
	for _, source := range []string{"nvd", "cve"} {
		s, err := m.cache.Snapshot(context.Background(), source)
		if err != nil {
			lines = append(lines, source+": "+m.tr("report_unavailable"))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s · %d · %s", source, m.tr("report_"+s.Status(time.Now().UTC(), 24*time.Hour, false)), s.Records, s.LastSuccess.UTC().Format("2006-01-02 15:04 UTC")))
	}
	m.cacheState.SetText(strings.Join(lines, "\n"))
	m.refreshControls()
}

func (m *m2UI) refreshKeys() {
	selectedID := m.selectedManagedKey()
	m.signer.Clear()
	m.manageKey.Clear()
	m.keyIDs = nil
	m.manageKeyIDs, m.manageKeyStatus = nil, nil
	if m.trust != nil {
		if records, err := m.trust.List(); err == nil {
			for _, k := range records {
				m.manageKeyIDs = append(m.manageKeyIDs, k.KeyID)
				m.manageKeyStatus = append(m.manageKeyStatus, k.Status)
				m.manageKey.AddItem(k.Identity + " (" + k.KeyID[:8] + ") · " + m.tr("m2_key_"+k.Status))
				if k.Status == "active" {
					m.keyIDs = append(m.keyIDs, k.KeyID)
					m.signer.AddItem(k.Identity + " (" + k.KeyID[:8] + ")")
				}
			}
		}
	}
	for i, id := range m.manageKeyIDs {
		if id == selectedID {
			m.manageKey.SetCurrentIndex(i)
			break
		}
	}
	m.refreshControls()
}

func (m *m2UI) selectedManagedKey() string {
	i := m.manageKey.CurrentIndex()
	if i < 0 || i >= len(m.manageKeyIDs) {
		return ""
	}
	return m.manageKeyIDs[i]
}

func (m *m2UI) begin(kind string, work func(context.Context) m2Completion) {
	if m.result != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelWork = cancel
	m.result = make(chan m2Completion, 1)
	channel := m.result
	m.status.SetText(m.tr("m2_working"))
	m.refreshControls()
	go func() { output := work(ctx); output.kind = kind; channel <- output }()
}

func (m *m2UI) poll() {
	if m.result == nil {
		return
	}
	select {
	case result := <-m.result:
		m.cancelWork()
		m.cancelWork = nil
		m.result = nil
		if result.err != nil {
			m.status.SetText(m.tr("scan_error", result.err))
		} else {
			switch result.kind {
			case "sync", "fetch":
				m.status.SetText(m.tr("m2_cache_updated"))
				m.refreshCache()
			case "match":
				m.updateRun()
				if result.runID == m.assessmentRun {
					m.assessments[result.assessment.Source+":"+result.assessment.CVEID] = result.assessment
					m.status.SetText(m.tr("m2_assessed"))
					m.showAssessments()
				}
			case "keygen":
				m.status.SetText(m.tr("m2_key_created", result.key, result.fingerprint))
				m.refreshKeys()
			case "rotate":
				m.status.SetText(m.tr("m2_key_rotated", result.key, result.fingerprint))
				m.refreshKeys()
			case "revoke":
				m.status.SetText(m.tr("m2_key_revoke_done", result.key))
				m.refreshKeys()
			case "public":
				m.status.SetText(m.tr("m2_public_exported"))
			case "import":
				m.status.SetText(m.tr("m2_public_imported", result.key, result.fingerprint))
				m.refreshKeys()
			case "export":
				m.status.SetText(m.tr("m2_exported"))
			case "verify":
				m.status.SetText(m.tr("m2_verified", result.verification.TrustedIdentity, result.verification.TrustStatus))
			}
		}
		m.refreshControls()
	default:
	}
}

func (m *m2UI) onSyncNVD() {
	if m.cache == nil {
		return
	}
	start, end, err := nvdWindow(m.startDate.Text(), m.endDate.Text(), time.Now().UTC())
	if err != nil {
		m.status.SetText(m.tr("m2_invalid_window"))
		return
	}
	m.begin("sync", func(ctx context.Context) m2Completion {
		_, err := m.cache.SyncNVD(ctx, &intelligence.NVDClient{}, start, end)
		return m2Completion{err: err}
	})
}

func nvdWindow(startText, endText string, now time.Time) (time.Time, time.Time, error) {
	start, e1 := time.Parse("2006-01-02", strings.TrimSpace(startText))
	endDay, e2 := time.Parse("2006-01-02", strings.TrimSpace(endText))
	now = now.UTC()
	if e1 != nil || e2 != nil || endDay.After(now.Truncate(24*time.Hour)) {
		return time.Time{}, time.Time{}, intelligence.ErrInvalid
	}
	end := endDay.Add(24*time.Hour - time.Millisecond)
	if end.After(now) {
		end = now
	}
	if end.Before(start) || end.Sub(start) > 120*24*time.Hour {
		return time.Time{}, time.Time{}, intelligence.ErrInvalid
	}
	return start, end, nil
}

func (m *m2UI) onFetchCVE() {
	if m.cache == nil {
		return
	}
	id := strings.TrimSpace(m.cveID.Text())
	m.begin("fetch", func(ctx context.Context) m2Completion {
		_, err := m.cache.RefreshCVE(ctx, &intelligence.CVEClient{}, []string{id})
		return m2Completion{err: err}
	})
}

func (m *m2UI) onMatch() {
	m.updateRun()
	if m.cache == nil || m.assessmentRun == "" {
		return
	}
	if m.method.CurrentIndex() < 0 || m.method.CurrentIndex() > 2 || m.confidence.CurrentIndex() < 0 || m.confidence.CurrentIndex() > 2 {
		m.status.SetText(m.tr("m2_invalid_signal"))
		return
	}
	source := "nvd"
	if m.source.CurrentIndex() == 1 {
		source = "cve"
	}
	method := []string{"inventory", "manual", "banner"}[m.method.CurrentIndex()]
	confidence := []string{"high", "medium", "low"}[m.confidence.CurrentIndex()]
	cpe23 := strings.TrimSpace(m.cpe23.Text())
	part := ""
	if source == "nvd" && cpe23 == "" {
		i := m.part.CurrentIndex()
		if i < 1 || i > 3 {
			m.status.SetText(m.tr("m2_part_required"))
			return
		}
		part = []string{"", "a", "o", "h"}[i]
	}
	signal := intelligence.ProductSignal{CPE23: cpe23, Version: strings.TrimSpace(m.version.Text()), Part: part, EvidenceRef: strings.TrimSpace(m.evidence.Text()), Method: method, Confidence: confidence}
	if cpe23 == "" {
		signal.Vendor, signal.Product = strings.TrimSpace(m.vendor.Text()), strings.TrimSpace(m.product.Text())
	}
	id := strings.TrimSpace(m.cveID.Text())
	var backport *intelligence.Backport
	var verification *intelligence.Verification
	backportURL, backportRef := strings.TrimSpace(m.backportAdvisory.Text()), strings.TrimSpace(m.backportEvidence.Text())
	verificationRef := strings.TrimSpace(m.verificationEvidence.Text())
	if m.backportConfirmed.IsChecked() && m.verificationConfirmed.IsChecked() {
		m.status.SetText(m.tr("m2_attestation_conflict"))
		return
	}
	if backportURL != "" || backportRef != "" || m.backportConfirmed.IsChecked() {
		if !m.backportConfirmed.IsChecked() || backportURL == "" || backportRef == "" {
			m.status.SetText(m.tr("m2_backport_incomplete"))
			return
		}
		backport = &intelligence.Backport{CVEID: id, Version: signal.Version, AdvisoryURL: backportURL, EvidenceRef: backportRef, Confirmed: true}
		if !intelligence.ValidBackport(*backport) {
			m.status.SetText(m.tr("m2_backport_incomplete"))
			return
		}
	}
	if verificationRef != "" || m.verificationConfirmed.IsChecked() {
		if !m.verificationConfirmed.IsChecked() || verificationRef == "" {
			m.status.SetText(m.tr("m2_verification_incomplete"))
			return
		}
		methodIndex := m.verificationMethod.CurrentIndex()
		if methodIndex < 0 || methodIndex > 1 {
			m.status.SetText(m.tr("m2_verification_incomplete"))
			return
		}
		verification = &intelligence.Verification{Method: []string{"manual", "safe_check"}[methodIndex], EvidenceRef: verificationRef, Confirmed: true}
		if !intelligence.ValidVerification(*verification) {
			m.status.SetText(m.tr("m2_verification_incomplete"))
			return
		}
	}
	runID := m.assessmentRun
	m.begin("match", func(ctx context.Context) m2Completion {
		record, err := m.cache.Get(ctx, source, id)
		if err != nil {
			return m2Completion{err: err, runID: runID}
		}
		assessment, err := intelligence.Assess(record, signal, backport, verification)
		return m2Completion{assessment: assessment, err: err, runID: runID}
	})
}

func (m *m2UI) onGenerate() {
	if m.keys == nil {
		return
	}
	name := strings.TrimSpace(m.name.Text())
	m.begin("keygen", func(ctx context.Context) m2Completion {
		key, err := m.keys.Generate(ctx, name)
		return m2Completion{key: key.KeyID, fingerprint: key.Fingerprint, err: err}
	})
}

func (m *m2UI) onRotate() {
	if m.keys == nil || m.selectedManagedKey() == "" {
		return
	}
	id := m.selectedManagedKey()
	if !m.confirmKeyChange(m.tr("m2_rotate"), m.tr("m2_rotate_confirm", id)) {
		return
	}
	m.begin("rotate", func(ctx context.Context) m2Completion {
		record, err := m.keys.Rotate(ctx, id)
		return m2Completion{key: record.KeyID, fingerprint: record.Fingerprint, err: err}
	})
}

func (m *m2UI) onRevoke() {
	if m.keys == nil || m.selectedManagedKey() == "" {
		return
	}
	id := m.selectedManagedKey()
	if !m.confirmKeyChange(m.tr("m2_revoke"), m.tr("m2_revoke_confirm", id)) {
		return
	}
	m.begin("revoke", func(ctx context.Context) m2Completion {
		return m2Completion{key: id, err: m.keys.Revoke(ctx, id)}
	})
}

func (m *m2UI) onExportPublic() {
	if m.trust == nil || m.selectedManagedKey() == "" {
		return
	}
	id, path := m.selectedManagedKey(), strings.TrimSpace(m.publicDestination.Text())
	if !filepath.IsAbs(path) {
		m.status.SetText(m.tr("m2_invalid_path"))
		return
	}
	m.begin("public", func(context.Context) m2Completion {
		return m2Completion{err: m.trust.ExportPublic(id, path)}
	})
}

func (m *m2UI) onImportPublic() {
	if m.trust == nil {
		return
	}
	path := strings.TrimSpace(m.publicImportPath.Text())
	fingerprint := strings.ToLower(strings.TrimSpace(m.importFingerprint.Text()))
	if !filepath.IsAbs(path) || len(fingerprint) != 64 {
		m.status.SetText(m.tr("m2_import_incomplete"))
		return
	}
	m.begin("import", func(context.Context) m2Completion {
		record, err := m.trust.ImportTrusted(path, fingerprint)
		return m2Completion{key: record.KeyID, fingerprint: record.Fingerprint, err: err}
	})
}

func (m *m2UI) onExport() {
	m.updateRun()
	runID, path := m.assessmentRun, strings.TrimSpace(m.destination.Text())
	if runID == "" || !filepath.IsAbs(path) {
		m.status.SetText(m.tr("m2_invalid_path"))
		return
	}
	unsigned := m.unsigned.IsChecked()
	keyID := ""
	if !unsigned {
		i := m.signer.CurrentIndex()
		if i < 0 || i >= len(m.keyIDs) {
			m.status.SetText(m.tr("m2_select_signer"))
			return
		}
		keyID = m.keyIDs[i]
	}
	assessments := make([]intelligence.Assessment, 0, len(m.assessments))
	keys := make([]string, 0, len(m.assessments))
	for key := range m.assessments {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		assessments = append(assessments, m.assessments[key])
	}
	m.begin("export", func(ctx context.Context) m2Completion {
		s, err := reporting.Load(ctx, m.parent.store, runID)
		if err != nil {
			return m2Completion{err: err}
		}
		s.Assessments = assessments
		if m.cache != nil {
			for _, source := range []string{"nvd", "cve"} {
				if cacheSnapshot, e := m.cache.Snapshot(ctx, source); e == nil {
					s.Intelligence = append(s.Intelligence, cacheSnapshot)
				}
			}
		}
		if unsigned {
			_, err = reporting.Export(ctx, s, path, reporting.ExportOptions{Offline: true})
		} else {
			_, err = m.keys.ExportSigned(ctx, s, path, keyID, reporting.ExportOptions{Offline: true})
		}
		return m2Completion{err: err}
	})
}

func (m *m2UI) onVerify() {
	if m.trust == nil {
		return
	}
	path := strings.TrimSpace(m.verifyBundle.Text())
	if !filepath.IsAbs(path) {
		m.status.SetText(m.tr("m2_invalid_path"))
		return
	}
	m.begin("verify", func(context.Context) m2Completion {
		result, err := reporting.Verify(path, m.trust)
		return m2Completion{verification: result, err: err}
	})
}

func (m *m2UI) dispose() {
	if m == nil {
		return
	}
	if m.cancelWork != nil {
		m.cancelWork()
	}
	if m.result != nil {
		<-m.result
	}
	m.timer.Stop()
	m.dialog.Close()
	m.dialog.Delete()
}

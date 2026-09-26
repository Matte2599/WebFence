package desktop

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/Matte2599/WebFence/internal/intelligence"
	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/reporting"
	"github.com/Matte2599/WebFence/internal/scanner"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/storage"
	"github.com/Matte2599/WebFence/internal/transport"
	qt "github.com/mappu/miqt/qt6"
)

// scannerUI is a small, bounded M1 alpha workflow. All Qt calls run on the
// GUI thread. The worker receives an immutable plan and sends counts/results
// over channels; no target URL or response content is rendered as evidence.
type scannerUI struct {
	w                                                        *workspace
	store                                                    *storage.Store
	dialog                                                   *qt.QDialog
	timer                                                    *qt.QTimer
	labels                                                   map[string]*qt.QLabel
	intro, status, coverage                                  *qt.QLabel
	projects, runs, mode                                     *qt.QComboBox
	id, name, owner, reference, expiry, origin               *qt.QLineEdit
	seed, allowed, excluded, pins                            *qt.QLineEdit
	confirmed, follow                                        *qt.QCheckBox
	maxPages, maxDepth, budget                               *qt.QSpinBox
	create, deleteProject, start, cancelRun, refresh, m2Open *qt.QPushButton
	progress                                                 *qt.QProgressBar
	results                                                  *qt.QPlainTextEdit
	projectIDs, runIDs                                       []string
	initErr                                                  error
	cancel                                                   context.CancelFunc
	done                                                     chan struct{}
	progressUpdates                                          chan int
	finished                                                 chan scanCompletion
	confirmDelete                                            func() bool
	m2                                                       *m2UI
}

type scanCompletion struct {
	report scanner.CrawlReport
	err    error
}

func newScannerUI(w *workspace, store *storage.Store, initErr error, cache *intelligence.Cache, cacheErr error, trust *reporting.TrustStore, trustErr error) *scannerUI {
	u := &scannerUI{w: w, store: store, initErr: initErr, labels: make(map[string]*qt.QLabel)}
	u.dialog = qt.NewQDialog(w.window.QWidget)
	u.dialog.Resize(820, 790)
	outer := qt.NewQVBoxLayout(u.dialog.QWidget)
	u.intro = qt.NewQLabel2()
	u.intro.SetWordWrap(true)
	outer.AddWidget(u.intro.QWidget)
	scroll := qt.NewQScrollArea2()
	scroll.SetWidgetResizable(true)
	formBody := qt.NewQWidget(nil)
	form := qt.NewQFormLayout(formBody)
	row := func(key string, widget *qt.QWidget) {
		label := qt.NewQLabel2()
		label.SetBuddy(widget)
		u.labels[key] = label
		form.AddRow(label.QWidget, widget)
	}
	u.projects = qt.NewQComboBox2()
	row("scan_projects", u.projects.QWidget)
	u.id = qt.NewQLineEdit2()
	u.id.SetMaxLength(64)
	row("scan_id", u.id.QWidget)
	u.name = qt.NewQLineEdit2()
	u.name.SetMaxLength(128)
	row("scan_name", u.name.QWidget)
	u.owner = qt.NewQLineEdit2()
	u.owner.SetMaxLength(256)
	row("scan_owner", u.owner.QWidget)
	u.reference = qt.NewQLineEdit2()
	u.reference.SetMaxLength(256)
	row("scan_reference", u.reference.QWidget)
	u.expiry = qt.NewQLineEdit2()
	u.expiry.SetMaxLength(10)
	u.expiry.SetText(time.Now().AddDate(0, 0, 1).Format("2006-01-02"))
	row("scan_expiry", u.expiry.QWidget)
	u.origin = qt.NewQLineEdit2()
	u.origin.SetMaxLength(2048)
	row("scan_origin", u.origin.QWidget)
	u.confirmed = qt.NewQCheckBox2()
	form.AddRowWithWidget(u.confirmed.QWidget)
	u.create = qt.NewQPushButton2()
	form.AddRowWithWidget(u.create.QWidget)
	u.seed = qt.NewQLineEdit2()
	u.seed.SetMaxLength(2048)
	row("scan_seed", u.seed.QWidget)
	u.allowed = qt.NewQLineEdit2()
	u.allowed.SetMaxLength(2048)
	u.allowed.SetText("/")
	row("scan_allowed", u.allowed.QWidget)
	u.excluded = qt.NewQLineEdit2()
	u.excluded.SetMaxLength(2048)
	row("scan_excluded", u.excluded.QWidget)
	u.mode = qt.NewQComboBox2()
	u.mode.AddItems([]string{"", ""})
	row("scan_mode", u.mode.QWidget)
	u.pins = qt.NewQLineEdit2()
	u.pins.SetMaxLength(1024)
	row("scan_pins", u.pins.QWidget)
	u.budget = qt.NewQSpinBox2()
	u.budget.SetRange(1, 1000)
	u.budget.SetValue(30)
	row("scan_budget", u.budget.QWidget)
	u.maxPages = qt.NewQSpinBox2()
	u.maxPages.SetRange(1, scanner.MaxCrawlPages)
	u.maxPages.SetValue(10)
	row("scan_pages", u.maxPages.QWidget)
	u.maxDepth = qt.NewQSpinBox2()
	u.maxDepth.SetRange(1, scanner.MaxCrawlDepth)
	u.maxDepth.SetValue(2)
	row("scan_depth", u.maxDepth.QWidget)
	u.follow = qt.NewQCheckBox2()
	form.AddRowWithWidget(u.follow.QWidget)
	u.start = qt.NewQPushButton2()
	form.AddRowWithWidget(u.start.QWidget)
	scroll.SetWidget(formBody)
	outer.AddWidget(scroll.QWidget)
	controls := qt.NewQWidget(nil)
	bar := qt.NewQHBoxLayout(controls)
	u.cancelRun = qt.NewQPushButton2()
	u.deleteProject = qt.NewQPushButton2()
	u.refresh = qt.NewQPushButton2()
	u.m2Open = qt.NewQPushButton2()
	bar.AddWidget(u.cancelRun.QWidget)
	bar.AddWidget(u.deleteProject.QWidget)
	bar.AddWidget(u.refresh.QWidget)
	bar.AddWidget(u.m2Open.QWidget)
	outer.AddWidget(controls)
	u.status = qt.NewQLabel2()
	u.status.SetWordWrap(true)
	outer.AddWidget(u.status.QWidget)
	u.progress = qt.NewQProgressBar2()
	u.progress.SetRange(0, u.maxPages.Value())
	outer.AddWidget(u.progress.QWidget)
	u.runs = qt.NewQComboBox2()
	outer.AddWidget(u.runs.QWidget)
	u.coverage = qt.NewQLabel2()
	u.coverage.SetWordWrap(true)
	outer.AddWidget(u.coverage.QWidget)
	u.results = qt.NewQPlainTextEdit2()
	u.results.SetReadOnly(true)
	u.results.SetTabChangesFocus(true)
	outer.AddWidget(u.results.QWidget)
	u.timer = qt.NewQTimer2(u.dialog.QObject)
	u.confirmDelete = func() bool {
		return qt.QMessageBox_Question6(u.dialog.QWidget, u.tr("scan_delete"), u.tr("scan_delete_confirm"),
			qt.QMessageBox__Yes|qt.QMessageBox__No, qt.QMessageBox__No) == qt.QMessageBox__Yes
	}
	u.timer.OnTimeout(u.poll)
	u.timer.Start(100)
	u.projects.OnCurrentIndexChanged(func(int) {
		u.start.SetEnabled(u.selectedProjectID() != "" && u.done == nil)
		u.deleteProject.SetEnabled(u.selectedProjectID() != "" && u.done == nil)
		u.refreshRuns()
	})
	u.runs.OnCurrentIndexChanged(func(int) {
		u.showRun()
		if u.m2 != nil {
			u.m2.updateRun()
			u.m2.refreshControls()
		}
	})
	u.create.OnClicked(u.createProject)
	u.deleteProject.OnClicked(u.removeProject)
	u.start.OnClicked(u.startScan)
	u.cancelRun.OnClicked(func() {
		if u.cancel != nil {
			u.cancel()
		}
	})
	u.refresh.OnClicked(func() { u.refreshProjects(); u.refreshRuns() })
	u.m2Open.OnClicked(func() { u.m2.show() })
	u.follow.OnToggled(func(on bool) { u.maxDepth.SetEnabled(on) })
	u.maxDepth.SetEnabled(false)
	u.cancelRun.SetEnabled(false)
	u.maxPages.OnValueChanged(func(int) { u.progress.SetRange(0, u.maxPages.Value()) })
	u.translate()
	u.refreshProjects()
	u.m2 = newM2UI(u, cache, cacheErr, trust, trustErr)
	return u
}

func (u *scannerUI) tr(key string, args ...any) string { return u.w.tr(key, args...) }
func (u *scannerUI) translate() {
	u.dialog.SetWindowTitle(u.tr("scan_title"))
	u.intro.SetText(u.tr("scan_intro"))
	for key, label := range u.labels {
		label.SetText(u.tr(key))
	}
	u.confirmed.SetText(u.tr("scan_confirmed"))
	u.follow.SetText(u.tr("scan_follow"))
	u.create.SetText(u.tr("scan_create"))
	u.deleteProject.SetText(u.tr("scan_delete"))
	u.start.SetText(u.tr("scan_start"))
	u.cancelRun.SetText(u.tr("scan_cancel"))
	u.refresh.SetText(u.tr("scan_refresh"))
	u.m2Open.SetText(u.tr("m2_open"))
	u.mode.SetItemText(0, u.tr("scan_loopback"))
	u.mode.SetItemText(1, u.tr("scan_public"))
	if u.projects.Count() > 0 {
		u.projects.SetItemText(0, u.tr("scan_select_project"))
	}
	for i, id := range u.runIDs {
		if run, err := u.store.LoadScanRun(context.Background(), id); err == nil {
			u.runs.SetItemText(i, u.runLabel(run))
		}
	}
	u.results.SetAccessibleName(u.tr("scan_results"))
	u.runs.SetAccessibleName(u.tr("scan_runs"))
	u.progress.SetAccessibleName(u.tr("scan_progress"))
	u.progress.SetFormat(u.tr("scan_progress_format"))
	u.showRun()
	if u.m2 != nil {
		u.m2.translate()
	}
}

func (u *scannerUI) show() { u.dialog.Show(); u.dialog.Raise(); u.dialog.ActivateWindow() }

func (u *scannerUI) refreshProjects() {
	if u.store == nil {
		u.status.SetText(u.tr("scan_store_error") + "\n" + u.tr("scan_error", u.initErr))
		u.create.SetEnabled(false)
		u.start.SetEnabled(false)
		return
	}
	projects, err := u.store.ListProjects(context.Background())
	if err != nil {
		u.status.SetText(u.tr("scan_error", err))
		return
	}
	selected := u.selectedProjectID()
	u.projects.Clear()
	u.projectIDs = []string{""}
	u.projects.AddItem(u.tr("scan_select_project"))
	for _, p := range projects {
		u.projectIDs = append(u.projectIDs, p.ID())
		u.projects.AddItem(p.Name() + " (" + p.ID() + ")")
	}
	for i, id := range u.projectIDs {
		if id == selected {
			u.projects.SetCurrentIndex(i)
			break
		}
	}
	u.deleteProject.SetEnabled(u.selectedProjectID() != "" && u.done == nil)
	u.start.SetEnabled(u.selectedProjectID() != "" && u.done == nil)
}

func (u *scannerUI) selectedProjectID() string {
	i := u.projects.CurrentIndex()
	if i < 0 || i >= len(u.projectIDs) {
		return ""
	}
	return u.projectIDs[i]
}

func (u *scannerUI) createProject() {
	if u.store == nil {
		return
	}
	expires, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(u.expiry.Text()), time.Local)
	if err != nil {
		u.status.SetText(u.tr("scan_invalid_date"))
		return
	}
	p, err := project.New(project.Draft{ID: strings.TrimSpace(u.id.Text()), Name: strings.TrimSpace(u.name.Text()),
		TargetOwner: strings.TrimSpace(u.owner.Text()), AuthorizationReference: strings.TrimSpace(u.reference.Text()),
		AuthorizationConfirmed: u.confirmed.IsChecked(), AuthorizationExpiresAt: expires.AddDate(0, 0, 1),
		Origins: []string{strings.TrimSpace(u.origin.Text())}})
	if err == nil {
		err = u.store.CreateProject(context.Background(), p)
	}
	if err != nil {
		u.status.SetText(u.tr("scan_error", err))
		return
	}
	u.confirmed.SetChecked(false)
	u.refreshProjects()
	for i, id := range u.projectIDs {
		if id == p.ID() {
			u.projects.SetCurrentIndex(i)
			break
		}
	}
	u.status.SetText(u.tr("scan_created"))
}

func (u *scannerUI) removeProject() {
	id := u.selectedProjectID()
	if id == "" || u.store == nil || u.done != nil {
		return
	}
	if !u.confirmDelete() {
		return
	}
	if err := u.store.DeleteProject(context.Background(), id); err != nil {
		u.status.SetText(u.tr("scan_error", err))
		return
	}
	u.refreshProjects()
	u.refreshRuns()
	u.status.SetText(u.tr("scan_deleted"))
}

func splitPaths(text string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == '\n' }) {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (u *scannerUI) plan() (scanner.CrawlPlan, error) {
	id := u.selectedProjectID()
	if id == "" {
		return scanner.CrawlPlan{}, scanner.ErrInvalidPlan
	}
	p, err := u.store.LoadProject(context.Background(), id)
	if err != nil {
		return scanner.CrawlPlan{}, err
	}
	origins := p.Origins()
	if len(origins) != 1 {
		return scanner.CrawlPlan{}, scanner.ErrInvalidPlan
	} // M1 dialog supports one origin per project
	policy, err := scope.NewRequestPolicy([]string{"GET"}, splitPaths(u.allowed.Text()), splitPaths(u.excluded.Text()))
	if err != nil {
		return scanner.CrawlPlan{}, err
	}
	mode := scanner.CrawlLoopback
	addresses := []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("::1")}
	parsed, err := url.Parse(origins[0])
	if err != nil {
		return scanner.CrawlPlan{}, scanner.ErrInvalidPlan
	}
	if ip, err := netip.ParseAddr(parsed.Hostname()); err == nil && ip.IsLoopback() {
		addresses = []netip.Addr{ip}
	}
	if u.mode.CurrentIndex() == 1 {
		mode = scanner.CrawlPinnedPublic
		addresses = nil
		for _, part := range splitPaths(u.pins.Text()) {
			ip, err := netip.ParseAddr(part)
			if err != nil {
				return scanner.CrawlPlan{}, transport.ErrConfig
			}
			addresses = append(addresses, ip)
		}
		if len(addresses) == 0 {
			return scanner.CrawlPlan{}, transport.ErrConfig
		}
	}
	depth := 0
	if u.follow.IsChecked() {
		depth = u.maxDepth.Value()
	}
	return scanner.CrawlPlan{ProjectID: id, SeedURLs: []string{strings.TrimSpace(u.seed.Text())}, Mode: mode,
		Grants: []transport.Grant{{Origin: origins[0], Addresses: addresses}}, Policy: policy,
		Limits: transport.Limits{MaxRequests: u.budget.Value(), MaxConcurrent: 1, MaxRedirects: 3,
			MaxBodyBytes: 1 << 20, RequestTimeout: 10 * time.Second, RunTimeout: 5 * time.Minute,
			MinRequestInterval: 250 * time.Millisecond}, Resolver: net.DefaultResolver,
		MaxPages: u.maxPages.Value(), MaxDepth: depth, FollowLinks: u.follow.IsChecked()}, nil
}

func (u *scannerUI) startScan() {
	if u.store == nil || u.done != nil {
		return
	}
	plan, err := u.plan()
	if err != nil {
		u.status.SetText(u.tr("scan_error", err))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = cancel
	u.done = make(chan struct{})
	u.progressUpdates = make(chan int, 1)
	u.finished = make(chan scanCompletion, 1)
	progressUpdates, finished, done := u.progressUpdates, u.finished, u.done
	u.progress.SetValue(0)
	u.status.SetText(u.tr("scan_running"))
	u.start.SetEnabled(false)
	u.deleteProject.SetEnabled(false)
	u.cancelRun.SetEnabled(true)
	plan.OnProgress = func(count int) {
		select {
		case progressUpdates <- count:
		default:
		}
	}
	go func() {
		defer close(done)
		report, err := scanner.RunCrawl(ctx, u.store, plan)
		finished <- scanCompletion{report: report, err: err}
	}()
}

func (u *scannerUI) poll() {
	if u.done == nil {
		return
	}
	select {
	case count := <-u.progressUpdates:
		u.progress.SetValue(count)
	default:
	}
	select {
	case result := <-u.finished:
		u.cancel()
		u.cancel = nil
		u.done = nil
		u.cancelRun.SetEnabled(false)
		u.start.SetEnabled(u.selectedProjectID() != "")
		u.deleteProject.SetEnabled(u.selectedProjectID() != "")
		u.progress.SetValue(result.report.CompletedVisits)
		if result.err != nil {
			u.status.SetText(u.tr("scan_error", result.err))
		} else {
			u.status.SetText(u.tr("scan_finished"))
		}
		u.refreshRuns()
	default:
	}
}

func (u *scannerUI) refreshRuns() {
	u.runs.Clear()
	u.runIDs = nil
	id := u.selectedProjectID()
	if id == "" || u.store == nil {
		u.showRun()
		return
	}
	runs, err := u.store.ListScanRuns(context.Background(), id)
	if err != nil {
		u.status.SetText(u.tr("scan_error", err))
		return
	}
	for _, run := range runs {
		u.runIDs = append(u.runIDs, run.ID)
		u.runs.AddItem(u.runLabel(run))
	}
	u.showRun()
}

func (u *scannerUI) runLabel(run storage.ScanRun) string {
	return fmt.Sprintf("%s · %s · %s", run.StartedAt.Local().Format("2006-01-02 15:04"), u.tr("scan_state_"+run.State), run.ID[:8])
}

func (u *scannerUI) showRun() {
	if u.results == nil {
		return
	}
	i := u.runs.CurrentIndex()
	if i < 0 || i >= len(u.runIDs) || u.store == nil {
		u.results.Clear()
		u.coverage.SetText(u.tr("scan_no_runs"))
		if u.m2 != nil {
			u.m2.updateRun()
			u.m2.refreshControls()
		}
		return
	}
	run, err := u.store.LoadScanRun(context.Background(), u.runIDs[i])
	if err != nil {
		u.results.Clear()
		u.coverage.SetText(u.tr("scan_error", err))
		return
	}
	if run.CoverageLimited || run.State != "complete" {
		u.coverage.SetText(u.tr("scan_incomplete"))
	} else {
		u.coverage.SetText(u.tr("scan_queue_complete"))
	}
	lines := []string{u.tr("scan_summary", run.CompletedVisits, run.RequestsUsed, u.tr("scan_state_"+run.State))}
	if run.StopCode != "" {
		lines = append(lines, u.tr("scan_stop", run.StopCode))
	}
	for _, v := range run.Visits {
		lines = append(lines, u.tr("scan_visit", v.VisitIndex+1, v.StatusCode, v.RuleID, v.RuleRevision, u.tr("scan_outcome_"+v.Outcome),
			u.tr("scan_evidence_"+v.EvidenceCode), v.EvidenceCode,
			u.tr("scan_discovery_"+v.DiscoveryStatus), v.Links, v.Forms, v.OutOfScope, v.Invalid))
	}
	u.results.SetPlainText(strings.Join(lines, "\n"))
	if u.m2 != nil {
		u.m2.updateRun()
		u.m2.refreshControls()
	}
}

func (u *scannerUI) dispose() {
	if u == nil {
		return
	}
	if u.cancel != nil {
		u.cancel()
	}
	if u.done != nil {
		<-u.done
	}
	u.timer.Stop()
	if u.m2 != nil {
		u.m2.dispose()
	}
	u.dialog.Close()
	u.dialog.Delete()
}

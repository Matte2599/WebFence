package desktop

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Matte2599/WebFence/internal/checks"
	"github.com/Matte2599/WebFence/internal/scanner"
	"github.com/Matte2599/WebFence/internal/session"
	"github.com/Matte2599/WebFence/internal/transport"
	qt "github.com/mappu/miqt/qt6"
)

// authUI holds test credentials only for one active run. It never stores them
// in the project database, report or a widget after starting the worker.
type authUI struct {
	parent                                             *scannerUI
	dialog                                             *qt.QDialog
	timer                                              *qt.QTimer
	labels                                             map[string]*qt.QLabel
	intro, status, originLabel, selectedOrigin         *qt.QLabel
	loginURL, verifyURL, resourceURL, privateBody      *qt.QLineEdit
	usernameField, passwordField, cookieName           *qt.QLineEdit
	csrfField, csrfCookieName                          *qt.QLineEdit
	ownerID, ownerUsername, ownerPassword, ownerMarker *qt.QLineEdit
	otherID, otherUsername, otherPassword, otherMarker *qt.QLineEdit
	loginConfirmed, publicConfirmed                    *qt.QCheckBox
	resourceConfirmed, otherForbidden                  *qt.QCheckBox
	start, cancelButton                                *qt.QPushButton
	result                                             *qt.QPlainTextEdit
	cancel                                             context.CancelFunc
	done                                               chan struct{}
	finished                                           chan authCompletion
	busy                                               bool
}

type authCompletion struct {
	result checks.CrossRoleResult
	err    error
}

type authRunSecrets struct{ owner, other []byte }

func (s *authRunSecrets) Get(ctx context.Context, id string) ([]byte, error) {
	if s == nil || ctx == nil || ctx.Err() != nil {
		return nil, errors.New("test_secret_unavailable")
	}
	switch id {
	case "owner-run-secret":
		return append([]byte(nil), s.owner...), nil
	case "other-run-secret":
		return append([]byte(nil), s.other...), nil
	default:
		return nil, errors.New("test_secret_unavailable")
	}
}

func (s *authRunSecrets) close() {
	if s != nil {
		clear(s.owner)
		clear(s.other)
		s.owner, s.other = nil, nil
	}
}

func newAuthUI(parent *scannerUI) *authUI {
	a := &authUI{parent: parent, labels: make(map[string]*qt.QLabel)}
	a.dialog = qt.NewQDialog(parent.dialog.QWidget)
	a.dialog.Resize(700, 740)
	outer := qt.NewQVBoxLayout(a.dialog.QWidget)
	a.intro = qt.NewQLabel2()
	a.intro.SetWordWrap(true)
	outer.AddWidget(a.intro.QWidget)
	a.originLabel, a.selectedOrigin = qt.NewQLabel2(), qt.NewQLabel2()
	a.selectedOrigin.SetWordWrap(true)
	outer.AddWidget(a.originLabel.QWidget)
	outer.AddWidget(a.selectedOrigin.QWidget)
	scroll := qt.NewQScrollArea2()
	scroll.SetWidgetResizable(true)
	body := qt.NewQWidget(nil)
	form := qt.NewQFormLayout(body)
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.SetRowWrapPolicy(qt.QFormLayout__WrapLongRows)
	row := func(key string, field *qt.QLineEdit, max int) {
		field.SetMaxLength(max)
		label := qt.NewQLabel2()
		label.SetBuddy(field.QWidget)
		a.labels[key] = label
		form.AddRow(label.QWidget, field.QWidget)
	}
	a.loginURL, a.verifyURL, a.resourceURL, a.privateBody = qt.NewQLineEdit2(), qt.NewQLineEdit2(), qt.NewQLineEdit2(), qt.NewQLineEdit2()
	row("auth_login_url", a.loginURL, 2048)
	row("auth_verify_url", a.verifyURL, 2048)
	row("auth_resource_url", a.resourceURL, 2048)
	row("auth_private_body", a.privateBody, 1024)
	a.usernameField, a.passwordField, a.cookieName = qt.NewQLineEdit2(), qt.NewQLineEdit2(), qt.NewQLineEdit2()
	a.usernameField.SetText("username")
	a.passwordField.SetText("password")
	a.cookieName.SetText("sid")
	row("auth_username_field", a.usernameField, 32)
	row("auth_password_field", a.passwordField, 32)
	row("auth_cookie_name", a.cookieName, 64)
	a.csrfField, a.csrfCookieName = qt.NewQLineEdit2(), qt.NewQLineEdit2()
	row("auth_csrf_field", a.csrfField, 32)
	row("auth_csrf_cookie_name", a.csrfCookieName, 64)
	a.ownerID, a.ownerUsername, a.ownerPassword, a.ownerMarker = qt.NewQLineEdit2(), qt.NewQLineEdit2(), qt.NewQLineEdit2(), qt.NewQLineEdit2()
	a.ownerPassword.SetEchoMode(qt.QLineEdit__Password)
	row("auth_owner_id", a.ownerID, 64)
	row("auth_owner_username", a.ownerUsername, 128)
	row("auth_owner_password", a.ownerPassword, 2048)
	row("auth_owner_marker", a.ownerMarker, 1024)
	a.otherID, a.otherUsername, a.otherPassword, a.otherMarker = qt.NewQLineEdit2(), qt.NewQLineEdit2(), qt.NewQLineEdit2(), qt.NewQLineEdit2()
	a.otherPassword.SetEchoMode(qt.QLineEdit__Password)
	row("auth_other_id", a.otherID, 64)
	row("auth_other_username", a.otherUsername, 128)
	row("auth_other_password", a.otherPassword, 2048)
	row("auth_other_marker", a.otherMarker, 1024)
	a.loginConfirmed, a.publicConfirmed = qt.NewQCheckBox2(), qt.NewQCheckBox2()
	a.resourceConfirmed, a.otherForbidden = qt.NewQCheckBox2(), qt.NewQCheckBox2()
	form.AddRowWithWidget(a.loginConfirmed.QWidget)
	form.AddRowWithWidget(a.publicConfirmed.QWidget)
	form.AddRowWithWidget(a.resourceConfirmed.QWidget)
	form.AddRowWithWidget(a.otherForbidden.QWidget)
	scroll.SetWidget(body)
	outer.AddWidget(scroll.QWidget)
	a.start, a.cancelButton = qt.NewQPushButton2(), qt.NewQPushButton2()
	buttons := qt.NewQWidget(nil)
	bar := qt.NewQHBoxLayout(buttons)
	bar.AddWidget(a.start.QWidget)
	bar.AddWidget(a.cancelButton.QWidget)
	outer.AddWidget(buttons)
	a.cancelButton.SetEnabled(false)
	a.status = qt.NewQLabel2()
	a.status.SetWordWrap(true)
	outer.AddWidget(a.status.QWidget)
	a.result = qt.NewQPlainTextEdit2()
	a.result.SetReadOnly(true)
	a.result.SetMaximumHeight(110)
	outer.AddWidget(a.result.QWidget)
	a.timer = qt.NewQTimer2(a.dialog.QObject)
	a.timer.OnTimeout(a.poll)
	a.timer.Start(100)
	a.start.OnClicked(a.startRun)
	a.cancelButton.OnClicked(func() {
		if a.cancel != nil {
			a.cancel()
		}
	})
	a.dialog.OnRejected(func() {
		if a.cancel != nil {
			a.cancel()
		}
		a.ownerPassword.Clear()
		a.otherPassword.Clear()
		a.loginConfirmed.SetChecked(false)
		a.publicConfirmed.SetChecked(false)
		a.resourceConfirmed.SetChecked(false)
		a.otherForbidden.SetChecked(false)
	})
	a.translate()
	return a
}

func (a *authUI) tr(key string) string { return a.parent.tr(key) }

func (a *authUI) translate() {
	a.dialog.SetWindowTitle(a.tr("auth_title"))
	a.intro.SetText(a.tr("auth_intro"))
	a.originLabel.SetText(a.tr("auth_selected_origin"))
	for key, label := range a.labels {
		label.SetText(a.tr(key))
	}
	a.loginConfirmed.SetText(a.tr("auth_confirm_login"))
	a.publicConfirmed.SetText(a.tr("auth_confirm_public"))
	a.resourceConfirmed.SetText(a.tr("auth_confirm_resource"))
	a.otherForbidden.SetText(a.tr("auth_confirm_other_forbidden"))
	a.start.SetText(a.tr("auth_start"))
	a.cancelButton.SetText(a.tr("auth_cancel"))
}

func (a *authUI) show() {
	a.selectedOrigin.SetText("")
	a.publicConfirmed.SetChecked(false)
	if a.parent.store != nil {
		if p, err := a.parent.store.LoadProject(context.Background(), a.parent.selectedProjectID()); err == nil {
			if origins := p.Origins(); len(origins) == 1 {
				a.selectedOrigin.SetText(origins[0])
			}
		}
	}
	a.dialog.Show()
	a.dialog.Raise()
	a.dialog.ActivateWindow()
}

// authModeAndLimits keeps a public credential run separate from the scan's
// broader traffic settings. The selected budget is never silently increased
// or reduced; an unsafe public setting is rejected before secrets are used.
func authModeAndLimits(crawl scanner.CrawlPlan, publicConfirmed bool, visibleOrigin string) (checks.CrossRoleRunMode, transport.Limits, error) {
	if len(crawl.Grants) != 1 {
		return "", transport.Limits{}, transport.ErrConfig
	}
	limits := crawl.Limits
	switch crawl.Mode {
	case scanner.CrawlLoopback:
		return checks.CrossRoleRunLoopback, limits, nil
	case scanner.CrawlPinnedPublic:
		if !publicConfirmed || visibleOrigin != crawl.Grants[0].Origin ||
			!strings.HasPrefix(visibleOrigin, "https://") || limits.MaxRequests < 1 || limits.MaxRequests > 128 ||
			limits.RunTimeout <= 0 || limits.RunTimeout > 15*time.Minute || limits.MaxConcurrent != 1 {
			return "", transport.Limits{}, transport.ErrConfig
		}
		if limits.MinRequestInterval < 500*time.Millisecond {
			limits.MinRequestInterval = 500 * time.Millisecond
		}
		if limits.MaxBodyBytes > 64<<10 {
			limits.MaxBodyBytes = 64 << 10
		}
		limits.MaxRedirects = 0
		return checks.CrossRoleRunPinnedPublic, limits, nil
	default:
		return "", transport.Limits{}, transport.ErrConfig
	}
}

func (a *authUI) startRun() {
	u := a.parent
	if a.busy || u.store == nil || u.done != nil || u.selectedProjectID() == "" {
		return
	}
	crawl, err := u.plan()
	if err != nil {
		a.ownerPassword.Clear()
		a.otherPassword.Clear()
		a.status.SetText(a.tr("auth_invalid"))
		return
	}
	mode, limits, err := authModeAndLimits(crawl, a.publicConfirmed.IsChecked(), a.selectedOrigin.Text())
	if err != nil {
		a.ownerPassword.Clear()
		a.otherPassword.Clear()
		a.publicConfirmed.SetChecked(false)
		a.status.SetText(a.tr("auth_invalid"))
		return
	}
	secrets := &authRunSecrets{owner: []byte(a.ownerPassword.Text()), other: []byte(a.otherPassword.Text())}
	a.ownerPassword.Clear()
	a.otherPassword.Clear()
	if len(secrets.owner) == 0 || len(secrets.owner) > 2048 || len(secrets.other) == 0 || len(secrets.other) > 2048 {
		secrets.close()
		a.status.SetText(a.tr("auth_invalid"))
		return
	}
	usernameField, passwordField, cookieName := strings.TrimSpace(a.usernameField.Text()), strings.TrimSpace(a.passwordField.Text()), strings.TrimSpace(a.cookieName.Text())
	csrfField, csrfCookieName := strings.TrimSpace(a.csrfField.Text()), strings.TrimSpace(a.csrfCookieName.Text())
	account := func(id, username, marker, secretID string) session.Account {
		return session.Account{ID: strings.TrimSpace(id), Username: username, SecretID: secretID,
			UsernameField: usernameField, PasswordField: passwordField, CookieName: cookieName,
			ExpectedBody: marker, CSRFField: csrfField, CSRFCookieName: csrfCookieName}
	}
	plan := checks.CrossRoleRunPlan{ProjectID: crawl.ProjectID, Mode: mode,
		Origin: crawl.Grants[0].Origin,
		Grant:  transport.Grant{Origin: crawl.Grants[0].Origin, Addresses: crawl.Grants[0].Addresses},
		Policy: crawl.Policy, Limits: limits, Resolver: crawl.Resolver,
		Routes: transport.SessionRoutes{LoginURL: strings.TrimSpace(a.loginURL.Text()),
			VerifyURL: strings.TrimSpace(a.verifyURL.Text()), LoginConfirmed: a.loginConfirmed.IsChecked(),
			PublicConfirmed: a.publicConfirmed.IsChecked() && mode == checks.CrossRoleRunPinnedPublic},
		Owner: account(a.ownerID.Text(), a.ownerUsername.Text(), a.ownerMarker.Text(), "owner-run-secret"),
		Other: account(a.otherID.Text(), a.otherUsername.Text(), a.otherMarker.Text(), "other-run-secret"),
		Check: checks.CrossRolePlan{ResourceURL: strings.TrimSpace(a.resourceURL.Text()),
			PrivateBody: a.privateBody.Text(), ResourceConfirmed: a.resourceConfirmed.IsChecked(),
			OtherForbiddenConfirmed: a.otherForbidden.IsChecked()},
	}
	if !session.ValidAccount(plan.Owner) || !session.ValidAccount(plan.Other) ||
		!plan.Routes.LoginConfirmed || !plan.Check.ResourceConfirmed || !plan.Check.OtherForbiddenConfirmed ||
		(mode == checks.CrossRoleRunPinnedPublic && !plan.Routes.PublicConfirmed) {
		secrets.close()
		a.status.SetText(a.tr("auth_invalid"))
		return
	}
	// Confirmations apply to this immutable plan only. A later run, including
	// one with the same fields, needs a new explicit operator decision.
	a.loginConfirmed.SetChecked(false)
	a.publicConfirmed.SetChecked(false)
	a.resourceConfirmed.SetChecked(false)
	a.otherForbidden.SetChecked(false)
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})
	a.finished = make(chan authCompletion, 1)
	a.busy = true
	a.result.Clear()
	a.status.SetText(a.tr("auth_running"))
	a.start.SetEnabled(false)
	a.cancelButton.SetEnabled(true)
	u.start.SetEnabled(false)
	u.importAPI.SetEnabled(false)
	u.scanAPI.SetEnabled(false)
	u.deleteProject.SetEnabled(false)
	u.authOpen.SetEnabled(false)
	u.projects.SetEnabled(false)
	u.refresh.SetEnabled(false)
	finished, done := a.finished, a.done
	go func() {
		defer close(done)
		defer secrets.close()
		result, err := checks.RunCrossRole(ctx, u.store, secrets, plan)
		finished <- authCompletion{result: result, err: err}
	}()
}

func (a *authUI) poll() {
	if !a.busy {
		return
	}
	select {
	case completed := <-a.finished:
		a.cancel()
		a.cancel, a.done, a.busy = nil, nil, false
		a.start.SetEnabled(true)
		a.cancelButton.SetEnabled(false)
		u := a.parent
		u.projects.SetEnabled(true)
		u.refresh.SetEnabled(true)
		available := u.selectedProjectID() != "" && u.done == nil
		u.start.SetEnabled(available)
		u.importAPI.SetEnabled(available)
		u.scanAPI.SetEnabled(available)
		u.deleteProject.SetEnabled(available)
		u.authOpen.SetEnabled(available)
		switch {
		case completed.err != nil:
			a.status.SetText(a.tr("auth_error"))
		case completed.result.Outcome == checks.Finding:
			a.status.SetText(a.tr("auth_finding"))
			a.result.SetPlainText(completed.result.EvidenceCode)
		case completed.result.Outcome == checks.Inconclusive:
			a.status.SetText(a.tr("auth_inconclusive"))
			a.result.SetPlainText(completed.result.EvidenceCode)
		default:
			a.status.SetText(a.tr("auth_error"))
		}
	default:
	}
}

func (a *authUI) dispose() {
	if a == nil {
		return
	}
	if a.cancel != nil {
		a.cancel()
	}
	if a.done != nil {
		<-a.done
	}
	a.ownerPassword.Clear()
	a.otherPassword.Clear()
	a.publicConfirmed.SetChecked(false)
	a.timer.Stop()
	a.dialog.Close()
	a.dialog.Delete()
}

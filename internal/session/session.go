// Package session holds one ephemeral test-account session per successful
// login. It never persists credentials or cookies and never treats failed
// authentication as a negative security finding.
package session

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Matte2599/WebFence/internal/transport"
	"golang.org/x/net/html"
)

var (
	ErrConfig       = errors.New("session_invalid_configuration")
	ErrLoginInvalid = errors.New("session_login_invalid")
	ErrInconclusive = errors.New("session_verification_inconclusive")
	ErrExpired      = errors.New("session_expired")
)

type SecretSource interface {
	Get(context.Context, string) ([]byte, error)
}

// Account describes one explicitly authorized test identity. SecretID is an
// opaque reference resolved by the supplied SecretSource, which can be a
// native credential store or a one-run source. ExpectedBody must be an exact,
// non-secret marker returned only for this identity by the declared validity URL.
type Account struct {
	ID            string
	Username      string
	SecretID      string
	UsernameField string
	PasswordField string
	CookieName    string
	ExpectedBody  string
	// CSRFField selects exactly one hidden input on the confirmed login page.
	// CSRFCookieName optionally selects one host-only pre-session cookie for POST.
	CSRFField      string
	CSRFCookieName string
}

type Manager struct {
	broker  *transport.Broker
	secrets SecretSource
}

func NewManager(broker *transport.Broker, secrets SecretSource) (*Manager, error) {
	if broker == nil || broker.SessionOrigin() == "" || broker.SessionProjectID() == "" ||
		broker.SessionDeadline().IsZero() || secrets == nil {
		return nil, ErrConfig
	}
	return &Manager{broker: broker, secrets: secrets}, nil
}

func validToken(s string, max int) bool {
	if len(s) == 0 || len(s) > max {
		return false
	}
	for _, c := range []byte(s) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func ValidAccount(a Account) bool {
	return validToken(a.ID, 64) && len(a.Username) > 0 && len(a.Username) <= 128 && utf8.ValidString(a.Username) &&
		!strings.ContainsAny(a.Username, "\r\n\x00") && validToken(a.SecretID, 64) &&
		validToken(a.UsernameField, 32) && validToken(a.PasswordField, 32) &&
		a.UsernameField != a.PasswordField && validToken(a.CookieName, 64) &&
		len(a.ExpectedBody) > 0 && len(a.ExpectedBody) <= 1024 &&
		(a.CSRFField == "" && a.CSRFCookieName == "" ||
			validToken(a.CSRFField, 32) && (a.CSRFCookieName == "" || validToken(a.CSRFCookieName, 64)) &&
				a.CSRFField != a.UsernameField && a.CSRFField != a.PasswordField)
}

// Login probes the validity route anonymously, posts the test credential once,
// then requires a distinct exact positive response with the captured host-only
// cookie. It returns no Session on any ambiguous or failed login.
func (m *Manager) Login(ctx context.Context, a Account) (*Session, error) {
	if m == nil || m.broker == nil || ctx == nil || !ValidAccount(a) {
		return nil, ErrConfig
	}
	baseline, err := m.broker.VerifyAnonymous(ctx)
	if err != nil || baseline.StatusCode >= 500 || baseline.StatusCode == 0 {
		return nil, ErrInconclusive
	}
	if baseline.StatusCode == http.StatusOK && string(baseline.Body) == a.ExpectedBody {
		return nil, ErrInconclusive
	}
	var token, preCookie []byte
	if a.CSRFField != "" {
		page, fetchErr := m.broker.FetchLoginForm(ctx)
		if fetchErr != nil || page.StatusCode != http.StatusOK {
			return nil, ErrInconclusive
		}
		mediaType, _, mediaErr := mime.ParseMediaType(page.Header.Get("Content-Type"))
		if mediaErr != nil || mediaType != "text/html" {
			clear(page.Body)
			return nil, ErrInconclusive
		}
		token, err = hiddenToken(page.Body, a.CSRFField)
		clear(page.Body)
		if err != nil {
			return nil, ErrInconclusive
		}
		defer clear(token)
		if a.CSRFCookieName != "" {
			preCookie, _, _, err = selectCookie(page.Header, a.CSRFCookieName, m.broker.SessionOrigin(),
				m.broker.LoginURL(), m.broker.SessionDeadline())
			if err != nil {
				return nil, ErrInconclusive
			}
			defer clear(preCookie)
		}
	}
	secret, err := m.secrets.Get(ctx, a.SecretID)
	if err != nil || len(secret) == 0 || len(secret) > 2048 {
		clear(secret)
		return nil, ErrInconclusive
	}
	// Go and the OS can retain copies despite clearing the returned byte slice.
	form := url.Values{a.UsernameField: {a.Username}, a.PasswordField: {string(secret)}}
	if len(token) != 0 {
		form.Set(a.CSRFField, string(token))
	}
	clear(secret)
	encoded := []byte(form.Encode())
	defer clear(encoded)
	login, err := m.broker.LoginFormWithCookie(ctx, m.broker.LoginURL(), encoded, string(preCookie))
	if err != nil {
		return nil, ErrInconclusive
	}
	if login.StatusCode == http.StatusUnauthorized || login.StatusCode == http.StatusForbidden {
		return nil, ErrLoginInvalid
	}
	if login.StatusCode != http.StatusOK && login.StatusCode != http.StatusFound &&
		login.StatusCode != http.StatusSeeOther {
		return nil, ErrInconclusive
	}
	cookie, cookiePath, expires, err := selectCookie(login.Header, a.CookieName, m.broker.SessionOrigin(),
		m.broker.VerifyURL(), m.broker.SessionDeadline())
	if err != nil {
		return nil, ErrLoginInvalid
	}
	if len(preCookie) != 0 && bytes.Equal(preCookie, cookie) {
		clear(cookie)
		return nil, ErrLoginInvalid // a pre-session must not become the authenticated session
	}
	verified, err := m.broker.FetchSession(ctx, m.broker.VerifyURL(), string(cookie))
	if err != nil || verified.StatusCode != http.StatusOK || string(verified.Body) != a.ExpectedBody {
		clear(cookie)
		return nil, ErrInconclusive
	}
	lifetime, stop := context.WithCancel(context.Background())
	return &Session{broker: m.broker, projectID: m.broker.SessionProjectID(),
		revision: m.broker.SessionRevision(), identity: a.ID, cookie: cookie,
		cookiePath: cookiePath, expiresAt: expires, expectedBody: a.ExpectedBody,
		lifetime: lifetime, stop: stop}, nil
}

func hiddenToken(body []byte, name string) ([]byte, error) {
	z := html.NewTokenizer(bytes.NewReader(body))
	var chosen []byte
	for {
		switch z.Next() {
		case html.ErrorToken:
			if z.Err() == io.EOF && len(chosen) != 0 {
				return chosen, nil
			}
			clear(chosen)
			return nil, ErrInconclusive
		case html.SelfClosingTagToken, html.StartTagToken:
			tag, hasAttr := z.TagName()
			if string(tag) != "input" || !hasAttr {
				continue
			}
			var field, kind, value string
			for {
				key, val, more := z.TagAttr()
				switch string(key) {
				case "name":
					field = string(val)
				case "type":
					kind = string(val)
				case "value":
					value = string(val)
				}
				if !more {
					break
				}
			}
			if field != name || !strings.EqualFold(kind, "hidden") {
				continue
			}
			if len(chosen) != 0 || len(value) == 0 || len(value) > 1024 || !utf8.ValidString(value) ||
				strings.ContainsAny(value, "\r\n\x00") {
				clear(chosen)
				return nil, ErrInconclusive
			}
			chosen = []byte(value)
		}
	}
}

func selectCookie(header http.Header, name, originRaw, verifyRaw string, deadline time.Time) ([]byte, string, time.Time, error) {
	u, err := url.Parse(originRaw)
	verify, verifyErr := url.Parse(verifyRaw)
	if err != nil || verifyErr != nil {
		return nil, "", time.Time{}, ErrConfig
	}
	var chosen *http.Cookie
	for _, cookie := range (&http.Response{Header: header}).Cookies() {
		if cookie.Name != name {
			continue
		}
		if chosen != nil || cookie.Domain != "" || cookie.Partitioned || cookie.MaxAge < 0 ||
			cookie.Quoted || cookie.Valid() != nil || len(cookie.Value) == 0 || len(cookie.Value) > 1024 ||
			(cookie.Secure && u.Scheme != "https") ||
			!cookiePathMatches(verify.EscapedPath(), cookie.Path) {
			return nil, "", time.Time{}, ErrLoginInvalid
		}
		chosen = cookie
	}
	if chosen == nil {
		return nil, "", time.Time{}, ErrLoginInvalid
	}
	if !chosen.Expires.IsZero() && chosen.Expires.Before(deadline) {
		deadline = chosen.Expires
	}
	if chosen.MaxAge > 0 {
		ageDeadline := time.Now().Add(time.Duration(chosen.MaxAge) * time.Second)
		if ageDeadline.Before(deadline) {
			deadline = ageDeadline
		}
	}
	if !deadline.After(time.Now()) {
		return nil, "", time.Time{}, ErrLoginInvalid
	}
	return []byte(chosen.Name + "=" + chosen.Value), chosen.Path, deadline, nil
}

func cookiePathMatches(path, cookiePath string) bool {
	if cookiePath == "" {
		return false // require an explicit path rather than guessing a default
	}
	return cookiePath == "/" || path == cookiePath ||
		(strings.HasPrefix(path, cookiePath) && strings.HasSuffix(cookiePath, "/")) ||
		strings.HasPrefix(path, cookiePath+"/")
}

// Session is bound to one project revision and test identity. Its cookie is
// never returned to callers. Closing or expiry prevents further requests.
type Session struct {
	broker       *transport.Broker
	projectID    string
	revision     uint64
	identity     string
	expectedBody string
	cookiePath   string
	expiresAt    time.Time
	lifetime     context.Context
	stop         context.CancelFunc
	mu           sync.Mutex
	cookie       []byte
	closed       bool
}

func (s *Session) Identity() string {
	if s == nil {
		return ""
	}
	return s.identity
}
func (s *Session) ProjectID() string {
	if s == nil {
		return ""
	}
	return s.projectID
}
func (s *Session) Revision() uint64 {
	if s == nil {
		return 0
	}
	return s.revision
}

// BoundTo prevents a cookie from being attached to a different broker/run,
// even when that broker uses the same project authorization revision.
func (s *Session) BoundTo(b *transport.Broker) bool {
	return s != nil && b != nil && s.broker == b
}

func (s *Session) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.cookie)
	s.cookie = nil
	s.closed = true
	if s.stop != nil {
		s.stop()
	}
}

func (s *Session) cookieCopy() ([]byte, error) {
	if s == nil {
		return nil, ErrConfig
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !time.Now().Before(s.expiresAt) {
		return nil, ErrExpired
	}
	return append([]byte(nil), s.cookie...), nil
}

// Verify checks the declared validity URL again before a dependent check.
// A changed or unavailable response makes the session inconclusive.
func (s *Session) Verify(ctx context.Context) error {
	if s == nil || ctx == nil {
		return ErrConfig
	}
	cookie, err := s.cookieCopy()
	if err != nil {
		return err
	}
	defer clear(cookie)
	run, cancel := s.requestContext(ctx)
	defer cancel()
	result, err := s.broker.FetchSession(run, s.broker.VerifyURL(), string(cookie))
	if err != nil || result.StatusCode != http.StatusOK || string(result.Body) != s.expectedBody {
		s.Close()
		return ErrInconclusive
	}
	return nil
}

// Fetch performs one same-origin, allowlisted GET and does not follow a
// redirect. Authentication failures invalidate the session; callers must
// report dependent checks as inconclusive, never as no finding.
func (s *Session) Fetch(ctx context.Context, raw string) (transport.Result, error) {
	if s == nil || ctx == nil {
		return transport.Result{}, ErrConfig
	}
	u, parseErr := url.Parse(raw)
	if parseErr != nil || !cookiePathMatches(u.EscapedPath(), s.cookiePath) {
		return transport.Result{}, ErrInconclusive
	}
	cookie, err := s.cookieCopy()
	if err != nil {
		return transport.Result{}, err
	}
	defer clear(cookie)
	run, cancel := s.requestContext(ctx)
	defer cancel()
	result, err := s.broker.FetchSession(run, raw, string(cookie))
	if err != nil {
		return transport.Result{}, err
	}
	if result.StatusCode == http.StatusUnauthorized || result.StatusCode == http.StatusForbidden ||
		result.StatusCode >= 300 && result.StatusCode < 400 {
		s.Close()
		return transport.Result{}, ErrInconclusive
	}
	return result, nil
}

func (s *Session) requestContext(ctx context.Context) (context.Context, context.CancelFunc) {
	run, cancel := context.WithDeadline(ctx, s.expiresAt)
	stop := context.AfterFunc(s.lifetime, cancel)
	return run, func() { stop(); cancel() }
}

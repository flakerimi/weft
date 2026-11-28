// Package auth signs people in to a Warp app: users kept through warp/data,
// passwords hashed with PBKDF2-HMAC-SHA256, sessions kept in a table and
// named by a signed cookie, the routes to sign in and out, four roles with
// permissions and a guard for other modules' routes, and the one-time setup
// of the owner, the first user.
//
// It is configured in .env: AUTH_SECRET (32 bytes or more) keys the cookie,
// AUTH_SESSION_DAYS (1 to 365, default 30) is how long a session lasts,
// AUTH_OWNER_EMAIL is the owner's address, AUTH_REGISTRATION (open, invite
// or closed, default invite) is who may ask for an account, and API_KEYS
// (on or off, default off) whether other programs may sign in with a key.
// They are read when the app starts. It mails its links through the mail
// module.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/mail"
)

// User is a person who can sign in. The password's hash is kept beside it in
// the users table and is never answered or read from a request.
type User struct {
	data.Record
	Email string `json:"email" warp:"required,email,unique"`
	Name  string `json:"name" warp:"max=120"`
	Role  string `json:"role" warp:"enum=owner|administrator|member|viewer"`
}

// userRow is a user as the users table keeps it, with the hash. No route takes
// or answers it; every answer is its User.
type userRow struct {
	data.Record
	Email        string `json:"email" warp:"unique" data:"sort,search"`
	Name         string `json:"name" data:"sort,search"`
	Role         string `json:"role" data:"sort,filter"`
	PasswordHash string `json:"password_hash"`
}

func (r userRow) user() User {
	return User{Record: r.Record, Email: r.Email, Name: r.Name, Role: r.Role}
}

// Module is the auth module, named "auth", which needs the data and mail
// modules before it. Registering adds its routes under /api/auth and the
// users store, and reads nothing; its Start reads the AUTH_ settings and,
// while no user exists, logs the link that sets up the owner. Grant adds
// permissions to a role.
func Module(opts ...Option) warp.Module {
	m := newModule(nil)
	for _, opt := range opts {
		if opt != nil {
			opt(&m.opts)
		}
	}
	return m
}

type module struct {
	app         *warp.App
	db          *data.DB
	users       *data.Store[userRow]
	sessions    *data.Store[sessionRow]
	tokens      *data.Store[tokenRow]
	invitations *data.Store[invitationRow]
	keys        *data.Store[keyRow]
	mails       *mails
	laterSlots  chan struct{}  // register's and forgot's mailings running
	pending     sync.WaitGroup // of those, for Stop
	base        context.Context
	cancelBase  context.CancelFunc
	stopping    bool         // set by Stop, under mu
	mail        *mail.Mailer // the mail module's, which sends the login's links
	opts        options
	roles       *roles // set by Register
	known       known  // the permissions Me may list

	perKey     *limiter // tries per email and address
	perAddress *limiter // tries per address, across emails
	perAccount *limiter // tries per email, across addresses

	mailPerAddress *limiter      // mails asked for per address, by register and forgot
	mailPerClient  *limiter      // mails asked for per client
	invitePerUser  *limiter      // invitations per inviter
	tokenPerClient *limiter      // token and password per client
	slots          chan struct{} // one per password hash running
	slotWait       time.Duration // how long a request waits for a slot
	hashes         atomic.Int64  // hashes begun, for tests
	// find is the user with an email, as stored; the users store's First.
	find func(ctx context.Context, email string) (userRow, bool, error)

	// Set by Start.
	secret       []byte
	maxAge       int // seconds
	owner        string
	registration string // open, invite or closed
	apiKeys      bool
	url, host    string // APP_URL and its host, for the mailed links

	mu         sync.Mutex
	setupHash  []byte // the SHA-256 of the setup token; nil when there is none
	setupToken string // the token itself, for SetupToken alone
	clock      func() time.Time
}

// newModule is a module whose limiters, sessions and tokens read the clock
// now, or the database's when now is nil.
func newModule(now func() time.Time) *module {
	m := &module{
		slots:      make(chan struct{}, runtime.GOMAXPROCS(0)),
		laterSlots: make(chan struct{}, maxLater),
		slotWait:   slotWait,
		clock:      now,
	}
	m.perKey = newLimiter(m.now, maxKeys, maxTries, window)
	m.perAddress = newLimiter(m.now, maxKeys, maxAddressTries, window)
	m.perAccount = newLimiter(m.now, maxKeys, maxAccountTries, accountWindow)
	m.mailPerAddress = newLimiter(m.now, maxKeys, maxMailsPerAddress, time.Hour)
	m.mailPerClient = newLimiter(m.now, maxKeys, maxMailsPerClient, time.Hour)
	m.invitePerUser = newLimiter(m.now, maxKeys, maxInvites, time.Hour)
	m.tokenPerClient = newLimiter(m.now, maxKeys, maxTokenTries, time.Minute)
	return m
}

// now is the time by the database's clock, which the stores write by and a
// test sets with data.DB's SetClock, so the login and the stores never
// disagree. A module made by newModule with a clock of its own, as the
// package's tests make one, reads that instead.
func (m *module) now() time.Time {
	m.mu.Lock()
	clock := m.clock
	m.mu.Unlock()
	switch {
	case clock != nil:
		return clock().UTC()
	case m.db != nil:
		return m.db.Now()
	}
	return time.Now().UTC()
}

func (m *module) Name() string { return "auth" }

func (m *module) Register(app *warp.App) error {
	var err error
	m.app = app
	m.mail = mail.For(app)
	r, mistakes := newRoles(m.opts.grants)
	for _, s := range mistakes {
		app.Mistake("%s", s)
	}
	m.roles = r
	m.db = data.For(app)
	m.users = data.NewStore[userRow](m.db, "users")
	m.sessions = data.NewStore[sessionRow](m.db, "sessions")
	m.tokens = data.NewStore[tokenRow](m.db, "auth_tokens")
	m.invitations = data.NewStore[invitationRow](m.db, "invitations")
	m.keys = data.NewStore[keyRow](m.db, "api_keys")
	if m.mails, err = parseMails(); err != nil {
		app.Mistake("%s", strings.TrimPrefix(err.Error(), "auth: "))
	}
	m.find = func(ctx context.Context, email string) (userRow, bool, error) {
		return m.users.First(ctx, "email", email)
	}
	warp.Provide(app, m)
	app.Wrap(m.carry)
	m.routes()
	m.mailRoutes()
	m.userRoutes()
	m.keyRoutes()
	return nil
}

// Start reads the AUTH_ settings and, when the users table is empty, makes a
// setup token and logs the link that uses it. The token is in the link's
// fragment, which a browser never sends, so it reaches no access log or
// Referer.
func (m *module) Start(ctx context.Context) error {
	cfg := m.app.Config()
	secret := cfg.Get("AUTH_SECRET")
	if len(secret) < minSecret {
		return fmt.Errorf("AUTH_SECRET is %d bytes; set it to 32 or more random ones in .env, for example from openssl rand -base64 48.", len(secret))
	}
	days := 30
	if v := cfg.Get("AUTH_SESSION_DAYS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || !digits(v) || n < 1 || n > 365 {
			return fmt.Errorf("AUTH_SESSION_DAYS is %q; set it to a whole number of days from 1 to 365 in .env.", v)
		}
		days = n
	}
	owner := normalEmail(cfg.Get("AUTH_OWNER_EMAIL"))
	if owner == "" {
		return errors.New("AUTH_OWNER_EMAIL is not set; set it to the address of the person who will own this app.")
	}
	if !isEmail(owner) {
		return fmt.Errorf("AUTH_OWNER_EMAIL is %q; set it to an email address, such as owner@example.com.", cfg.Get("AUTH_OWNER_EMAIL"))
	}
	registration := cfg.Get("AUTH_REGISTRATION")
	switch registration {
	case "":
		registration = registrationInvite
	case registrationOpen, registrationInvite, registrationClosed:
	default:
		return fmt.Errorf("AUTH_REGISTRATION is %q; set it to open, invite or closed in .env.", registration)
	}
	var apiKeys bool
	switch v := cfg.Get("API_KEYS"); v {
	case "on":
		apiKeys = true
	case "", "off":
	default:
		return fmt.Errorf("API_KEYS is %q; set it to on or off in .env.", v)
	}
	m.secret, m.maxAge, m.owner = []byte(secret), days*24*60*60, owner
	m.registration, m.apiKeys = registration, apiKeys
	m.setURL(cfg.URL)
	m.mu.Lock()
	m.base, m.cancelBase = context.WithCancel(context.Background())
	m.mu.Unlock()

	if err := m.expired(ctx, m.now()); err != nil {
		return fmt.Errorf("the login's tables cannot be written (%v); check the database DB_URL names.", err)
	}
	n, err := m.users.Count(ctx)
	if err != nil {
		return fmt.Errorf("the users table cannot be read (%v); check the database DB_URL names.", err)
	}
	if n > 0 {
		return nil
	}
	b := make([]byte, 32)
	rand.Read(b) // never fails, as crypto/rand documents
	token := base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	m.mu.Lock()
	m.setupHash, m.setupToken = sum[:], token
	m.mu.Unlock()
	log.Printf("Set up the owner at %s/setup#token=%s", cfg.URL, token)
	return nil
}

// Stop waits, until ctx ends, for the mailings register and forgot left
// running, then ends what is left of them. No mailing begins after it.
func (m *module) Stop(ctx context.Context) error {
	m.mu.Lock()
	cancel := m.cancelBase
	m.stopping = true // later adds no more to pending
	m.mu.Unlock()
	if cancel == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		m.pending.Wait()
		close(done)
	}()
	var err error
	select {
	case <-done:
	case <-ctx.Done():
		err = errors.New("the mail register or forgot asked for was still being made when the app stopped, so it was not sent; ask for it again.")
	}
	cancel()
	return err
}

// isEmail reports whether s passes the email rule.
func isEmail(s string) bool {
	return warp.Validate(struct {
		Email string `json:"email" warp:"email"`
	}{s}) == nil
}

// tokenIs reports, in constant time, whether token is the setup token.
func (m *module) tokenIs(token string) bool {
	sum := sha256.Sum256([]byte(token))
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.setupHash != nil && subtle.ConstantTimeCompare(sum[:], m.setupHash) == 1
}

// settingUp reports whether the app keeps a setup token: it started with no
// user, and none has been made since.
func (m *module) settingUp() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.setupHash != nil
}

// endSetup forgets the setup token, once a user exists.
func (m *module) endSetup() {
	m.mu.Lock()
	m.setupHash, m.setupToken = nil, ""
	m.mu.Unlock()
}

// of returns the auth module of app, or nil when it has none.
func of(app *warp.App) *module {
	for _, r := range app.Routes() {
		if r.Module == "auth" {
			return warp.Need[*module](app, "auth")
		}
	}
	return nil
}

// SetupToken returns the started app's setup token, or "" when it has none:
// once a user exists, before the app started, or without the auth module. It
// is meant for tests and tools, such as authtest.Owner, which cannot read the
// log; an app has no use for it.
func SetupToken(app *warp.App) string {
	m := of(app)
	if m == nil {
		return ""
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.setupToken
}

// AddUser makes a user with a role, one of owner, administrator, member and
// viewer. The email is kept lower-cased and trimmed, and the
// password is held to setup's rules, 8 to 200 characters; a mistake is a
// warp.Invalid on email, password or role, and an email another user has is
// a warp.Invalid on email. It is for seeding and tools, after the app has
// started.
func AddUser(ctx context.Context, app *warp.App, email, password, role string) (User, error) {
	m := of(app)
	if m == nil {
		return User{}, errors.New("auth: AddUser was given an app without the auth module; list auth.Module() in app/modules.go.")
	}
	email = normalEmail(email)
	if err := warp.Validate(struct {
		Email    string `json:"email" warp:"required,email"`
		Password string `json:"password" warp:"required,min=8,max=200"`
		Role     string `json:"role" warp:"required,enum=owner|administrator|member|viewer"`
	}{email, password, role}); err != nil {
		return User{}, err
	}
	hash, err := m.hash(ctx, password)
	if err != nil {
		return User{}, err
	}
	row := userRow{Email: email, Role: role, PasswordHash: hash}
	if err := m.users.Create(ctx, &row); err != nil {
		return User{}, err
	}
	m.endSetup()
	return row.user(), nil
}

// RemoveUser removes the user with the id and, in the same transaction,
// every session and API key of theirs and every invitation they sent. A user no one has is a warp.NotFound. It is for tools and tests, after the app has started, as
// AddUser is. It may remove the last user, the owner included: the running
// app then has no setup link, and the next start, finding no user, logs a
// new one.
func RemoveUser(ctx context.Context, app *warp.App, id int64) error {
	m := of(app)
	if m == nil {
		return errors.New("auth: RemoveUser was given an app without the auth module; list auth.Module() in app/modules.go.")
	}
	return m.removeUser(ctx, id)
}

// removeUser removes the user with the id and, in one transaction, every
// session, API key and mailed link of theirs and every invitation they
// sent: what RemoveUser and the route remove_user do. A reset link never
// outlives its user, so it cannot act on an account made later with the
// same address.
func (m *module) removeUser(ctx context.Context, id int64) error {
	return m.db.Tx(ctx, func(ctx context.Context) error {
		u, err := m.users.Get(ctx, id)
		if err != nil {
			return err
		}
		if _, err := m.tokens.DeleteBy(ctx, "email", u.Email); err != nil {
			return err
		}
		if _, err := m.sessions.DeleteBy(ctx, "user_id", id); err != nil {
			return err
		}
		if _, err := m.invitations.DeleteBy(ctx, "invited_by", id); err != nil {
			return err
		}
		if _, err := m.keys.DeleteBy(ctx, "user_id", id); err != nil {
			return err
		}
		return m.users.Delete(ctx, id)
	})
}

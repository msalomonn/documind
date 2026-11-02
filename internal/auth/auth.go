package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
	"net/mail"
	"strings"
	"sync"
	"time"
)

var ErrCredentials = errors.New("invalid_credentials")
var ErrInvalid = errors.New("invalid_credentials_format")
var ErrExists = errors.New("account_unavailable")
var ErrBusy = errors.New("auth_rate_limited")
var ErrSession = errors.New("authentication_required")

const Lifetime = 8 * time.Hour

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}
type Service struct {
	DB     *pgxpool.Pool
	slots  chan struct{}
	mu     sync.Mutex
	window time.Time
	count  int
}

func New(db *pgxpool.Pool) *Service { return &Service{DB: db, slots: make(chan struct{}, 2)} }
func Random() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func Digest(s string) string { d := sha256.Sum256([]byte(s)); return hex.EncodeToString(d[:]) }
func derive(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
}
func (s *Service) acquire() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.window) > time.Minute {
		s.window = time.Now()
		s.count = 0
	}
	if s.count >= 30 {
		return false
	}
	s.count++
	select {
	case s.slots <- struct{}{}:
		return true
	default:
		return false
	}
}
func (s *Service) Authenticate(ctx context.Context, email, password string, register bool) (User, string, error) {
	if !s.acquire() {
		return User{}, "", ErrBusy
	}
	defer func() { <-s.slots }()
	email = strings.ToLower(strings.TrimSpace(email))
	a, e := mail.ParseAddress(email)
	if e != nil || a.Address != email || len(email) > 254 || len(password) < 12 || len(password) > 128 {
		return User{}, "", ErrInvalid
	}
	user := User{Email: email}
	salt := make([]byte, 16)
	var stored []byte
	if register {
		if _, e = rand.Read(salt); e != nil {
			return User{}, "", e
		}
		user.ID = Random()
	} else {
		e = s.DB.QueryRow(ctx, "SELECT id,password_salt,password_hash FROM documind.users WHERE email=$1", email).Scan(&user.ID, &salt, &stored)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return User{}, "", e
		}
	}
	hash := derive(password, salt)
	if !register && (e != nil || subtle.ConstantTimeCompare(hash, stored) != 1) {
		return User{}, "", ErrCredentials
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return User{}, "", e
	}
	defer tx.Rollback(context.Background())
	if register {
		_, e = tx.Exec(ctx, "INSERT INTO documind.users(id,email,password_salt,password_hash) VALUES($1,$2,$3,$4)", user.ID, email, salt, hash)
		if e != nil {
			var pe *pgconn.PgError
			if errors.As(e, &pe) && pe.Code == "23505" {
				return User{}, "", ErrExists
			}
			return User{}, "", e
		}
	}
	token := Random()
	_, e = tx.Exec(ctx, "INSERT INTO documind.sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", Digest(token), user.ID, time.Now().Add(Lifetime))
	if e != nil {
		return User{}, "", e
	}
	if e = tx.Commit(ctx); e != nil {
		return User{}, "", e
	}
	return user, token, nil
}
func (s *Service) Resolve(ctx context.Context, token string) (User, error) {
	var u User
	if len(token) != 64 {
		return u, ErrSession
	}
	e := s.DB.QueryRow(ctx, `SELECT u.id,u.email FROM documind.sessions s JOIN documind.users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now()`, Digest(token)).Scan(&u.ID, &u.Email)
	if errors.Is(e, pgx.ErrNoRows) {
		e = ErrSession
	}
	return u, e
}
func (s *Service) Logout(ctx context.Context, token string) error {
	_, e := s.DB.Exec(ctx, "DELETE FROM documind.sessions WHERE token_hash=$1", Digest(token))
	return e
}

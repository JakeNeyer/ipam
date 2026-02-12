package setup

import (
	"context"
	"os"
	"strings"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/server/validation"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
)

// NewStore creates a store from the environment (Postgres if DATABASE_URL is set, otherwise in-memory).
// The returned close function should be called when the program exits (no-op for in-memory).
func NewStore(ctx context.Context) (st store.Storer, close func(), err error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		logger.Info("store", slog.String("type", "in_memory"))
		return store.NewStore(), func() {}, nil
	}
	st, err = store.NewPostgresStore(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}
	logger.Info("store", slog.String("type", "postgres"))
	closeFn := func() {}
	if pg, ok := st.(*store.PostgresStore); ok {
		closeFn = func() { pg.Close() }
	}
	return st, closeFn, nil
}

// EnsureInitialAdmin creates the first admin when INITIAL_ADMIN_EMAIL is set and no users exist.
// When oauthEnabled is true, INITIAL_ADMIN_PASSWORD is optional; otherwise both email and password are required.
// Also accepts INTIAL_ADMIN_EMAIL (missing N) as a common typo.
func EnsureInitialAdmin(st store.Storer, oauthEnabled bool) {
	email := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_EMAIL"))
	if email == "" {
		email = strings.TrimSpace(os.Getenv("INTIAL_ADMIN_EMAIL"))
	}
	password := os.Getenv("INITIAL_ADMIN_PASSWORD")
	if email == "" {
		return
	}
	if !oauthEnabled && password == "" {
		return
	}
	users, err := st.ListUsers(nil)
	if err != nil || len(users) > 0 {
		return
	}
	if !validation.ValidateEmail(email) {
		logger.Info("initial admin skipped: invalid email")
		return
	}
	var passwordHash string
	if password != "" {
		if !validation.ValidatePassword(password) {
			logger.Info("initial admin skipped: invalid password (8–72 chars)")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			logger.Error("initial admin password hash failed", logger.ErrAttr(err))
			return
		}
		passwordHash = string(hash)
	}
	admin := &store.User{
		Email:          strings.TrimSpace(strings.ToLower(email)),
		PasswordHash:   passwordHash,
		Role:           store.RoleAdmin,
		OrganizationID: uuid.Nil,
	}
	if err := st.CreateUser(admin); err != nil {
		logger.Error("initial admin create failed", logger.ErrAttr(err))
		return
	}
	logger.Info("initial admin created", slog.String("email", admin.Email))
}

package setup

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/server/validation"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	initialAdminAPITokenEnv    = "INITIAL_ADMIN_API_TOKEN"
	initialAdminAPITokenTTLEnv = "INITIAL_ADMIN_API_TOKEN_TTL"
	initialAdminAPITokenName   = "initial_admin"
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
		closeFn = func() { _ = pg.Close() }
	}
	return st, closeFn, nil
}

// EnsureInitialAdmin creates the first admin when INITIAL_ADMIN_EMAIL is set and no users exist.
// When oauthEnabled is true, INITIAL_ADMIN_PASSWORD is optional; otherwise both email and password are required.
// Also accepts INTIAL_ADMIN_EMAIL (missing N) as a common typo.
//
// EnsureInitialAdmin only creates an API token for initial admin if INITIAL_ADMIN_API_TOKEN is set.
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
	if err := ensureInitialAdminAPIToken(st, admin); err != nil {
		logger.Error("initial admin API token create failed", logger.ErrAttr(err))
	}
	logger.Info("initial admin API token created")
}

// initialAdminAPITokenExpiration validates that INITIAL_ADMIN_API_TOKEN_TTL is set and is a valid time duration
func initialAdminAPITokenExpiration() (*time.Time, error) {
	raw := strings.TrimSpace(os.Getenv(initialAdminAPITokenTTLEnv))
	if raw == "" {
		return nil, fmt.Errorf("%s must be set when configuring initial admin API token", initialAdminAPITokenTTLEnv)
	}

	ttl, err := parseAPITokenTTL(raw)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid %s: %w",
			initialAdminAPITokenTTLEnv,
			err,
		)
	}

	if ttl <= 0 {
		return nil, fmt.Errorf("%s must be greater than zero", initialAdminAPITokenTTLEnv)
	}

	expiration := time.Now().UTC().Add(ttl)
	return &expiration, nil
}

// ensureInitialAdminAPIToken creates the initial admin api token from INITIAL_ADMIN_API_TOKEN
func ensureInitialAdminAPIToken(st store.Storer, admin *store.User) error {
	rawToken := strings.TrimSpace(os.Getenv(initialAdminAPITokenEnv))
	if rawToken == "" {
		return nil
	}

	expiresAt, err := initialAdminAPITokenExpiration()
	if err != nil {
		return err
	}

	_, err = st.CreateAPITokenWithRaw(
		admin.ID,
		initialAdminAPITokenName,
		rawToken,
		expiresAt,
		nil,
	)
	if err != nil {
		return fmt.Errorf("create initial admin API token: %w", err)
	}

	return nil
}

// parseAPITokenTTL is simple helper to allow more flexible duration times instead of just hours
func parseAPITokenTTL(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}

	if d, err := time.ParseDuration(raw); err == nil {
		return d, nil
	}

	var multiplier time.Duration
	var value string

	switch {
	case strings.HasSuffix(raw, "d"):
		multiplier = 24 * time.Hour
		value = strings.TrimSuffix(raw, "d")

	case strings.HasSuffix(raw, "w"):
		multiplier = 7 * 24 * time.Hour
		value = strings.TrimSuffix(raw, "w")

	case strings.HasSuffix(raw, "y"):
		multiplier = 365 * 24 * time.Hour
		value = strings.TrimSuffix(raw, "y")

	default:
		return 0, fmt.Errorf("invalid duration %q", raw)
	}

	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid duration %q", raw)
	}

	return time.Duration(n) * multiplier, nil
}

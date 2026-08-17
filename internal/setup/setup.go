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
	initialOrganizationNameEnv = "INITIAL_ORGANIZATION_NAME"
	initialAdminAPITokenName   = "initial_admin"
	minBootstrapTokenLen       = 16
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
func EnsureInitialAdmin(st store.Storer, oauthEnabled bool) error {
	email := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_EMAIL"))
	password := os.Getenv("INITIAL_ADMIN_PASSWORD")
	orgName := strings.TrimSpace(os.Getenv(initialOrganizationNameEnv))
	rawToken := strings.TrimSpace(os.Getenv(initialAdminAPITokenEnv))

	wantOrg := orgName != ""
	wantToken := rawToken != ""
	strict := wantOrg || wantToken

	if email == "" {
		if strict {
			return fmt.Errorf("INITIAL_ADMIN_EMAIL must be set when configuring %s or %s", initialOrganizationNameEnv, initialAdminAPITokenEnv)
		}
		return nil
	}
	if !oauthEnabled && password == "" {
		if strict {
			return fmt.Errorf("INITIAL_ADMIN_PASSWORD must be set when configuring %s or %s (or enable OAuth)", initialOrganizationNameEnv, initialAdminAPITokenEnv)
		}
		return nil
	}

	users, err := st.ListUsers(nil)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	if len(users) > 0 {
		return nil
	}

	if !validation.ValidateEmail(email) {
		if strict {
			return fmt.Errorf("initial admin email is invalid")
		}
		logger.Info("initial admin skipped: invalid email")
		return nil
	}

	var passwordHash string
	if password != "" {
		if !validation.ValidatePassword(password) {
			if strict {
				return fmt.Errorf("initial admin password is invalid (8–72 chars)")
			}
			logger.Info("initial admin skipped: invalid password (8–72 chars)")
			return nil
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("initial admin password hash failed: %w", err)
		}
		passwordHash = string(hash)
	}

	var expiresAt *time.Time
	if wantToken {
		if !wantOrg {
			return fmt.Errorf("%s must be set when configuring %s", initialOrganizationNameEnv, initialAdminAPITokenEnv)
		}
		if err := validateBootstrapToken(rawToken); err != nil {
			return err
		}
		expiresAt, err = initialAdminAPITokenExpiration()
		if err != nil {
			return err
		}
	}

	admin := &store.User{
		Email:          strings.TrimSpace(strings.ToLower(email)),
		PasswordHash:   passwordHash,
		Role:           store.RoleAdmin,
		OrganizationID: uuid.Nil,
	}
	if err := st.CreateUser(admin); err != nil {
		return fmt.Errorf("initial admin create failed: %w", err)
	}
	logger.Info("initial admin created", slog.String("email", admin.Email))

	if !wantOrg {
		return nil
	}

	org := &store.Organization{Name: orgName}
	if err := st.CreateOrganization(org); err != nil {
		return fmt.Errorf("initial organization create failed: %w", err)
	}
	logger.Info("initial organization created", slog.String("name", org.Name), slog.String("org_id", org.ID.String()))

	if !wantToken {
		return nil
	}

	if _, err := st.CreateAPITokenWithRaw(admin.ID, initialAdminAPITokenName, rawToken, expiresAt, &org.ID); err != nil {
		return fmt.Errorf("initial admin API token create failed: %w", err)
	}
	logger.Info("initial admin API token created", slog.String("name", initialAdminAPITokenName), slog.String("org_id", org.ID.String()))
	return nil
}

func validateBootstrapToken(rawToken string) error {
	if len(rawToken) < minBootstrapTokenLen {
		return fmt.Errorf("%s must be at least %d characters", initialAdminAPITokenEnv, minBootstrapTokenLen)
	}
	return nil
}

func initialAdminAPITokenExpiration() (*time.Time, error) {
	raw := strings.TrimSpace(os.Getenv(initialAdminAPITokenTTLEnv))
	if raw == "" {
		return nil, fmt.Errorf("%s must be set when configuring %s", initialAdminAPITokenTTLEnv, initialAdminAPITokenEnv)
	}
	ttl, err := parseAPITokenTTL(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", initialAdminAPITokenTTLEnv, err)
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("%s must be greater than zero", initialAdminAPITokenTTLEnv)
	}
	expiration := time.Now().UTC().Add(ttl)
	return &expiration, nil
}

// parseAPITokenTTL accepts Go durations (e.g. 30m, 12h) plus day/week/year suffixes (e.g. 60d, 8w, 1y).
func parseAPITokenTTL(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty duration")
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

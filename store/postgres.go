package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/network"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// nullUUID supports scanning NULL UUID from PostgreSQL.
type nullUUID struct {
	UUID  uuid.UUID
	Valid bool
}

func (n *nullUUID) Scan(value interface{}) error {
	if value == nil {
		n.UUID, n.Valid = uuid.Nil, false
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 16 {
			u, err := uuid.FromBytes(v)
			if err != nil {
				return err
			}
			n.UUID, n.Valid = u, true
			return nil
		}
	case string:
		u, err := uuid.Parse(v)
		if err != nil {
			return err
		}
		n.UUID, n.Valid = u, true
		return nil
	}
	return fmt.Errorf("cannot scan %T into nullUUID", value)
}

// PostgresStore implements Storer using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore connects to PostgreSQL, runs migrations, and returns a Storer.
func NewPostgresStore(ctx context.Context, dsn string) (Storer, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if err := runMigrations(dsn); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &PostgresStore{db: db}, nil
}

// Close closes the database connection. Call this when shutting down.
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// runMigrations runs SQL migrations from the embedded migrations FS using golang-migrate.
// The migrate postgres driver expects a postgres:// URL (lib/pq format).
func runMigrations(dsn string) error {
	// Normalize scheme for migrate's postgres driver (accepts postgres:// or postgresql://)
	migrateURL := dsn
	if strings.HasPrefix(dsn, "postgresql://") {
		migrateURL = "postgres://" + strings.TrimPrefix(dsn, "postgresql://")
	}
	// lib/pq defaults to SSL; if the server has no SSL, add sslmode=disable when not set
	if !strings.Contains(migrateURL, "sslmode=") {
		if strings.Contains(migrateURL, "?") {
			migrateURL += "&sslmode=disable"
		} else {
			migrateURL += "?sslmode=disable"
		}
	}
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, migrateURL)
	if err != nil {
		return fmt.Errorf("migrate new: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func (s *PostgresStore) GenerateID() uuid.UUID {
	return uuid.New()
}

// uuidPtr returns a pointer for INSERT/UPDATE; use NULL for uuid.Nil.
func uuidPtr(u uuid.UUID) interface{} {
	if u == uuid.Nil {
		return nil
	}
	return u
}

func (s *PostgresStore) CreateEnvironment(env *network.Environment) error {
	_, err := s.db.Exec(
		`INSERT INTO environments (id, name) VALUES ($1, $2)`,
		env.Id, env.Name,
	)
	return err
}

func (s *PostgresStore) GetEnvironment(id uuid.UUID) (*network.Environment, error) {
	var name string
	err := s.db.QueryRow(`SELECT id, name FROM environments WHERE id = $1`, id).Scan(&id, &name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment not found")
	}
	if err != nil {
		return nil, err
	}
	return &network.Environment{Id: id, Name: name, Block: []network.Block{}}, nil
}

func (s *PostgresStore) ListEnvironments() ([]*network.Environment, error) {
	out, _, err := s.ListEnvironmentsFiltered("", 0, 0)
	return out, err
}

func (s *PostgresStore) ListEnvironmentsFiltered(name string, limit, offset int) ([]*network.Environment, int, error) {
	name = strings.TrimSpace(name)
	nameLower := strings.ToLower(name)
	var total int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM environments WHERE $1 = '' OR LOWER(name) LIKE '%' || $1 || '%'`,
		nameLower,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	q := `SELECT id, name FROM environments WHERE $1 = '' OR LOWER(name) LIKE '%' || $1 || '%' ORDER BY name`
	args := []interface{}{nameLower}
	if limit > 0 {
		q += ` LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*network.Environment
	for rows.Next() {
		var id uuid.UUID
		var n string
		if err := rows.Scan(&id, &n); err != nil {
			return nil, 0, err
		}
		out = append(out, &network.Environment{Id: id, Name: n, Block: []network.Block{}})
	}
	return out, total, rows.Err()
}

func (s *PostgresStore) UpdateEnvironment(id uuid.UUID, env *network.Environment) error {
	res, err := s.db.Exec(`UPDATE environments SET name = $1 WHERE id = $2`, env.Name, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("environment not found")
	}
	return nil
}

func (s *PostgresStore) DeleteEnvironment(id uuid.UUID) error {
	// Delete allocations in blocks that belong to this env, then blocks, then env
	_, err := s.db.Exec(
		`DELETE FROM allocations WHERE block_name IN (SELECT name FROM blocks WHERE environment_id = $1)`,
		id,
	)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM blocks WHERE environment_id = $1`, id)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM environments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("environment not found")
	}
	return nil
}

func (s *PostgresStore) CreateBlock(block *network.Block) error {
	if block.ID == uuid.Nil {
		block.ID = s.GenerateID()
	}
	total := block.Usage.TotalIPs
	if total == 0 {
		total = 1 << 20 // placeholder if not set
	}
	_, err := s.db.Exec(
		`INSERT INTO blocks (id, name, cidr, environment_id, total_ips) VALUES ($1, $2, $3, $4, $5)`,
		block.ID, block.Name, block.CIDR, uuidPtr(block.EnvironmentID), total,
	)
	return err
}

func (s *PostgresStore) GetBlock(id uuid.UUID) (*network.Block, error) {
	var name, cidr string
	var envID nullUUID
	var totalIPs int64
	err := s.db.QueryRow(
		`SELECT id, name, cidr, environment_id, total_ips FROM blocks WHERE id = $1`,
		id,
	).Scan(&id, &name, &cidr, &envID, &totalIPs)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("block not found")
	}
	if err != nil {
		return nil, err
	}
	envUUID := uuid.Nil
	if envID.Valid {
		envUUID = envID.UUID
	}
	t := int(totalIPs)
	return &network.Block{
		ID:            id,
		Name:          name,
		CIDR:          cidr,
		EnvironmentID: envUUID,
		Usage:         network.Usage{TotalIPs: t, UsedIPs: 0, AvailableIPs: t},
		Children:      []network.Block{},
	}, nil
}

func (s *PostgresStore) ListBlocks() ([]*network.Block, error) {
	out, _, err := s.ListBlocksFiltered("", nil, false, 0, 0)
	return out, err
}

func (s *PostgresStore) ListBlocksFiltered(name string, environmentID *uuid.UUID, orphanedOnly bool, limit, offset int) ([]*network.Block, int, error) {
	// Count
	var countArgs []interface{}
	countQ := `SELECT COUNT(*) FROM blocks WHERE 1=1`
	idx := 1
	if name != "" {
		countQ += fmt.Sprintf(` AND LOWER(name) LIKE $%d`, idx)
		countArgs = append(countArgs, "%"+strings.ToLower(strings.TrimSpace(name))+"%")
		idx++
	}
	if environmentID != nil {
		countQ += fmt.Sprintf(` AND environment_id = $%d`, idx)
		countArgs = append(countArgs, *environmentID)
		idx++
	}
	if orphanedOnly {
		countQ += ` AND environment_id IS NULL`
	}
	var total int
	if err := s.db.QueryRow(countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	// Select
	selQ := `SELECT id, name, cidr, environment_id, total_ips FROM blocks WHERE 1=1`
	selArgs := []interface{}{}
	i := 1
	if name != "" {
		selQ += fmt.Sprintf(` AND LOWER(name) LIKE $%d`, i)
		selArgs = append(selArgs, "%"+strings.ToLower(strings.TrimSpace(name))+"%")
		i++
	}
	if environmentID != nil {
		selQ += fmt.Sprintf(` AND environment_id = $%d`, i)
		selArgs = append(selArgs, *environmentID)
		i++
	}
	if orphanedOnly {
		selQ += ` AND environment_id IS NULL`
	}
	selQ += ` ORDER BY name`
	if limit > 0 {
		selQ += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, i, i+1)
		selArgs = append(selArgs, limit, offset)
	}
	rows, err := s.db.Query(selQ, selArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*network.Block
	for rows.Next() {
		var id uuid.UUID
		var n, cidr string
		var envID nullUUID
		var totalIPs int64
		if err := rows.Scan(&id, &n, &cidr, &envID, &totalIPs); err != nil {
			return nil, 0, err
		}
		envUUID := uuid.Nil
		if envID.Valid {
			envUUID = envID.UUID
		}
		t := int(totalIPs)
		out = append(out, &network.Block{
			ID:            id,
			Name:          n,
			CIDR:          cidr,
			EnvironmentID: envUUID,
			Usage:         network.Usage{TotalIPs: t, UsedIPs: 0, AvailableIPs: t},
			Children:      []network.Block{},
		})
	}
	return out, total, rows.Err()
}

func (s *PostgresStore) ListBlocksByEnvironment(envID uuid.UUID) ([]*network.Block, error) {
	blocks, _, err := s.ListBlocksFiltered("", &envID, false, 0, 0)
	return blocks, err
}

func (s *PostgresStore) UpdateBlock(id uuid.UUID, block *network.Block) error {
	total := block.Usage.TotalIPs
	if total == 0 {
		total = 1 << 20
	}
	res, err := s.db.Exec(
		`UPDATE blocks SET name = $1, cidr = $2, environment_id = $3, total_ips = $4 WHERE id = $5`,
		block.Name, block.CIDR, uuidPtr(block.EnvironmentID), total, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("block not found")
	}
	return nil
}

func (s *PostgresStore) DeleteBlock(id uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM blocks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("block not found")
	}
	// Allocations are keyed by block_name; we don't cascade by block id. Leave orphaned allocations?
	// In-memory store deletes block only. So we only delete the block row.
	return nil
}

func (s *PostgresStore) CreateAllocation(id uuid.UUID, alloc *network.Allocation) error {
	_, err := s.db.Exec(
		`INSERT INTO allocations (id, name, block_name, block_cidr) VALUES ($1, $2, $3, $4)`,
		id, alloc.Name, alloc.Block.Name, alloc.Block.CIDR,
	)
	return err
}

func (s *PostgresStore) GetAllocation(id uuid.UUID) (*network.Allocation, error) {
	var name, blockName, blockCidr string
	err := s.db.QueryRow(
		`SELECT id, name, block_name, block_cidr FROM allocations WHERE id = $1`,
		id,
	).Scan(&id, &name, &blockName, &blockCidr)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("allocation not found")
	}
	if err != nil {
		return nil, err
	}
	return &network.Allocation{
		Id:    id,
		Name:  name,
		Block: network.Block{Name: blockName, CIDR: blockCidr},
	}, nil
}

func (s *PostgresStore) ListAllocations() ([]*network.Allocation, error) {
	out, _, err := s.ListAllocationsFiltered("", "", uuid.Nil, 0, 0)
	return out, err
}

func (s *PostgresStore) ListAllocationsFiltered(name string, blockName string, environmentID uuid.UUID, limit, offset int) ([]*network.Allocation, int, error) {
	name = strings.TrimSpace(name)
	blockName = strings.TrimSpace(blockName)
	nameLower := strings.ToLower(name)
	blockLower := strings.ToLower(blockName)
	envFilter := environmentID != uuid.Nil
	var total int
	countQ := `SELECT COUNT(*) FROM allocations WHERE ($1 = '' OR LOWER(name) LIKE '%' || $1 || '%') AND ($2 = '' OR LOWER(block_name) = $2)`
	args := []interface{}{nameLower, blockLower}
	if envFilter {
		countQ += ` AND LOWER(block_name) IN (SELECT LOWER(name) FROM blocks WHERE environment_id = $3)`
		args = append(args, environmentID)
	}
	if err := s.db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	selQ := `SELECT id, name, block_name, block_cidr FROM allocations WHERE ($1 = '' OR LOWER(name) LIKE '%' || $1 || '%') AND ($2 = '' OR LOWER(block_name) = $2)`
	selArgs := []interface{}{nameLower, blockLower}
	if envFilter {
		selQ += ` AND LOWER(block_name) IN (SELECT LOWER(name) FROM blocks WHERE environment_id = $3)`
		selArgs = append(selArgs, environmentID)
	}
	selQ += ` ORDER BY name`
	if limit > 0 {
		n := len(selArgs) + 1
		selQ += ` LIMIT $` + strconv.Itoa(n) + ` OFFSET $` + strconv.Itoa(n+1)
		selArgs = append(selArgs, limit, offset)
	}
	rows, err := s.db.Query(selQ, selArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*network.Allocation
	for rows.Next() {
		var id uuid.UUID
		var n, bn, bc string
		if err := rows.Scan(&id, &n, &bn, &bc); err != nil {
			return nil, 0, err
		}
		out = append(out, &network.Allocation{
			Id:    id,
			Name:  n,
			Block: network.Block{Name: bn, CIDR: bc},
		})
	}
	return out, total, rows.Err()
}

func (s *PostgresStore) UpdateAllocation(id uuid.UUID, alloc *network.Allocation) error {
	res, err := s.db.Exec(
		`UPDATE allocations SET name = $1, block_name = $2, block_cidr = $3 WHERE id = $4`,
		alloc.Name, alloc.Block.Name, alloc.Block.CIDR, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("allocation not found")
	}
	return nil
}

func (s *PostgresStore) DeleteAllocation(id uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM allocations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("allocation not found")
	}
	return nil
}

func (s *PostgresStore) ListReservedBlocks() ([]*ReservedBlock, error) {
	rows, err := s.db.Query(`SELECT id, name, cidr, reason, created_at FROM reserved_blocks ORDER BY name, cidr`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ReservedBlock
	for rows.Next() {
		var r ReservedBlock
		if err := rows.Scan(&r.ID, &r.Name, &r.CIDR, &r.Reason, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (s *PostgresStore) CreateReservedBlock(r *ReservedBlock) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(
		`INSERT INTO reserved_blocks (id, name, cidr, reason, created_at) VALUES ($1, $2, $3, $4, $5)`,
		r.ID, strings.TrimSpace(r.Name), r.CIDR, r.Reason, r.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetReservedBlock(id uuid.UUID) (*ReservedBlock, error) {
	var r ReservedBlock
	err := s.db.QueryRow(`SELECT id, name, cidr, reason, created_at FROM reserved_blocks WHERE id = $1`, id).Scan(&r.ID, &r.Name, &r.CIDR, &r.Reason, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("reserved block not found")
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *PostgresStore) DeleteReservedBlock(id uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM reserved_blocks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("reserved block not found")
	}
	return nil
}

func (s *PostgresStore) OverlapsReservedBlock(cidr string) (*ReservedBlock, error) {
	list, err := s.ListReservedBlocks()
	if err != nil {
		return nil, err
	}
	for _, r := range list {
		overlap, err := network.Overlaps(cidr, r.CIDR)
		if err != nil {
			return nil, err
		}
		if overlap {
			return r, nil
		}
	}
	return nil, nil
}

func (s *PostgresStore) CreateUser(u *User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	email := strings.TrimSpace(u.Email)
	_, err := s.db.Exec(
		`INSERT INTO users (id, email, password_hash, role, tour_completed) VALUES ($1, $2, $3, $4, $5)`,
		u.ID.String(), email, u.PasswordHash, u.Role, u.TourCompleted,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return fmt.Errorf("user with email already exists")
		}
		// Return a plain error so handlers map to 4xx, not raw driver error (which can become 500)
		return fmt.Errorf("failed to create user")
	}
	return nil
}

func (s *PostgresStore) GetUser(id uuid.UUID) (*User, error) {
	var u User
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, role, tour_completed FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TourCompleted)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStore) GetUserByEmail(email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var u User
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, role, tour_completed FROM users WHERE LOWER(email) = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TourCompleted)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStore) ListUsers() ([]*User, error) {
	rows, err := s.db.Query(`SELECT id, email, password_hash, role, tour_completed FROM users ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TourCompleted); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

func (s *PostgresStore) DeleteUser(userID uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *PostgresStore) SetUserRole(userID uuid.UUID, role string) error {
	if role != RoleUser && role != RoleAdmin {
		return fmt.Errorf("invalid role")
	}
	res, err := s.db.Exec(`UPDATE users SET role = $1 WHERE id = $2`, role, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *PostgresStore) SetUserTourCompleted(userID uuid.UUID, completed bool) error {
	res, err := s.db.Exec(`UPDATE users SET tour_completed = $1 WHERE id = $2`, completed, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *PostgresStore) CreateSession(sessionID string, userID uuid.UUID, expiry time.Time) {
	_, _ = s.db.Exec(`INSERT INTO sessions (session_id, user_id, expiry) VALUES ($1, $2, $3) ON CONFLICT (session_id) DO UPDATE SET user_id = $2, expiry = $3`, sessionID, userID, expiry)
}

func (s *PostgresStore) GetSession(sessionID string) (*Session, error) {
	var userID uuid.UUID
	var expiry time.Time
	err := s.db.QueryRow(`SELECT user_id, expiry FROM sessions WHERE session_id = $1 AND expiry > NOW()`, sessionID).Scan(&userID, &expiry)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found or expired")
	}
	if err != nil {
		return nil, err
	}
	return &Session{UserID: userID, Expiry: expiry}, nil
}

func (s *PostgresStore) DeleteSession(sessionID string) {
	_, _ = s.db.Exec(`DELETE FROM sessions WHERE session_id = $1`, sessionID)
}

func (s *PostgresStore) CreateAPIToken(userID uuid.UUID, name string, expiresAt *time.Time) (token *APIToken, rawToken string, err error) {
	var n int
	if err = s.db.QueryRow(`SELECT 1 FROM users WHERE id = $1`, userID).Scan(&n); err == sql.ErrNoRows {
		return nil, "", fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, "", err
	}
	secret := make([]byte, apiTokenSecretBytes)
	if _, err := rand.Read(secret); err != nil {
		return nil, "", err
	}
	rawToken = apiTokenPrefix + hex.EncodeToString(secret)
	keyHash := hashToken(rawToken)
	id := uuid.New()
	token = &APIToken{
		ID:        id,
		UserID:    userID,
		Name:      strings.TrimSpace(name),
		KeyHash:   keyHash,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
	_, err = s.db.Exec(
		`INSERT INTO api_tokens (id, user_id, name, key_hash, created_at, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		token.ID, token.UserID, token.Name, token.KeyHash, token.CreatedAt, token.ExpiresAt,
	)
	if err != nil {
		return nil, "", err
	}
	return token, rawToken, nil
}

func (s *PostgresStore) GetUserByTokenHash(keyHash string) (*User, error) {
	var tokenID, userID uuid.UUID
	var expiresAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, user_id, expires_at FROM api_tokens WHERE key_hash = $1`,
		keyHash,
	).Scan(&tokenID, &userID, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		return nil, fmt.Errorf("token expired")
	}
	return s.GetUser(userID)
}

func (s *PostgresStore) ListAPITokens(userID uuid.UUID) ([]*APIToken, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, name, key_hash, created_at, expires_at FROM api_tokens WHERE user_id = $1 ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*APIToken
	for rows.Next() {
		var t APIToken
		var expiresAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.KeyHash, &t.CreatedAt, &expiresAt); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			t.ExpiresAt = &expiresAt.Time
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (s *PostgresStore) DeleteAPIToken(tokenID, userID uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`, tokenID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("token not found")
	}
	return nil
}

func (s *PostgresStore) GetAPIToken(tokenID uuid.UUID) (*APIToken, error) {
	var t APIToken
	var expiresAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, user_id, name, key_hash, created_at, expires_at FROM api_tokens WHERE id = $1`,
		tokenID,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.KeyHash, &t.CreatedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		t.ExpiresAt = &expiresAt.Time
	}
	return &t, nil
}

func (s *PostgresStore) CreateSignupInvite(createdBy uuid.UUID, expiresAt time.Time) (*SignupInvite, string, error) {
	var n int
	err := s.db.QueryRow(`SELECT 1 FROM users WHERE id = $1`, createdBy).Scan(&n)
	if err == sql.ErrNoRows {
		return nil, "", fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, "", err
	}
	secret := make([]byte, signupInviteSecretBytes)
	if _, err := rand.Read(secret); err != nil {
		return nil, "", err
	}
	rawToken := signupInviteTokenPrefix + hex.EncodeToString(secret)
	tokenHash := hashToken(rawToken)
	now := time.Now()
	if expiresAt.Before(now) {
		return nil, "", fmt.Errorf("expires_at must be in the future")
	}
	inv := &SignupInvite{
		ID:        uuid.New(),
		TokenHash: tokenHash,
		CreatedBy: createdBy,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	_, err = s.db.Exec(
		`INSERT INTO signup_invites (id, token_hash, created_by, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)`,
		inv.ID, inv.TokenHash, inv.CreatedBy, inv.ExpiresAt, inv.CreatedAt,
	)
	if err != nil {
		return nil, "", err
	}
	return inv, rawToken, nil
}

func (s *PostgresStore) GetSignupInviteByToken(rawToken string) (*SignupInvite, error) {
	if rawToken == "" || !strings.HasPrefix(rawToken, signupInviteTokenPrefix) {
		return nil, fmt.Errorf("invalid token")
	}
	tokenHash := hashToken(rawToken)
	var inv SignupInvite
	var usedAt sql.NullTime
	var usedByUserID nullUUID
	err := s.db.QueryRow(
		`SELECT id, token_hash, created_by, expires_at, created_at, used_at, used_by_user_id FROM signup_invites WHERE token_hash = $1`,
		tokenHash,
	).Scan(&inv.ID, &inv.TokenHash, &inv.CreatedBy, &inv.ExpiresAt, &inv.CreatedAt, &usedAt, &usedByUserID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invite not found")
	}
	if err != nil {
		return nil, err
	}
	if usedAt.Valid {
		return nil, fmt.Errorf("invite already used")
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, fmt.Errorf("invite expired")
	}
	return &inv, nil
}

func (s *PostgresStore) MarkSignupInviteUsed(inviteID, userID uuid.UUID) error {
	res, err := s.db.Exec(
		`UPDATE signup_invites SET used_at = NOW(), used_by_user_id = $1 WHERE id = $2`,
		userID, inviteID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("invite not found")
	}
	return nil
}

func (s *PostgresStore) DeleteSignupInvite(id uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM signup_invites WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("invite not found")
	}
	return nil
}

func (s *PostgresStore) ListSignupInvites(createdBy uuid.UUID) ([]*SignupInvite, error) {
	rows, err := s.db.Query(
		`SELECT id, token_hash, created_by, expires_at, created_at, used_at, used_by_user_id FROM signup_invites WHERE created_by = $1 ORDER BY created_at DESC`,
		createdBy,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*SignupInvite
	for rows.Next() {
		var inv SignupInvite
		var usedAt sql.NullTime
		var usedByUserID nullUUID
		if err := rows.Scan(&inv.ID, &inv.TokenHash, &inv.CreatedBy, &inv.ExpiresAt, &inv.CreatedAt, &usedAt, &usedByUserID); err != nil {
			return nil, err
		}
		if usedAt.Valid {
			inv.UsedAt = &usedAt.Time
		}
		if usedByUserID.Valid {
			inv.UsedByUserID = &usedByUserID.UUID
		}
		out = append(out, &inv)
	}
	return out, rows.Err()
}

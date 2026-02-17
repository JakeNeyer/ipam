package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"fmt"
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
	migrateURL := dsn
	if strings.HasPrefix(dsn, "postgresql://") {
		migrateURL = "postgres://" + strings.TrimPrefix(dsn, "postgresql://")
	}
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

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (s *PostgresStore) CreateOrganization(org *Organization) error {
	if org.ID == uuid.Nil {
		org.ID = s.GenerateID()
	}
	if org.CreatedAt.IsZero() {
		org.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(
		`INSERT INTO organizations (id, name, created_at) VALUES ($1, $2, $3)`,
		org.ID, org.Name, org.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetOrganization(id uuid.UUID) (*Organization, error) {
	var o Organization
	err := s.db.QueryRow(`SELECT id, name, created_at FROM organizations WHERE id = $1`, id).Scan(&o.ID, &o.Name, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *PostgresStore) ListOrganizations() ([]*Organization, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM organizations ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &o)
	}
	return out, rows.Err()
}

func (s *PostgresStore) UpdateOrganization(org *Organization) error {
	res, err := s.db.Exec(`UPDATE organizations SET name = $1 WHERE id = $2`, org.Name, org.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("organization not found")
	}
	return nil
}

func (s *PostgresStore) DeleteOrganization(id uuid.UUID) error {
	if _, err := s.GetOrganization(id); err != nil {
		return err
	}
	// Cascade: allocations in blocks (envs in org + org-scoped orphan blocks) → blocks → environments → ...
	_, err := s.db.Exec(
		`DELETE FROM allocations WHERE LOWER(block_name) IN (
			SELECT LOWER(name) FROM blocks WHERE
				environment_id IN (SELECT id FROM environments WHERE organization_id = $1)
				OR (environment_id IS NULL AND organization_id = $1)
		)`,
		id,
	)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`DELETE FROM blocks WHERE environment_id IN (SELECT id FROM environments WHERE organization_id = $1) OR (environment_id IS NULL AND organization_id = $1)`,
		id,
	)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM environments WHERE organization_id = $1`, id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM reserved_blocks WHERE organization_id = $1`, id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM signup_invites WHERE organization_id = $1`, id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM cloud_connections WHERE organization_id = $1`, id)
	if err != nil {
		return err
	}
	// Sessions and api_tokens are CASCADE when users are deleted
	_, err = s.db.Exec(`DELETE FROM users WHERE organization_id = $1`, id)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("organization not found")
	}
	return nil
}

func (s *PostgresStore) CreateEnvironment(env *network.Environment) error {
	_, err := s.db.Exec(
		`INSERT INTO environments (id, name, organization_id) VALUES ($1, $2, $3)`,
		env.Id, env.Name, uuidPtr(env.OrganizationID),
	)
	return err
}

func (s *PostgresStore) GetEnvironment(id uuid.UUID) (*network.Environment, error) {
	var name string
	var orgID nullUUID
	err := s.db.QueryRow(`SELECT id, name, organization_id FROM environments WHERE id = $1`, id).Scan(&id, &name, &orgID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment not found")
	}
	if err != nil {
		return nil, err
	}
	orgUUID := uuid.Nil
	if orgID.Valid {
		orgUUID = orgID.UUID
	}
	return &network.Environment{Id: id, Name: name, OrganizationID: orgUUID, Block: []network.Block{}}, nil
}

func (s *PostgresStore) ListEnvironments() ([]*network.Environment, error) {
	out, _, err := s.ListEnvironmentsFiltered("", nil, 0, 0)
	return out, err
}

func (s *PostgresStore) ListEnvironmentsFiltered(name string, organizationID *uuid.UUID, limit, offset int) ([]*network.Environment, int, error) {
	name = strings.TrimSpace(name)
	nameLower := strings.ToLower(name)
	countQ := `SELECT COUNT(*) FROM environments WHERE ($1 = '' OR LOWER(name) LIKE '%' || $1 || '%')`
	countArgs := []interface{}{nameLower}
	if organizationID != nil {
		countQ += ` AND organization_id = $2`
		countArgs = append(countArgs, *organizationID)
	}
	var total int
	if err := s.db.QueryRow(countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	selQ := `SELECT id, name, organization_id FROM environments WHERE ($1 = '' OR LOWER(name) LIKE '%' || $1 || '%')`
	selArgs := []interface{}{nameLower}
	if organizationID != nil {
		selQ += ` AND organization_id = $2`
		selArgs = append(selArgs, *organizationID)
	}
	selQ += ` ORDER BY name`
	argIdx := len(selArgs) + 1
	if limit > 0 {
		// #nosec G202 -- placeholder indices only, no user input in query text
		selQ += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
		selArgs = append(selArgs, limit, offset)
	}
	rows, err := s.db.Query(selQ, selArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*network.Environment
	for rows.Next() {
		var id uuid.UUID
		var n string
		var orgID nullUUID
		if err := rows.Scan(&id, &n, &orgID); err != nil {
			return nil, 0, err
		}
		orgUUID := uuid.Nil
		if orgID.Valid {
			orgUUID = orgID.UUID
		}
		out = append(out, &network.Environment{Id: id, Name: n, OrganizationID: orgUUID, Block: []network.Block{}})
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
	_, err = s.db.Exec(`DELETE FROM pools WHERE environment_id = $1`, id)
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

func uuidPtrOptional(p *uuid.UUID) interface{} {
	if p == nil || *p == uuid.Nil {
		return nil
	}
	return *p
}

func timePtrOptional(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

func scanPools(rows *sql.Rows) ([]*network.Pool, error) {
	var out []*network.Pool
	for rows.Next() {
		var p network.Pool
		var prov, extID sql.NullString
		var connID, parentID nullUUID
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.EnvironmentID, &p.Name, &p.CIDR, &prov, &extID, &connID, &parentID); err != nil {
			return nil, err
		}
		if prov.Valid {
			p.Provider = prov.String
		}
		if extID.Valid {
			p.ExternalID = extID.String
		}
		if connID.Valid && connID.UUID != uuid.Nil {
			p.ConnectionID = &connID.UUID
		}
		if parentID.Valid && parentID.UUID != uuid.Nil {
			p.ParentPoolID = &parentID.UUID
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (s *PostgresStore) CreatePool(pool *network.Pool) error {
	if pool.ID == uuid.Nil {
		pool.ID = s.GenerateID()
	}
	provider := pool.Provider
	if provider == "" {
		provider = "native"
	}
	_, err := s.db.Exec(
		`INSERT INTO pools (id, organization_id, environment_id, name, cidr, provider, external_id, connection_id, parent_pool_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		pool.ID, pool.OrganizationID, pool.EnvironmentID, pool.Name, pool.CIDR, provider, nullStr(pool.ExternalID), uuidPtrOptional(pool.ConnectionID), uuidPtrOptional(pool.ParentPoolID),
	)
	return err
}

func (s *PostgresStore) GetPool(id uuid.UUID) (*network.Pool, error) {
	var p network.Pool
	var prov, extID sql.NullString
	var connID, parentID nullUUID
	err := s.db.QueryRow(
		`SELECT id, organization_id, environment_id, name, cidr, COALESCE(provider, 'native'), external_id, connection_id, parent_pool_id FROM pools WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&p.ID, &p.OrganizationID, &p.EnvironmentID, &p.Name, &p.CIDR, &prov, &extID, &connID, &parentID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pool not found")
	}
	if err != nil {
		return nil, err
	}
	if prov.Valid {
		p.Provider = prov.String
	}
	if extID.Valid {
		p.ExternalID = extID.String
	}
	if connID.Valid && connID.UUID != uuid.Nil {
		p.ConnectionID = &connID.UUID
	}
	if parentID.Valid && parentID.UUID != uuid.Nil {
		p.ParentPoolID = &parentID.UUID
	}
	return &p, nil
}

func (s *PostgresStore) ListPoolsByEnvironment(envID uuid.UUID) ([]*network.Pool, error) {
	rows, err := s.db.Query(
		`SELECT id, organization_id, environment_id, name, cidr, COALESCE(provider, 'native'), external_id, connection_id, parent_pool_id FROM pools WHERE environment_id = $1 AND deleted_at IS NULL ORDER BY name`,
		envID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPools(rows)
}

func (s *PostgresStore) ListPoolsByOrganization(orgID uuid.UUID) ([]*network.Pool, error) {
	rows, err := s.db.Query(
		`SELECT id, organization_id, environment_id, name, cidr, COALESCE(provider, 'native'), external_id, connection_id, parent_pool_id FROM pools WHERE organization_id = $1 AND deleted_at IS NULL ORDER BY name`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPools(rows)
}

func (s *PostgresStore) ListPoolsByOrganizationIncludingDeleted(orgID uuid.UUID) ([]*network.Pool, error) {
	rows, err := s.db.Query(
		`SELECT id, organization_id, environment_id, name, cidr, COALESCE(provider, 'native'), external_id, connection_id, parent_pool_id FROM pools WHERE organization_id = $1 ORDER BY name`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPools(rows)
}

func (s *PostgresStore) UpdatePool(id uuid.UUID, pool *network.Pool) error {
	provider := pool.Provider
	if provider == "" {
		provider = "native"
	}
	res, err := s.db.Exec(
		`UPDATE pools SET name = $1, cidr = $2, provider = $3, external_id = $4, connection_id = $5, parent_pool_id = $6, deleted_at = $7 WHERE id = $8`,
		pool.Name, pool.CIDR, provider, nullStr(pool.ExternalID), uuidPtrOptional(pool.ConnectionID), uuidPtrOptional(pool.ParentPoolID), timePtrOptional(pool.DeletedAt), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pool not found")
	}
	return nil
}

func (s *PostgresStore) DeletePool(id uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM pools WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pool not found")
	}
	return nil
}

func (s *PostgresStore) SoftDeletePool(id uuid.UUID) error {
	res, err := s.db.Exec(`UPDATE pools SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pool not found")
	}
	return nil
}

func (s *PostgresStore) ListPoolsPendingCloudDelete(connID uuid.UUID) ([]*network.Pool, error) {
	rows, err := s.db.Query(
		`SELECT id, organization_id, environment_id, name, cidr, COALESCE(provider, 'native'), external_id, connection_id, parent_pool_id FROM pools WHERE connection_id = $1 AND external_id IS NOT NULL AND external_id != '' AND deleted_at IS NOT NULL ORDER BY name`,
		connID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPools(rows)
}

// CloudConnection CRUD
func (s *PostgresStore) CreateCloudConnection(c *CloudConnection) error {
	if c.ID == uuid.Nil {
		c.ID = s.GenerateID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = c.CreatedAt
	}
	config := c.Config
	if config == nil {
		config = []byte("{}")
	}
	syncMode, conflictRes := c.SyncMode, c.ConflictResolution
	if syncMode == "" {
		syncMode = "read_only"
	}
	if conflictRes == "" {
		conflictRes = "cloud"
	}
	_, err := s.db.Exec(
		`INSERT INTO cloud_connections (id, organization_id, provider, name, config, credentials_ref, sync_interval_minutes, sync_mode, conflict_resolution, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		c.ID, c.OrganizationID, c.Provider, c.Name, config, c.CredentialsRef, c.SyncIntervalMinutes, syncMode, conflictRes, c.LastSyncAt, c.LastSyncStatus, c.LastSyncError, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) GetCloudConnection(id uuid.UUID) (*CloudConnection, error) {
	var c CloudConnection
	var config []byte
	var credRef, lastStatus, lastErr sql.NullString
	var lastSyncAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, organization_id, provider, name, config, credentials_ref, sync_interval_minutes, sync_mode, conflict_resolution, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at FROM cloud_connections WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.OrganizationID, &c.Provider, &c.Name, &config, &credRef, &c.SyncIntervalMinutes, &c.SyncMode, &c.ConflictResolution, &lastSyncAt, &lastStatus, &lastErr, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cloud connection not found")
	}
	if err != nil {
		return nil, err
	}
	c.Config = config
	if credRef.Valid {
		c.CredentialsRef = &credRef.String
	}
	if lastSyncAt.Valid {
		c.LastSyncAt = &lastSyncAt.Time
	}
	if lastStatus.Valid {
		c.LastSyncStatus = &lastStatus.String
	}
	if lastErr.Valid {
		c.LastSyncError = &lastErr.String
	}
	if c.SyncMode == "" {
		c.SyncMode = "read_only"
	}
	if c.ConflictResolution == "" {
		c.ConflictResolution = "cloud"
	}
	return &c, nil
}

func (s *PostgresStore) ListCloudConnectionsByOrganization(orgID uuid.UUID) ([]*CloudConnection, error) {
	rows, err := s.db.Query(
		`SELECT id, organization_id, provider, name, config, credentials_ref, sync_interval_minutes, sync_mode, conflict_resolution, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at FROM cloud_connections WHERE organization_id = $1 ORDER BY name`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*CloudConnection
	for rows.Next() {
		var c CloudConnection
		var config []byte
		var credRef, lastStatus, lastErr sql.NullString
		var lastSyncAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.OrganizationID, &c.Provider, &c.Name, &config, &credRef, &c.SyncIntervalMinutes, &c.SyncMode, &c.ConflictResolution, &lastSyncAt, &lastStatus, &lastErr, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Config = config
		if credRef.Valid {
			c.CredentialsRef = &credRef.String
		}
		if lastSyncAt.Valid {
			c.LastSyncAt = &lastSyncAt.Time
		}
		if lastStatus.Valid {
			c.LastSyncStatus = &lastStatus.String
		}
		if lastErr.Valid {
			c.LastSyncError = &lastErr.String
		}
		if c.SyncMode == "" {
			c.SyncMode = "read_only"
		}
		if c.ConflictResolution == "" {
			c.ConflictResolution = "cloud"
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *PostgresStore) UpdateCloudConnection(id uuid.UUID, c *CloudConnection) error {
	c.UpdatedAt = time.Now()
	config := c.Config
	if config == nil {
		config = []byte("{}")
	}
	syncMode, conflictRes := c.SyncMode, c.ConflictResolution
	if syncMode == "" {
		syncMode = "read_only"
	}
	if conflictRes == "" {
		conflictRes = "cloud"
	}
	res, err := s.db.Exec(
		`UPDATE cloud_connections SET name = $1, config = $2, credentials_ref = $3, sync_interval_minutes = $4, sync_mode = $5, conflict_resolution = $6, last_sync_at = $7, last_sync_status = $8, last_sync_error = $9, updated_at = $10 WHERE id = $11`,
		c.Name, config, c.CredentialsRef, c.SyncIntervalMinutes, syncMode, conflictRes, c.LastSyncAt, c.LastSyncStatus, c.LastSyncError, c.UpdatedAt, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("cloud connection not found")
	}
	return nil
}

func (s *PostgresStore) ListCloudConnections() ([]*CloudConnection, error) {
	rows, err := s.db.Query(
		`SELECT id, organization_id, provider, name, config, credentials_ref, sync_interval_minutes, sync_mode, conflict_resolution, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at FROM cloud_connections ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*CloudConnection
	for rows.Next() {
		var c CloudConnection
		var config []byte
		var credRef, lastStatus, lastErr sql.NullString
		var lastSyncAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.OrganizationID, &c.Provider, &c.Name, &config, &credRef, &c.SyncIntervalMinutes, &c.SyncMode, &c.ConflictResolution, &lastSyncAt, &lastStatus, &lastErr, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Config = config
		if credRef.Valid {
			c.CredentialsRef = &credRef.String
		}
		if lastSyncAt.Valid {
			c.LastSyncAt = &lastSyncAt.Time
		}
		if lastStatus.Valid {
			c.LastSyncStatus = &lastStatus.String
		}
		if lastErr.Valid {
			c.LastSyncError = &lastErr.String
		}
		if c.SyncMode == "" {
			c.SyncMode = "read_only"
		}
		if c.ConflictResolution == "" {
			c.ConflictResolution = "cloud"
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *PostgresStore) DeleteCloudConnection(id uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM cloud_connections WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("cloud connection not found")
	}
	return nil
}

// syncLockKey derives a bigint key from a connection UUID for pg_try_advisory_lock.
// Uses lower 63 bits to avoid uint64->int64 overflow (G115).
func syncLockKey(connectionID uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(connectionID[:8]) & 0x7FFFFFFFFFFFFFFF)
}

func (s *PostgresStore) WithSyncLock(ctx context.Context, connectionID uuid.UUID, fn func() error) (acquired bool, err error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	key := syncLockKey(connectionID)
	var ok bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&ok); err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", key)
	}()
	err = fn()
	return true, err
}

func (s *PostgresStore) CreateBlock(block *network.Block) error {
	if block.ID == uuid.Nil {
		block.ID = s.GenerateID()
	}
	provider := block.Provider
	if provider == "" {
		provider = "native"
	}
	total := network.CIDRAddressCountInt64(block.CIDR)
	_, err := s.db.Exec(
		`INSERT INTO blocks (id, name, cidr, environment_id, organization_id, pool_id, total_ips, provider, external_id, connection_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		block.ID, block.Name, block.CIDR, uuidPtr(block.EnvironmentID), uuidPtr(block.OrganizationID), uuidPtrOptional(block.PoolID), total, provider, nullStr(block.ExternalID), uuidPtrOptional(block.ConnectionID),
	)
	return err
}

func (s *PostgresStore) GetBlock(id uuid.UUID) (*network.Block, error) {
	var name, cidr string
	var envID, orgID, poolID, connID nullUUID
	var totalIPs int64
	var prov, extID sql.NullString
	err := s.db.QueryRow(
		`SELECT id, name, cidr, environment_id, organization_id, pool_id, total_ips, COALESCE(provider, 'native'), external_id, connection_id FROM blocks WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&id, &name, &cidr, &envID, &orgID, &poolID, &totalIPs, &prov, &extID, &connID)
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
	orgUUID := uuid.Nil
	if orgID.Valid && orgID.UUID != uuid.Nil {
		orgUUID = orgID.UUID
	}
	var poolUUID *uuid.UUID
	if poolID.Valid && poolID.UUID != uuid.Nil {
		poolUUID = &poolID.UUID
	}
	var connUUID *uuid.UUID
	if connID.Valid && connID.UUID != uuid.Nil {
		connUUID = &connID.UUID
	}
	b := &network.Block{
		ID:             id,
		Name:           name,
		CIDR:           cidr,
		EnvironmentID:  envUUID,
		OrganizationID: orgUUID,
		PoolID:         poolUUID,
		ConnectionID:   connUUID,
		Usage:          network.Usage{TotalIPs: int(totalIPs), UsedIPs: 0, AvailableIPs: int(totalIPs)},
		Children:       []network.Block{},
	}
	if prov.Valid {
		b.Provider = prov.String
	}
	if extID.Valid {
		b.ExternalID = extID.String
	}
	return b, nil
}

func (s *PostgresStore) ListBlocks() ([]*network.Block, error) {
	out, _, err := s.ListBlocksFiltered("", nil, nil, nil, false, "", nil, 0, 0)
	return out, err
}

func (s *PostgresStore) ListBlocksFiltered(name string, environmentID *uuid.UUID, poolID *uuid.UUID, organizationID *uuid.UUID, orphanedOnly bool, provider string, connectionID *uuid.UUID, limit, offset int) ([]*network.Block, int, error) {
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
	if poolID != nil {
		countQ += fmt.Sprintf(` AND pool_id = $%d`, idx)
		countArgs = append(countArgs, *poolID)
		idx++
	}
	if organizationID != nil {
		countQ += fmt.Sprintf(` AND ((environment_id IN (SELECT id FROM environments WHERE organization_id = $%d)) OR (environment_id IS NULL AND organization_id = $%d))`, idx, idx)
		countArgs = append(countArgs, *organizationID)
		idx++
	}
	if orphanedOnly {
		countQ += ` AND environment_id IS NULL`
		if organizationID != nil {
			countQ += fmt.Sprintf(` AND organization_id = $%d`, idx)
			countArgs = append(countArgs, *organizationID)
			idx++
		}
	}
	if provider != "" {
		countQ += fmt.Sprintf(` AND COALESCE(provider, 'native') = $%d`, idx)
		countArgs = append(countArgs, provider)
		idx++
	}
	if connectionID != nil {
		countQ += fmt.Sprintf(` AND connection_id = $%d`, idx)
		countArgs = append(countArgs, *connectionID)
		idx++
	}
	countQ += ` AND deleted_at IS NULL`
	var total int
	if err := s.db.QueryRow(countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	selQ := `SELECT id, name, cidr, environment_id, organization_id, pool_id, total_ips, COALESCE(provider, 'native'), external_id, connection_id FROM blocks WHERE 1=1 AND deleted_at IS NULL`
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
	if poolID != nil {
		selQ += fmt.Sprintf(` AND pool_id = $%d`, i)
		selArgs = append(selArgs, *poolID)
		i++
	}
	if organizationID != nil {
		selQ += fmt.Sprintf(` AND ((environment_id IN (SELECT id FROM environments WHERE organization_id = $%d)) OR (environment_id IS NULL AND organization_id = $%d))`, i, i)
		selArgs = append(selArgs, *organizationID)
		i++
	}
	if orphanedOnly {
		selQ += ` AND environment_id IS NULL`
		if organizationID != nil {
			selQ += fmt.Sprintf(` AND organization_id = $%d`, i)
			selArgs = append(selArgs, *organizationID)
			i++
		}
	}
	if provider != "" {
		selQ += fmt.Sprintf(` AND COALESCE(provider, 'native') = $%d`, i)
		selArgs = append(selArgs, provider)
		i++
	}
	if connectionID != nil {
		selQ += fmt.Sprintf(` AND connection_id = $%d`, i)
		selArgs = append(selArgs, *connectionID)
		i++
	}
	selQ += ` ORDER BY name`
	if limit > 0 {
		// #nosec G202 -- placeholder indices only, no user input in query text
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
		var envID, orgID, poolID, connID nullUUID
		var totalIPs int64
		var prov, extID sql.NullString
		if err := rows.Scan(&id, &n, &cidr, &envID, &orgID, &poolID, &totalIPs, &prov, &extID, &connID); err != nil {
			return nil, 0, err
		}
		envUUID := uuid.Nil
		if envID.Valid {
			envUUID = envID.UUID
		}
		orgUUID := uuid.Nil
		if orgID.Valid && orgID.UUID != uuid.Nil {
			orgUUID = orgID.UUID
		}
		var poolUUID *uuid.UUID
		if poolID.Valid && poolID.UUID != uuid.Nil {
			poolUUID = &poolID.UUID
		}
		var connUUID *uuid.UUID
		if connID.Valid && connID.UUID != uuid.Nil {
			connUUID = &connID.UUID
		}
		b := &network.Block{
			ID:             id,
			Name:           n,
			CIDR:           cidr,
			EnvironmentID:  envUUID,
			OrganizationID: orgUUID,
			PoolID:         poolUUID,
			ConnectionID:   connUUID,
			Usage:          network.Usage{TotalIPs: int(totalIPs), UsedIPs: 0, AvailableIPs: int(totalIPs)},
			Children:       []network.Block{},
		}
		if prov.Valid {
			b.Provider = prov.String
		}
		if extID.Valid {
			b.ExternalID = extID.String
		}
		out = append(out, b)
	}
	return out, total, rows.Err()
}

func (s *PostgresStore) ListBlocksByEnvironment(envID uuid.UUID) ([]*network.Block, error) {
	blocks, _, err := s.ListBlocksFiltered("", &envID, nil, nil, false, "", nil, 0, 0)
	return blocks, err
}

func (s *PostgresStore) ListBlocksByPool(poolID uuid.UUID) ([]*network.Block, error) {
	rows, err := s.db.Query(
		`SELECT id, name, cidr, environment_id, organization_id, pool_id, total_ips, COALESCE(provider, 'native'), external_id, connection_id FROM blocks WHERE pool_id = $1 AND deleted_at IS NULL ORDER BY name`,
		poolID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*network.Block
	for rows.Next() {
		var id uuid.UUID
		var n, cidr string
		var envID, orgID, poolIDCol, connID nullUUID
		var totalIPs int64
		var prov, extID sql.NullString
		if err := rows.Scan(&id, &n, &cidr, &envID, &orgID, &poolIDCol, &totalIPs, &prov, &extID, &connID); err != nil {
			return nil, err
		}
		envUUID := uuid.Nil
		if envID.Valid {
			envUUID = envID.UUID
		}
		orgUUID := uuid.Nil
		if orgID.Valid && orgID.UUID != uuid.Nil {
			orgUUID = orgID.UUID
		}
		var poolUUID *uuid.UUID
		if poolIDCol.Valid && poolIDCol.UUID != uuid.Nil {
			poolUUID = &poolIDCol.UUID
		}
		var connUUID *uuid.UUID
		if connID.Valid && connID.UUID != uuid.Nil {
			connUUID = &connID.UUID
		}
		b := &network.Block{
			ID:             id,
			Name:           n,
			CIDR:           cidr,
			EnvironmentID:  envUUID,
			OrganizationID: orgUUID,
			PoolID:         poolUUID,
			ConnectionID:   connUUID,
			Usage:          network.Usage{TotalIPs: int(totalIPs), UsedIPs: 0, AvailableIPs: int(totalIPs)},
			Children:       []network.Block{},
		}
		if prov.Valid {
			b.Provider = prov.String
		}
		if extID.Valid {
			b.ExternalID = extID.String
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *PostgresStore) UpdateBlock(id uuid.UUID, block *network.Block) error {
	provider := block.Provider
	if provider == "" {
		provider = "native"
	}
	total := network.CIDRAddressCountInt64(block.CIDR)
	res, err := s.db.Exec(
		`UPDATE blocks SET name = $1, cidr = $2, environment_id = $3, organization_id = $4, pool_id = $5, total_ips = $6, provider = $7, external_id = $8, connection_id = $9, deleted_at = $10 WHERE id = $11`,
		block.Name, block.CIDR, uuidPtr(block.EnvironmentID), uuidPtr(block.OrganizationID), uuidPtrOptional(block.PoolID), total, provider, nullStr(block.ExternalID), uuidPtrOptional(block.ConnectionID), timePtrOptional(block.DeletedAt), id,
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
	return nil
}

func (s *PostgresStore) SoftDeleteBlock(id uuid.UUID) error {
	res, err := s.db.Exec(`UPDATE blocks SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("block not found")
	}
	return nil
}

func (s *PostgresStore) ListBlocksPendingCloudDelete(connID uuid.UUID) ([]*network.Block, error) {
	rows, err := s.db.Query(
		`SELECT id, name, cidr, environment_id, organization_id, pool_id, total_ips, COALESCE(provider, 'native'), external_id, connection_id FROM blocks WHERE connection_id = $1 AND external_id IS NOT NULL AND external_id != '' AND deleted_at IS NOT NULL ORDER BY name`,
		connID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*network.Block
	for rows.Next() {
		var id uuid.UUID
		var n, cidr string
		var envID, orgID, poolIDCol, connID nullUUID
		var totalIPs int64
		var prov, extID sql.NullString
		if err := rows.Scan(&id, &n, &cidr, &envID, &orgID, &poolIDCol, &totalIPs, &prov, &extID, &connID); err != nil {
			return nil, err
		}
		envUUID := uuid.Nil
		if envID.Valid {
			envUUID = envID.UUID
		}
		orgUUID := uuid.Nil
		if orgID.Valid && orgID.UUID != uuid.Nil {
			orgUUID = orgID.UUID
		}
		var poolUUID *uuid.UUID
		if poolIDCol.Valid && poolIDCol.UUID != uuid.Nil {
			poolUUID = &poolIDCol.UUID
		}
		var connUUID *uuid.UUID
		if connID.Valid && connID.UUID != uuid.Nil {
			connUUID = &connID.UUID
		}
		b := &network.Block{
			ID:             id,
			Name:           n,
			CIDR:           cidr,
			EnvironmentID:  envUUID,
			OrganizationID: orgUUID,
			PoolID:         poolUUID,
			ConnectionID:   connUUID,
			Usage:          network.Usage{TotalIPs: int(totalIPs), UsedIPs: 0, AvailableIPs: int(totalIPs)},
			Children:       []network.Block{},
		}
		if prov.Valid {
			b.Provider = prov.String
		}
		if extID.Valid {
			b.ExternalID = extID.String
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *PostgresStore) ListBlocksFilteredIncludingDeleted(name string, environmentID *uuid.UUID, poolID *uuid.UUID, organizationID *uuid.UUID, orphanedOnly bool, provider string, connectionID *uuid.UUID, limit, offset int) ([]*network.Block, int, error) {
	// Same as ListBlocksFiltered but without deleted_at IS NULL (for sync to match cloud to soft-deleted rows)
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
	if poolID != nil {
		countQ += fmt.Sprintf(` AND pool_id = $%d`, idx)
		countArgs = append(countArgs, *poolID)
		idx++
	}
	if organizationID != nil {
		countQ += fmt.Sprintf(` AND ((environment_id IN (SELECT id FROM environments WHERE organization_id = $%d)) OR (environment_id IS NULL AND organization_id = $%d))`, idx, idx)
		countArgs = append(countArgs, *organizationID)
		idx++
	}
	if orphanedOnly {
		countQ += ` AND environment_id IS NULL`
		if organizationID != nil {
			countQ += fmt.Sprintf(` AND organization_id = $%d`, idx)
			countArgs = append(countArgs, *organizationID)
			idx++
		}
	}
	if provider != "" {
		countQ += fmt.Sprintf(` AND COALESCE(provider, 'native') = $%d`, idx)
		countArgs = append(countArgs, provider)
		idx++
	}
	if connectionID != nil {
		countQ += fmt.Sprintf(` AND connection_id = $%d`, idx)
		countArgs = append(countArgs, *connectionID)
		idx++
	}
	var total int
	if err := s.db.QueryRow(countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	selQ := `SELECT id, name, cidr, environment_id, organization_id, pool_id, total_ips, COALESCE(provider, 'native'), external_id, connection_id FROM blocks WHERE 1=1`
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
	if poolID != nil {
		selQ += fmt.Sprintf(` AND pool_id = $%d`, i)
		selArgs = append(selArgs, *poolID)
		i++
	}
	if organizationID != nil {
		selQ += fmt.Sprintf(` AND ((environment_id IN (SELECT id FROM environments WHERE organization_id = $%d)) OR (environment_id IS NULL AND organization_id = $%d))`, i, i)
		selArgs = append(selArgs, *organizationID)
		i++
	}
	if orphanedOnly {
		selQ += ` AND environment_id IS NULL`
		if organizationID != nil {
			selQ += fmt.Sprintf(` AND organization_id = $%d`, i)
			selArgs = append(selArgs, *organizationID)
			i++
		}
	}
	if provider != "" {
		selQ += fmt.Sprintf(` AND COALESCE(provider, 'native') = $%d`, i)
		selArgs = append(selArgs, provider)
		i++
	}
	if connectionID != nil {
		selQ += fmt.Sprintf(` AND connection_id = $%d`, i)
		selArgs = append(selArgs, *connectionID)
		i++
	}
	selQ += ` ORDER BY name`
	if limit > 0 {
		// #nosec G202 -- placeholder indices only, no user input in query text
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
		var envID, orgID, poolIDCol, connID nullUUID
		var totalIPs int64
		var prov, extID sql.NullString
		if err := rows.Scan(&id, &n, &cidr, &envID, &orgID, &poolIDCol, &totalIPs, &prov, &extID, &connID); err != nil {
			return nil, 0, err
		}
		envUUID := uuid.Nil
		if envID.Valid {
			envUUID = envID.UUID
		}
		orgUUID := uuid.Nil
		if orgID.Valid && orgID.UUID != uuid.Nil {
			orgUUID = orgID.UUID
		}
		var poolUUID *uuid.UUID
		if poolIDCol.Valid && poolIDCol.UUID != uuid.Nil {
			poolUUID = &poolIDCol.UUID
		}
		var connUUID *uuid.UUID
		if connID.Valid && connID.UUID != uuid.Nil {
			connUUID = &connID.UUID
		}
		b := &network.Block{
			ID:             id,
			Name:           n,
			CIDR:           cidr,
			EnvironmentID:  envUUID,
			OrganizationID: orgUUID,
			PoolID:         poolUUID,
			ConnectionID:   connUUID,
			Usage:          network.Usage{TotalIPs: int(totalIPs), UsedIPs: 0, AvailableIPs: int(totalIPs)},
			Children:       []network.Block{},
		}
		if prov.Valid {
			b.Provider = prov.String
		}
		if extID.Valid {
			b.ExternalID = extID.String
		}
		out = append(out, b)
	}
	return out, total, rows.Err()
}

func (s *PostgresStore) CreateAllocation(id uuid.UUID, alloc *network.Allocation) error {
	provider := alloc.Provider
	if provider == "" {
		provider = "native"
	}
	_, err := s.db.Exec(
		`INSERT INTO allocations (id, name, block_name, block_cidr, provider, external_id, connection_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, alloc.Name, alloc.Block.Name, alloc.Block.CIDR, provider, nullStr(alloc.ExternalID), uuidPtrOptional(alloc.ConnectionID),
	)
	return err
}

func (s *PostgresStore) GetAllocation(id uuid.UUID) (*network.Allocation, error) {
	var name, blockName, blockCidr string
	var prov sql.NullString
	var extID sql.NullString
	var connID nullUUID
	err := s.db.QueryRow(
		`SELECT id, name, block_name, block_cidr, provider, external_id, connection_id FROM allocations WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&id, &name, &blockName, &blockCidr, &prov, &extID, &connID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("allocation not found")
	}
	if err != nil {
		return nil, err
	}
	out := &network.Allocation{
		Id:    id,
		Name:  name,
		Block: network.Block{Name: blockName, CIDR: blockCidr},
	}
	if prov.Valid {
		out.Provider = prov.String
	}
	if extID.Valid {
		out.ExternalID = extID.String
	}
	if connID.Valid {
		out.ConnectionID = &connID.UUID
	}
	return out, nil
}

func (s *PostgresStore) ListAllocations() ([]*network.Allocation, error) {
	out, _, err := s.ListAllocationsFiltered("", "", uuid.Nil, nil, "", nil, 0, 0)
	return out, err
}

func (s *PostgresStore) ListAllocationsFiltered(name string, blockName string, environmentID uuid.UUID, organizationID *uuid.UUID, provider string, connectionID *uuid.UUID, limit, offset int) ([]*network.Allocation, int, error) {
	name = strings.TrimSpace(name)
	blockName = strings.TrimSpace(blockName)
	nameLower := strings.ToLower(name)
	blockLower := strings.ToLower(blockName)
	envFilter := environmentID != uuid.Nil
	var total int
	countQ := `SELECT COUNT(*) FROM allocations WHERE ($1 = '' OR LOWER(name) LIKE '%' || $1 || '%') AND ($2 = '' OR LOWER(block_name) = $2) AND deleted_at IS NULL`
	args := []interface{}{nameLower, blockLower}
	idx := 3
	if envFilter {
		countQ += fmt.Sprintf(` AND LOWER(block_name) IN (SELECT LOWER(name) FROM blocks WHERE environment_id = $%d AND deleted_at IS NULL)`, idx)
		args = append(args, environmentID)
		idx++
	}
	if organizationID != nil {
		countQ += fmt.Sprintf(` AND LOWER(block_name) IN (SELECT LOWER(b.name) FROM blocks b LEFT JOIN environments e ON b.environment_id = e.id WHERE (e.organization_id = $%d OR (b.environment_id IS NULL AND b.organization_id = $%d)) AND b.deleted_at IS NULL)`, idx, idx)
		args = append(args, *organizationID)
		idx++
	}
	if provider != "" {
		countQ += fmt.Sprintf(` AND provider = $%d`, idx)
		args = append(args, provider)
		idx++
	}
	if connectionID != nil {
		countQ += fmt.Sprintf(` AND connection_id = $%d`, idx)
		args = append(args, *connectionID)
		idx++
	}
	if err := s.db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	selQ := `SELECT id, name, block_name, block_cidr, provider, external_id, connection_id FROM allocations WHERE ($1 = '' OR LOWER(name) LIKE '%' || $1 || '%') AND ($2 = '' OR LOWER(block_name) = $2) AND deleted_at IS NULL`
	selArgs := []interface{}{nameLower, blockLower}
	i := 3
	if envFilter {
		selQ += fmt.Sprintf(` AND LOWER(block_name) IN (SELECT LOWER(name) FROM blocks WHERE environment_id = $%d AND deleted_at IS NULL)`, i)
		selArgs = append(selArgs, environmentID)
		i++
	}
	if organizationID != nil {
		selQ += fmt.Sprintf(` AND LOWER(block_name) IN (SELECT LOWER(b.name) FROM blocks b LEFT JOIN environments e ON b.environment_id = e.id WHERE (e.organization_id = $%d OR (b.environment_id IS NULL AND b.organization_id = $%d)) AND b.deleted_at IS NULL)`, i, i)
		selArgs = append(selArgs, *organizationID)
		i++
	}
	if provider != "" {
		selQ += fmt.Sprintf(` AND provider = $%d`, i)
		selArgs = append(selArgs, provider)
		i++
	}
	if connectionID != nil {
		selQ += fmt.Sprintf(` AND connection_id = $%d`, i)
		selArgs = append(selArgs, *connectionID)
		i++
	}
	selQ += ` ORDER BY name`
	if limit > 0 {
		// #nosec G202 -- placeholder indices only, no user input in query text
		selQ += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, i, i+1)
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
		var prov sql.NullString
		var extID sql.NullString
		var connID nullUUID
		if err := rows.Scan(&id, &n, &bn, &bc, &prov, &extID, &connID); err != nil {
			return nil, 0, err
		}
		a := &network.Allocation{Id: id, Name: n, Block: network.Block{Name: bn, CIDR: bc}}
		if prov.Valid {
			a.Provider = prov.String
		}
		if extID.Valid {
			a.ExternalID = extID.String
		}
		if connID.Valid {
			a.ConnectionID = &connID.UUID
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (s *PostgresStore) UpdateAllocation(id uuid.UUID, alloc *network.Allocation) error {
	provider := alloc.Provider
	if provider == "" {
		provider = "native"
	}
	res, err := s.db.Exec(
		`UPDATE allocations SET name = $1, block_name = $2, block_cidr = $3, provider = $4, external_id = $5, connection_id = $6, deleted_at = $7 WHERE id = $8`,
		alloc.Name, alloc.Block.Name, alloc.Block.CIDR, provider, nullStr(alloc.ExternalID), uuidPtrOptional(alloc.ConnectionID), timePtrOptional(alloc.DeletedAt), id,
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

func (s *PostgresStore) SoftDeleteAllocation(id uuid.UUID) error {
	res, err := s.db.Exec(`UPDATE allocations SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("allocation not found")
	}
	return nil
}

func (s *PostgresStore) ListAllocationsPendingCloudDelete(connID uuid.UUID) ([]*network.Allocation, error) {
	rows, err := s.db.Query(
		`SELECT id, name, block_name, block_cidr, provider, external_id, connection_id FROM allocations WHERE connection_id = $1 AND external_id IS NOT NULL AND external_id != '' AND deleted_at IS NOT NULL ORDER BY name`,
		connID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*network.Allocation
	for rows.Next() {
		var id uuid.UUID
		var n, bn, bc string
		var prov sql.NullString
		var extID sql.NullString
		var connID nullUUID
		if err := rows.Scan(&id, &n, &bn, &bc, &prov, &extID, &connID); err != nil {
			return nil, err
		}
		a := &network.Allocation{Id: id, Name: n, Block: network.Block{Name: bn, CIDR: bc}}
		if prov.Valid {
			a.Provider = prov.String
		}
		if extID.Valid {
			a.ExternalID = extID.String
		}
		if connID.Valid {
			a.ConnectionID = &connID.UUID
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *PostgresStore) ListAllocationsFilteredIncludingDeleted(name string, blockName string, environmentID uuid.UUID, organizationID *uuid.UUID, provider string, connectionID *uuid.UUID, limit, offset int) ([]*network.Allocation, int, error) {
	name = strings.TrimSpace(name)
	blockName = strings.TrimSpace(blockName)
	nameLower := strings.ToLower(name)
	blockLower := strings.ToLower(blockName)
	envFilter := environmentID != uuid.Nil
	var total int
	countQ := `SELECT COUNT(*) FROM allocations WHERE ($1 = '' OR LOWER(name) LIKE '%' || $1 || '%') AND ($2 = '' OR LOWER(block_name) = $2)`
	args := []interface{}{nameLower, blockLower}
	idx := 3
	if envFilter {
		countQ += fmt.Sprintf(` AND LOWER(block_name) IN (SELECT LOWER(name) FROM blocks WHERE environment_id = $%d)`, idx)
		args = append(args, environmentID)
		idx++
	}
	if organizationID != nil {
		countQ += fmt.Sprintf(` AND LOWER(block_name) IN (SELECT LOWER(b.name) FROM blocks b LEFT JOIN environments e ON b.environment_id = e.id WHERE (e.organization_id = $%d OR (b.environment_id IS NULL AND b.organization_id = $%d)))`, idx, idx)
		args = append(args, *organizationID)
		idx++
	}
	if provider != "" {
		countQ += fmt.Sprintf(` AND provider = $%d`, idx)
		args = append(args, provider)
		idx++
	}
	if connectionID != nil {
		countQ += fmt.Sprintf(` AND connection_id = $%d`, idx)
		args = append(args, *connectionID)
		idx++
	}
	if err := s.db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	selQ := `SELECT id, name, block_name, block_cidr, provider, external_id, connection_id FROM allocations WHERE ($1 = '' OR LOWER(name) LIKE '%' || $1 || '%') AND ($2 = '' OR LOWER(block_name) = $2)`
	selArgs := []interface{}{nameLower, blockLower}
	i := 3
	if envFilter {
		selQ += fmt.Sprintf(` AND LOWER(block_name) IN (SELECT LOWER(name) FROM blocks WHERE environment_id = $%d)`, i)
		selArgs = append(selArgs, environmentID)
		i++
	}
	if organizationID != nil {
		selQ += fmt.Sprintf(` AND LOWER(block_name) IN (SELECT LOWER(b.name) FROM blocks b LEFT JOIN environments e ON b.environment_id = e.id WHERE (e.organization_id = $%d OR (b.environment_id IS NULL AND b.organization_id = $%d)))`, i, i)
		selArgs = append(selArgs, *organizationID)
		i++
	}
	if provider != "" {
		selQ += fmt.Sprintf(` AND provider = $%d`, i)
		selArgs = append(selArgs, provider)
		i++
	}
	if connectionID != nil {
		selQ += fmt.Sprintf(` AND connection_id = $%d`, i)
		selArgs = append(selArgs, *connectionID)
		i++
	}
	selQ += ` ORDER BY name`
	if limit > 0 {
		// #nosec G202 -- placeholder indices only, no user input in query text
		selQ += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, i, i+1)
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
		var prov sql.NullString
		var extID sql.NullString
		var connID nullUUID
		if err := rows.Scan(&id, &n, &bn, &bc, &prov, &extID, &connID); err != nil {
			return nil, 0, err
		}
		a := &network.Allocation{Id: id, Name: n, Block: network.Block{Name: bn, CIDR: bc}}
		if prov.Valid {
			a.Provider = prov.String
		}
		if extID.Valid {
			a.ExternalID = extID.String
		}
		if connID.Valid {
			a.ConnectionID = &connID.UUID
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (s *PostgresStore) ListReservedBlocks(organizationID *uuid.UUID) ([]*ReservedBlock, error) {
	q := `SELECT id, name, cidr, reason, created_at, organization_id FROM reserved_blocks`
	args := []interface{}{}
	if organizationID != nil {
		q += ` WHERE organization_id = $1`
		args = append(args, *organizationID)
	}
	q += ` ORDER BY name, cidr`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ReservedBlock
	for rows.Next() {
		var r ReservedBlock
		var orgID nullUUID
		if err := rows.Scan(&r.ID, &r.Name, &r.CIDR, &r.Reason, &r.CreatedAt, &orgID); err != nil {
			return nil, err
		}
		if orgID.Valid {
			r.OrganizationID = orgID.UUID
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
		`INSERT INTO reserved_blocks (id, name, cidr, reason, created_at, organization_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		r.ID, strings.TrimSpace(r.Name), r.CIDR, r.Reason, r.CreatedAt, r.OrganizationID,
	)
	return err
}

func (s *PostgresStore) GetReservedBlock(id uuid.UUID) (*ReservedBlock, error) {
	var r ReservedBlock
	var orgID nullUUID
	err := s.db.QueryRow(`SELECT id, name, cidr, reason, created_at, organization_id FROM reserved_blocks WHERE id = $1`, id).Scan(&r.ID, &r.Name, &r.CIDR, &r.Reason, &r.CreatedAt, &orgID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("reserved block not found")
	}
	if err != nil {
		return nil, err
	}
	if orgID.Valid {
		r.OrganizationID = orgID.UUID
	}
	return &r, nil
}

func (s *PostgresStore) UpdateReservedBlock(id uuid.UUID, r *ReservedBlock) error {
	res, err := s.db.Exec(
		`UPDATE reserved_blocks SET name = $1, cidr = $2, reason = $3 WHERE id = $4`,
		strings.TrimSpace(r.Name), r.CIDR, r.Reason, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("reserved block not found")
	}
	return nil
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

func (s *PostgresStore) OverlapsReservedBlock(cidr string, organizationID *uuid.UUID) (*ReservedBlock, error) {
	list, err := s.ListReservedBlocks(organizationID)
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
		`INSERT INTO users (id, email, password_hash, role, tour_completed, organization_id, oauth_provider, oauth_provider_user_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		u.ID.String(), email, u.PasswordHash, u.Role, u.TourCompleted, uuidPtr(u.OrganizationID), nullStr(u.OAuthProvider), nullStr(u.OAuthProviderUserID),
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return fmt.Errorf("user with email already exists")
		}
		return fmt.Errorf("failed to create user")
	}
	return nil
}

func (s *PostgresStore) GetUser(id uuid.UUID) (*User, error) {
	var u User
	var orgID nullUUID
	var oauthProvider, oauthProviderUserID sql.NullString
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, role, tour_completed, organization_id, oauth_provider, oauth_provider_user_id FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TourCompleted, &orgID, &oauthProvider, &oauthProviderUserID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	if orgID.Valid {
		u.OrganizationID = orgID.UUID
	}
	if oauthProvider.Valid {
		u.OAuthProvider = oauthProvider.String
	}
	if oauthProviderUserID.Valid {
		u.OAuthProviderUserID = oauthProviderUserID.String
	}
	return &u, nil
}

func (s *PostgresStore) GetUserByEmail(email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var u User
	var orgID nullUUID
	var oauthProvider, oauthProviderUserID sql.NullString
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, role, tour_completed, organization_id, oauth_provider, oauth_provider_user_id FROM users WHERE LOWER(email) = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TourCompleted, &orgID, &oauthProvider, &oauthProviderUserID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	if orgID.Valid {
		u.OrganizationID = orgID.UUID
	}
	if oauthProvider.Valid {
		u.OAuthProvider = oauthProvider.String
	}
	if oauthProviderUserID.Valid {
		u.OAuthProviderUserID = oauthProviderUserID.String
	}
	return &u, nil
}

func (s *PostgresStore) GetUserByOAuth(provider, providerUserID string) (*User, error) {
	if provider == "" || providerUserID == "" {
		return nil, fmt.Errorf("user not found")
	}
	var u User
	var orgID nullUUID
	var oauthProvider, oauthProviderUserID sql.NullString
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, role, tour_completed, organization_id, oauth_provider, oauth_provider_user_id FROM users WHERE oauth_provider = $1 AND oauth_provider_user_id = $2`,
		provider, providerUserID,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TourCompleted, &orgID, &oauthProvider, &oauthProviderUserID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	if orgID.Valid {
		u.OrganizationID = orgID.UUID
	}
	if oauthProvider.Valid {
		u.OAuthProvider = oauthProvider.String
	}
	if oauthProviderUserID.Valid {
		u.OAuthProviderUserID = oauthProviderUserID.String
	}
	return &u, nil
}

func (s *PostgresStore) ListUsers(organizationID *uuid.UUID) ([]*User, error) {
	q := `SELECT id, email, password_hash, role, tour_completed, organization_id, oauth_provider, oauth_provider_user_id FROM users`
	args := []interface{}{}
	if organizationID != nil {
		q += ` WHERE organization_id = $1`
		args = append(args, *organizationID)
	}
	q += ` ORDER BY email`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		var u User
		var orgID nullUUID
		var oauthProvider, oauthProviderUserID sql.NullString
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TourCompleted, &orgID, &oauthProvider, &oauthProviderUserID); err != nil {
			return nil, err
		}
		if orgID.Valid {
			u.OrganizationID = orgID.UUID
		}
		if oauthProvider.Valid {
			u.OAuthProvider = oauthProvider.String
		}
		if oauthProviderUserID.Valid {
			u.OAuthProviderUserID = oauthProviderUserID.String
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

func (s *PostgresStore) SetUserOrganization(userID uuid.UUID, organizationID uuid.UUID) error {
	res, err := s.db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, uuidPtr(organizationID), userID)
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

func (s *PostgresStore) SetUserOAuth(userID uuid.UUID, provider, providerUserID string) error {
	res, err := s.db.Exec(`UPDATE users SET oauth_provider = $1, oauth_provider_user_id = $2 WHERE id = $3`, provider, providerUserID, userID)
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

func (s *PostgresStore) CreateAPIToken(userID uuid.UUID, name string, expiresAt *time.Time, organizationID *uuid.UUID) (token *APIToken, rawToken string, err error) {
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
	orgID := uuid.Nil
	if organizationID != nil {
		orgID = *organizationID
	}
	token = &APIToken{
		ID:             id,
		UserID:         userID,
		Name:           strings.TrimSpace(name),
		KeyHash:        keyHash,
		CreatedAt:      time.Now(),
		ExpiresAt:      expiresAt,
		OrganizationID: orgID,
	}
	_, err = s.db.Exec(
		`INSERT INTO api_tokens (id, user_id, name, key_hash, created_at, expires_at, organization_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		token.ID, token.UserID, token.Name, token.KeyHash, token.CreatedAt, token.ExpiresAt, uuidPtr(token.OrganizationID),
	)
	if err != nil {
		return nil, "", err
	}
	return token, rawToken, nil
}

func (s *PostgresStore) GetUserByTokenHash(keyHash string) (*User, error) {
	tok, err := s.GetAPITokenByKeyHash(keyHash)
	if err != nil {
		return nil, err
	}
	return s.GetUser(tok.UserID)
}

func (s *PostgresStore) GetAPITokenByKeyHash(keyHash string) (*APIToken, error) {
	var t APIToken
	var expiresAt sql.NullTime
	var orgID nullUUID
	err := s.db.QueryRow(
		`SELECT id, user_id, name, key_hash, created_at, expires_at, organization_id FROM api_tokens WHERE key_hash = $1`,
		keyHash,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.KeyHash, &t.CreatedAt, &expiresAt, &orgID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		return nil, fmt.Errorf("token expired")
	}
	if orgID.Valid {
		t.OrganizationID = orgID.UUID
	}
	return &t, nil
}

func (s *PostgresStore) ListAPITokens(userID uuid.UUID) ([]*APIToken, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, name, key_hash, created_at, expires_at, organization_id FROM api_tokens WHERE user_id = $1 ORDER BY created_at`,
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
		var orgID nullUUID
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.KeyHash, &t.CreatedAt, &expiresAt, &orgID); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			t.ExpiresAt = &expiresAt.Time
		}
		if orgID.Valid {
			t.OrganizationID = orgID.UUID
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
	var orgID nullUUID
	err := s.db.QueryRow(
		`SELECT id, user_id, name, key_hash, created_at, expires_at, organization_id FROM api_tokens WHERE id = $1`,
		tokenID,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.KeyHash, &t.CreatedAt, &expiresAt, &orgID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		t.ExpiresAt = &expiresAt.Time
	}
	if orgID.Valid {
		t.OrganizationID = orgID.UUID
	}
	return &t, nil
}

func (s *PostgresStore) CreateSignupInvite(createdBy uuid.UUID, expiresAt time.Time, organizationID uuid.UUID, role string) (*SignupInvite, string, error) {
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
		ID:             uuid.New(),
		TokenHash:      tokenHash,
		CreatedBy:      createdBy,
		ExpiresAt:      expiresAt,
		CreatedAt:      now,
		OrganizationID: organizationID,
		Role:           role,
	}
	_, err = s.db.Exec(
		`INSERT INTO signup_invites (id, token_hash, created_by, expires_at, created_at, organization_id, role) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		inv.ID, inv.TokenHash, inv.CreatedBy, inv.ExpiresAt, inv.CreatedAt, uuidPtr(organizationID), inv.Role,
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
	var orgID nullUUID
	err := s.db.QueryRow(
		`SELECT id, token_hash, created_by, expires_at, created_at, used_at, used_by_user_id, organization_id, role FROM signup_invites WHERE token_hash = $1`,
		tokenHash,
	).Scan(&inv.ID, &inv.TokenHash, &inv.CreatedBy, &inv.ExpiresAt, &inv.CreatedAt, &usedAt, &usedByUserID, &orgID, &inv.Role)
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
	if orgID.Valid {
		inv.OrganizationID = orgID.UUID
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
		`SELECT id, token_hash, created_by, expires_at, created_at, used_at, used_by_user_id, organization_id, role FROM signup_invites WHERE created_by = $1 ORDER BY created_at DESC`,
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
		var orgID nullUUID
		if err := rows.Scan(&inv.ID, &inv.TokenHash, &inv.CreatedBy, &inv.ExpiresAt, &inv.CreatedAt, &usedAt, &usedByUserID, &orgID, &inv.Role); err != nil {
			return nil, err
		}
		if usedAt.Valid {
			inv.UsedAt = &usedAt.Time
		}
		if usedByUserID.Valid {
			inv.UsedByUserID = &usedByUserID.UUID
		}
		if orgID.Valid {
			inv.OrganizationID = orgID.UUID
		}
		out = append(out, &inv)
	}
	return out, rows.Err()
}

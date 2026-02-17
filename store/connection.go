package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CloudConnection is a per-organization link to a cloud provider (AWS, Azure, GCP).
// Config holds provider-specific settings (e.g. role ARN, regions); no raw secrets.
// SyncIntervalMinutes: 0 = background sync disabled; 1–1440 = minutes between syncs. Default 5.
// SyncMode: "read_only" = pull only; "read_write" = bi-directional (push allowed).
// ConflictResolution: "cloud" = overwrite with cloud on pull; "ipam" = never overwrite existing IPAM resource on pull.
type CloudConnection struct {
	ID                   uuid.UUID       `json:"id"`
	OrganizationID       uuid.UUID       `json:"organization_id"`
	Provider             string          `json:"provider"` // "aws", "azure", "gcp"
	Name                 string          `json:"name"`
	Config               json.RawMessage `json:"config"`
	CredentialsRef       *string         `json:"credentials_ref,omitempty"`
	SyncIntervalMinutes  int             `json:"sync_interval_minutes"`  // 0 = off; default 5
	SyncMode             string          `json:"sync_mode"`              // "read_only" | "read_write"; default "read_only"
	ConflictResolution   string          `json:"conflict_resolution"`    // "cloud" | "ipam"; default "cloud"
	LastSyncAt           *time.Time      `json:"last_sync_at,omitempty"`
	LastSyncStatus       *string         `json:"last_sync_status,omitempty"`
	LastSyncError        *string         `json:"last_sync_error,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}


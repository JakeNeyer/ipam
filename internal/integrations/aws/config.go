package aws

import (
	"encoding/json"
	"github.com/google/uuid"
)

// AWSConnectionConfig is the config stored in CloudConnection.Config for provider "aws".
type AWSConnectionConfig struct {
	// Region is the AWS region for the EC2/IPAM API (e.g. "us-east-1").
	Region string `json:"region"`
	// IpamScopeId optionally limits sync to pools in this IPAM scope (private or public).
	IpamScopeId string `json:"ipam_scope_id,omitempty"`
	// EnvironmentID is the app environment to attach synced pools (and blocks) to.
	EnvironmentID uuid.UUID `json:"environment_id,omitempty"`
	// SyncPools, SyncBlocks, SyncAllocations control which resource types are synced (pull and push). Nil or true = sync; false = skip. Default all true.
	SyncPools       *bool `json:"sync_pools,omitempty"`
	SyncBlocks      *bool `json:"sync_blocks,omitempty"`
	SyncAllocations *bool `json:"sync_allocations,omitempty"`
}

// ParseAWSConfig parses conn.Config into AWSConnectionConfig. Returns nil if invalid.
func ParseAWSConfig(config []byte) (*AWSConnectionConfig, error) {
	if len(config) == 0 {
		return nil, nil
	}
	var c AWSConnectionConfig
	if err := json.Unmarshal(config, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// SyncResources returns whether to sync pools, blocks, and allocations. Defaults to true when unset.
func (c *AWSConnectionConfig) SyncResources() (pools, blocks, allocations bool) {
	pools = c.SyncPools == nil || *c.SyncPools
	blocks = c.SyncBlocks == nil || *c.SyncBlocks
	allocations = c.SyncAllocations == nil || *c.SyncAllocations
	return pools, blocks, allocations
}

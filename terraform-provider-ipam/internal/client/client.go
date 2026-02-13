package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client talks to the IPAM API with Bearer token authentication.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New creates an IPAM API client. baseURL should be the scheme + host (e.g. https://ipam.example.com).
func New(baseURL, token string, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if token == "" {
		return nil, fmt.Errorf("API token is required")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: baseURL, token: token, httpClient: httpClient}, nil
}

// apiError is the JSON body for API errors.
type apiError struct {
	Error string `json:"error"`
}

func (c *Client) do(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// #nosec G704 -- base URL is from provider config, request path is built from resource IDs
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var ae apiError
		_ = json.Unmarshal(raw, &ae)
		msg := ae.Error
		if msg == "" {
			msg = string(raw)
		}
		return fmt.Errorf("API %s %s: %s", method, path, msg)
	}

	if result != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) get(path string, result interface{}) error {
	return c.do(http.MethodGet, path, nil, result)
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
	return c.do(http.MethodPost, path, body, result)
}

func (c *Client) put(path string, body interface{}, result interface{}) error {
	return c.do(http.MethodPut, path, body, result)
}

func (c *Client) delete(path string) error {
	return c.do(http.MethodDelete, path, nil, nil)
}

// ListEnvironments returns environments with optional name filter and pagination.
func (c *Client) ListEnvironments(name string, limit, offset int) (*EnvListResponse, error) {
	path := "/api/environments?"
	if limit > 0 {
		path += "limit=" + url.QueryEscape(fmt.Sprintf("%d", limit)) + "&"
	}
	if offset > 0 {
		path += "offset=" + url.QueryEscape(fmt.Sprintf("%d", offset)) + "&"
	}
	if name != "" {
		path += "name=" + url.QueryEscape(name)
	}
	var out EnvListResponse
	if err := c.get(strings.TrimSuffix(path, "&"), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetEnvironment returns a single environment by ID (includes blocks).
func (c *Client) GetEnvironment(id string) (*EnvDetailResponse, error) {
	var out EnvDetailResponse
	if err := c.get("/api/environments/"+url.PathEscape(id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PoolInput is one pool when creating an environment.
type PoolInput struct {
	Name string
	CIDR string
}

// CreateEnvironment creates an environment with one or more pools.
func (c *Client) CreateEnvironment(name string, pools []PoolInput) (*EnvResponse, error) {
	if len(pools) == 0 {
		return nil, fmt.Errorf("at least one pool is required")
	}
	poolMaps := make([]map[string]string, 0, len(pools))
	for _, p := range pools {
		if p.Name == "" || p.CIDR == "" {
			return nil, fmt.Errorf("each pool must have name and CIDR")
		}
		poolMaps = append(poolMaps, map[string]string{"name": p.Name, "cidr": p.CIDR})
	}
	body := map[string]interface{}{
		"name":  name,
		"pools": poolMaps,
	}
	var out EnvResponse
	if err := c.post("/api/environments", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateEnvironment updates an environment.
func (c *Client) UpdateEnvironment(id, name string) (*EnvResponse, error) {
	var out EnvResponse
	if err := c.put("/api/environments/"+url.PathEscape(id), map[string]string{"name": name}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteEnvironment deletes an environment.
func (c *Client) DeleteEnvironment(id string) error {
	return c.delete("/api/environments/" + url.PathEscape(id))
}

// ListBlocks returns blocks with optional filters.
func (c *Client) ListBlocks(name, environmentID string, orphanedOnly bool, limit, offset int) (*BlockListResponse, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}
	if name != "" {
		params.Set("name", name)
	}
	if environmentID != "" {
		params.Set("environment_id", environmentID)
	}
	if orphanedOnly {
		params.Set("orphaned_only", "true")
	}
	path := "/api/blocks"
	if q := params.Encode(); q != "" {
		path += "?" + q
	}
	var out BlockListResponse
	if err := c.get(path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetBlock returns a single block by ID.
func (c *Client) GetBlock(id string) (*BlockResponse, error) {
	var out BlockResponse
	if err := c.get("/api/blocks/"+url.PathEscape(id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateBlock creates a network block.
func (c *Client) CreateBlock(name, cidr, environmentID string, poolID *string) (*BlockResponse, error) {
	body := map[string]interface{}{"name": name, "cidr": cidr}
	if environmentID != "" {
		body["environment_id"] = environmentID
	}
	if poolID != nil && *poolID != "" {
		body["pool_id"] = *poolID
	}
	var out BlockResponse
	if err := c.post("/api/blocks", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateBlock updates a block.
func (c *Client) UpdateBlock(id, name string, environmentID, poolID *string) (*BlockResponse, error) {
	body := map[string]interface{}{"name": name}
	if environmentID != nil {
		body["environment_id"] = *environmentID
	}
	if poolID != nil {
		body["pool_id"] = *poolID
	}
	var out BlockResponse
	if err := c.put("/api/blocks/"+url.PathEscape(id), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteBlock deletes a block.
func (c *Client) DeleteBlock(id string) error {
	return c.delete("/api/blocks/" + url.PathEscape(id))
}

// CreatePool creates an environment pool.
func (c *Client) CreatePool(environmentID, name, cidr string) (*PoolResponse, error) {
	body := map[string]string{"environment_id": environmentID, "name": name, "cidr": cidr}
	var out PoolResponse
	if err := c.post("/api/pools", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPool returns a pool by ID.
func (c *Client) GetPool(id string) (*PoolResponse, error) {
	var out PoolResponse
	if err := c.get("/api/pools/"+url.PathEscape(id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListPools returns pools for an environment.
func (c *Client) ListPools(environmentID string) (*PoolListResponse, error) {
	path := "/api/pools?environment_id=" + url.QueryEscape(environmentID)
	var out PoolListResponse
	if err := c.get(path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdatePool updates a pool.
func (c *Client) UpdatePool(id, name, cidr string) (*PoolResponse, error) {
	body := map[string]string{"name": name, "cidr": cidr}
	var out PoolResponse
	if err := c.put("/api/pools/"+url.PathEscape(id), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeletePool deletes a pool.
func (c *Client) DeletePool(id string) error {
	return c.delete("/api/pools/" + url.PathEscape(id))
}

// ListAllocations returns allocations with optional filters.
func (c *Client) ListAllocations(name, blockName string, limit, offset int) (*AllocationListResponse, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}
	if name != "" {
		params.Set("name", name)
	}
	if blockName != "" {
		params.Set("block_name", blockName)
	}
	path := "/api/allocations"
	if q := params.Encode(); q != "" {
		path += "?" + q
	}
	var out AllocationListResponse
	if err := c.get(path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAllocation returns a single allocation by ID.
func (c *Client) GetAllocation(id string) (*AllocationResponse, error) {
	var out AllocationResponse
	if err := c.get("/api/allocations/"+url.PathEscape(id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateAllocation creates an allocation.
func (c *Client) CreateAllocation(name, blockName, cidr string) (*AllocationResponse, error) {
	body := map[string]string{"name": name, "block_name": blockName, "cidr": cidr}
	var out AllocationResponse
	if err := c.post("/api/allocations", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AutoAllocate finds the next available CIDR in a block using bin-packing and creates an allocation.
func (c *Client) AutoAllocate(name, blockName string, prefixLength int) (*AllocationResponse, error) {
	body := map[string]interface{}{"name": name, "block_name": blockName, "prefix_length": prefixLength}
	var out AllocationResponse
	if err := c.post("/api/allocations/auto", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateAllocation updates an allocation (name only).
func (c *Client) UpdateAllocation(id, name string) (*AllocationResponse, error) {
	body := map[string]string{"name": name}
	var out AllocationResponse
	if err := c.put("/api/allocations/"+url.PathEscape(id), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteAllocation deletes an allocation.
func (c *Client) DeleteAllocation(id string) error {
	return c.delete("/api/allocations/" + url.PathEscape(id))
}

// ListReservedBlocks returns all reserved blocks (admin only).
func (c *Client) ListReservedBlocks() (*ReservedBlockListResponse, error) {
	var out ReservedBlockListResponse
	if err := c.get("/api/reserved-blocks", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateReservedBlock creates a reserved block (admin only).
func (c *Client) CreateReservedBlock(name, cidr, reason string) (*ReservedBlockResponse, error) {
	body := map[string]string{"name": name, "cidr": cidr, "reason": reason}
	var out ReservedBlockResponse
	if err := c.post("/api/reserved-blocks", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteReservedBlock deletes a reserved block (admin only).
func (c *Client) DeleteReservedBlock(id string) error {
	return c.delete("/api/reserved-blocks/" + url.PathEscape(id))
}

// API response types (match server JSON; use json tags for lowercase).

type EnvResponse struct {
	Id            string   `json:"id"`
	Name          string   `json:"name"`
	InitialPoolID string   `json:"initial_pool_id,omitempty"`
	PoolIDs       []string `json:"pool_ids,omitempty"`
}

type EnvListResponse struct {
	Environments []EnvResponse `json:"environments"`
	Total        int          `json:"total"`
}

type BlockRef struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CIDR           string `json:"cidr"`
	TotalIPs       string `json:"total_ips"`   // derive-only; string supports IPv6 /64 etc.
	UsedIPs        string `json:"used_ips"`
	Available      string `json:"available_ips"`
	EnvironmentID  string `json:"environment_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"` // for orphan blocks
}

type EnvDetailResponse struct {
	Id     string     `json:"id"`
	Name   string     `json:"name"`
	Blocks []BlockRef `json:"blocks"`
}

type BlockResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	CIDR           string  `json:"cidr"`
	TotalIPs       string  `json:"total_ips"`   // derive-only; string supports IPv6 /64 etc.
	UsedIPs        string  `json:"used_ips"`
	Available      string  `json:"available_ips"`
	EnvironmentID  string  `json:"environment_id,omitempty"`
	OrganizationID string  `json:"organization_id,omitempty"` // for orphan blocks
	PoolID         *string `json:"pool_id,omitempty"`
}

type BlockListResponse struct {
	Blocks []BlockResponse `json:"blocks"`
	Total  int             `json:"total"`
}

type PoolResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	EnvironmentID  string `json:"environment_id"`
	Name           string `json:"name"`
	CIDR           string `json:"cidr"`
}

type PoolListResponse struct {
	Pools []PoolResponse `json:"pools"`
}

type AllocationResponse struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	BlockName string `json:"block_name"`
	CIDR      string `json:"cidr"`
}

type AllocationListResponse struct {
	Allocations []AllocationResponse `json:"allocations"`
	Total      int                  `json:"total"`
}

type ReservedBlockResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CIDR      string `json:"cidr"`
	Reason    string `json:"reason,omitempty"`
	CreatedAt string `json:"created_at"`
}

type ReservedBlockListResponse struct {
	ReservedBlocks []ReservedBlockResponse `json:"reserved_blocks"`
}

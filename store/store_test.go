package store

import (
	"strings"
	"testing"
	"time"

	"github.com/JakeNeyer/ipam/network"
	"github.com/google/uuid"
)

// TestCreateReservedBlock tests CreateReservedBlock with table-driven cases.
func TestCreateReservedBlock(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		reason  string
		wantErr bool
	}{
		{
			name:    "valid CIDR with reason",
			cidr:    "10.0.0.0/24",
			reason:  "dmz",
			wantErr: false,
		},
		{
			name:    "valid CIDR no reason",
			cidr:    "192.168.0.0/16",
			reason:  "",
			wantErr: false,
		},
		{
			name:    "valid IPv6 CIDR",
			cidr:    "2001:db8::/32",
			reason:  "v6",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			r := &ReservedBlock{CIDR: tt.cidr, Reason: tt.reason}
			err := s.CreateReservedBlock(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateReservedBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if r.ID == uuid.Nil {
					t.Error("CreateReservedBlock() did not set ID")
				}
				if r.CreatedAt.IsZero() {
					t.Error("CreateReservedBlock() did not set CreatedAt")
				}
				list, _ := s.ListReservedBlocks(nil)
				if len(list) != 1 {
					t.Errorf("ListReservedBlocks() len = %v, want 1", len(list))
				}
			}
		})
	}
}

// TestListReservedBlocks tests ListReservedBlocks with table-driven cases.
func TestListReservedBlocks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Store)
		wantCount   int
		wantFirstCIDR string
	}{
		{
			name:        "empty store",
			setup:       func(s *Store) {},
			wantCount:   0,
			wantFirstCIDR: "",
		},
		{
			name: "one reserved block",
			setup: func(s *Store) {
				_ = s.CreateReservedBlock(&ReservedBlock{CIDR: "10.0.0.0/24", Reason: "a"})
			},
			wantCount:   1,
			wantFirstCIDR: "10.0.0.0/24",
		},
		{
			name: "multiple reserved blocks sorted by CIDR",
			setup: func(s *Store) {
				_ = s.CreateReservedBlock(&ReservedBlock{CIDR: "10.0.0.0/24"})
				_ = s.CreateReservedBlock(&ReservedBlock{CIDR: "10.0.0.0/16"})
				_ = s.CreateReservedBlock(&ReservedBlock{CIDR: "192.168.0.0/24"})
			},
			wantCount:   3,
			wantFirstCIDR: "10.0.0.0/16",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			tt.setup(s)
			list, err := s.ListReservedBlocks(nil)
			if err != nil {
				t.Errorf("ListReservedBlocks() error = %v", err)
				return
			}
			if len(list) != tt.wantCount {
				t.Errorf("ListReservedBlocks() len = %v, want %v", len(list), tt.wantCount)
			}
			if tt.wantFirstCIDR != "" && len(list) > 0 && list[0].CIDR != tt.wantFirstCIDR {
				t.Errorf("ListReservedBlocks()[0].CIDR = %v, want %v", list[0].CIDR, tt.wantFirstCIDR)
			}
		})
	}
}

// TestGetReservedBlock tests GetReservedBlock with table-driven cases.
func TestGetReservedBlock(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Store) uuid.UUID
		id        uuid.UUID
		wantCIDR  string
		wantErr   bool
	}{
		{
			name:     "not found",
			setup:    func(s *Store) uuid.UUID { return uuid.Nil },
			id:       uuid.New(),
			wantCIDR: "",
			wantErr:  true,
		},
		{
			name: "found",
			setup: func(s *Store) uuid.UUID {
				r := &ReservedBlock{CIDR: "10.0.0.0/24"}
				_ = s.CreateReservedBlock(r)
				return r.ID
			},
			id:       uuid.Nil,
			wantCIDR: "10.0.0.0/24",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			createdID := tt.setup(s)
			id := tt.id
			if createdID != uuid.Nil {
				id = createdID
			}
			got, err := s.GetReservedBlock(id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReservedBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.CIDR != tt.wantCIDR {
				t.Errorf("GetReservedBlock().CIDR = %v, want %v", got.CIDR, tt.wantCIDR)
			}
		})
	}
}

// TestDeleteReservedBlock tests DeleteReservedBlock with table-driven cases.
func TestDeleteReservedBlock(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Store) uuid.UUID
		wantErr bool
	}{
		{
			name:    "not found",
			setup:   func(s *Store) uuid.UUID { return uuid.New() },
			wantErr: true,
		},
		{
			name: "deleted",
			setup: func(s *Store) uuid.UUID {
				r := &ReservedBlock{CIDR: "10.0.0.0/24"}
				_ = s.CreateReservedBlock(r)
				return r.ID
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			id := tt.setup(s)
			err := s.DeleteReservedBlock(id)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteReservedBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				_, err := s.GetReservedBlock(id)
				if err == nil {
					t.Error("GetReservedBlock() after delete expected error")
				}
			}
		})
	}
}

// TestOverlapsReservedBlock tests OverlapsReservedBlock with table-driven cases.
func TestOverlapsReservedBlock(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Store)
		cidr        string
		wantOverlap bool
		wantCIDR    string
		wantErr     bool
	}{
		{
			name:        "no reserved blocks",
			setup:       func(s *Store) {},
			cidr:        "10.0.0.0/24",
			wantOverlap: false,
			wantCIDR:    "",
			wantErr:     false,
		},
		{
			name: "overlaps exact",
			setup: func(s *Store) {
				_ = s.CreateReservedBlock(&ReservedBlock{CIDR: "10.0.0.0/24"})
			},
			cidr:        "10.0.0.0/24",
			wantOverlap: true,
			wantCIDR:    "10.0.0.0/24",
			wantErr:     false,
		},
		{
			name: "overlaps subnet",
			setup: func(s *Store) {
				_ = s.CreateReservedBlock(&ReservedBlock{CIDR: "10.0.0.0/16"})
			},
			cidr:        "10.0.1.0/24",
			wantOverlap: true,
			wantCIDR:    "10.0.0.0/16",
			wantErr:     false,
		},
		{
			name: "no overlap",
			setup: func(s *Store) {
				_ = s.CreateReservedBlock(&ReservedBlock{CIDR: "10.0.0.0/24"})
			},
			cidr:        "10.0.1.0/24",
			wantOverlap: false,
			wantCIDR:    "",
			wantErr:     false,
		},
		{
			name: "invalid CIDR returns error when checking reserved",
			setup: func(s *Store) {
				_ = s.CreateReservedBlock(&ReservedBlock{CIDR: "10.0.0.0/24"})
			},
			cidr:        "invalid",
			wantOverlap: false,
			wantCIDR:    "",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			tt.setup(s)
			got, err := s.OverlapsReservedBlock(tt.cidr, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("OverlapsReservedBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if (got != nil) != tt.wantOverlap {
				t.Errorf("OverlapsReservedBlock() overlap = %v, want %v", got != nil, tt.wantOverlap)
				return
			}
			if tt.wantOverlap && got != nil && got.CIDR != tt.wantCIDR {
				t.Errorf("OverlapsReservedBlock().CIDR = %v, want %v", got.CIDR, tt.wantCIDR)
			}
		})
	}
}

// TestCreateEnvironment tests CreateEnvironment and GetEnvironment with table-driven cases.
func TestCreateEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		env     *network.Environment
		wantErr bool
	}{
		{
			name:    "valid",
			env:     &network.Environment{Id: uuid.New(), Name: "prod"},
			wantErr: false,
		},
		{
			name:    "valid with nil blocks",
			env:     &network.Environment{Id: uuid.New(), Name: "staging", Block: nil},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			err := s.CreateEnvironment(tt.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateEnvironment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				got, err := s.GetEnvironment(tt.env.Id)
				if err != nil {
					t.Errorf("GetEnvironment() error = %v", err)
					return
				}
				if got.Name != tt.env.Name {
					t.Errorf("GetEnvironment().Name = %v, want %v", got.Name, tt.env.Name)
				}
			}
		})
	}
}

// TestGetEnvironment tests GetEnvironment with table-driven cases.
func TestGetEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Store) uuid.UUID
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "not found",
			setup:   func(s *Store) uuid.UUID { return uuid.Nil },
			id:      uuid.New(),
			wantErr: true,
		},
		{
			name: "found",
			setup: func(s *Store) uuid.UUID {
				env := &network.Environment{Id: uuid.New(), Name: "prod"}
				_ = s.CreateEnvironment(env)
				return env.Id
			},
			id:      uuid.Nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			createdID := tt.setup(s)
			id := tt.id
			if createdID != uuid.Nil {
				id = createdID
			}
			_, err := s.GetEnvironment(id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEnvironment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestListEnvironmentsFiltered tests ListEnvironmentsFiltered with table-driven cases.
func TestListEnvironmentsFiltered(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Store)
		nameFilter string
		limit     int
		offset    int
		wantLen   int
		wantTotal int
	}{
		{
			name:      "empty",
			setup:     func(s *Store) {},
			nameFilter: "",
			limit:     10,
			offset:    0,
			wantLen:   0,
			wantTotal: 0,
		},
		{
			name: "filter by name",
			setup: func(s *Store) {
				_ = s.CreateEnvironment(&network.Environment{Id: uuid.New(), Name: "production"})
				_ = s.CreateEnvironment(&network.Environment{Id: uuid.New(), Name: "staging"})
				_ = s.CreateEnvironment(&network.Environment{Id: uuid.New(), Name: "prod-test"})
			},
			nameFilter: "prod",
			limit:      10,
			offset:     0,
			wantLen:    2,
			wantTotal:  2,
		},
		{
			name: "limit and offset",
			setup: func(s *Store) {
				for i := 0; i < 5; i++ {
					_ = s.CreateEnvironment(&network.Environment{Id: uuid.New(), Name: "env" + string(rune('a'+i))})
				}
			},
			nameFilter: "",
			limit:     2,
			offset:    1,
			wantLen:   2,
			wantTotal: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			tt.setup(s)
			list, total, err := s.ListEnvironmentsFiltered(tt.nameFilter, nil, tt.limit, tt.offset)
			if err != nil {
				t.Errorf("ListEnvironmentsFiltered() error = %v", err)
				return
			}
			if len(list) != tt.wantLen {
				t.Errorf("ListEnvironmentsFiltered() len = %v, want %v", len(list), tt.wantLen)
			}
			if total != tt.wantTotal {
				t.Errorf("ListEnvironmentsFiltered() total = %v, want %v", total, tt.wantTotal)
			}
		})
	}
}

// TestCreateUser tests CreateUser with table-driven cases.
func TestCreateUser(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		role     string
		wantErr  bool
		errContains string
	}{
		{
			name:     "valid admin",
			email:    "admin@example.com",
			role:     RoleAdmin,
			wantErr:  false,
			errContains: "",
		},
		{
			name:     "valid user",
			email:    "user@example.com",
			role:     RoleUser,
			wantErr:  false,
			errContains: "",
		},
		{
			name:     "duplicate email",
			email:    "dup@example.com",
			role:     RoleUser,
			wantErr:  true,
			errContains: "already exists",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			if tt.name == "duplicate email" {
				u := &User{Email: "dup@example.com", PasswordHash: "h", Role: RoleUser}
				_ = s.CreateUser(u)
			}
			u := &User{Email: tt.email, PasswordHash: "hash", Role: tt.role}
			err := s.CreateUser(u)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" && err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("CreateUser() error = %v, want substring %q", err, tt.errContains)
			}
			if !tt.wantErr && u.ID == uuid.Nil {
				t.Error("CreateUser() did not set ID")
			}
		})
	}
}

// TestGetUserByEmail tests GetUserByEmail with table-driven cases.
func TestGetUserByEmail(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Store)
		email     string
		wantRole  string
		wantErr   bool
	}{
		{
			name:     "not found",
			setup:    func(s *Store) {},
			email:    "nobody@example.com",
			wantRole: "",
			wantErr:  true,
		},
		{
			name: "found case insensitive",
			setup: func(s *Store) {
				u := &User{Email: "Admin@Example.com", PasswordHash: "h", Role: RoleAdmin}
				_ = s.CreateUser(u)
			},
			email:    "admin@example.com",
			wantRole: RoleAdmin,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			tt.setup(s)
			got, err := s.GetUserByEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Role != tt.wantRole {
				t.Errorf("GetUserByEmail().Role = %v, want %v", got.Role, tt.wantRole)
			}
		})
	}
}

// TestGetUserByTokenHash tests GetUserByTokenHash with table-driven cases (valid, not found, expired).
func TestGetUserByTokenHash(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	tests := []struct {
		name      string
		setup     func(*Store) string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "token not found",
			setup: func(s *Store) string {
				u := &User{Email: "u@x.com", PasswordHash: "h", Role: RoleUser}
				_ = s.CreateUser(u)
				return "nonexistent-hash"
			},
			wantErr:   true,
			errSubstr: "token not found",
		},
		{
			name: "valid token returns user",
			setup: func(s *Store) string {
				u := &User{Email: "u@x.com", PasswordHash: "h", Role: RoleUser}
				_ = s.CreateUser(u)
				tok, raw, _ := s.CreateAPIToken(u.ID, "t", nil, nil)
				_ = tok
				h := hashToken(raw)
				return h
			},
			wantErr:   false,
			errSubstr: "",
		},
		{
			name: "expired token returns error",
			setup: func(s *Store) string {
				u := &User{Email: "u@x.com", PasswordHash: "h", Role: RoleUser}
				_ = s.CreateUser(u)
				tok, raw, _ := s.CreateAPIToken(u.ID, "t", &past, nil)
				_ = tok
				return hashToken(raw)
			},
			wantErr:   true,
			errSubstr: "token expired",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			keyHash := tt.setup(s)
			got, err := s.GetUserByTokenHash(keyHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByTokenHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errSubstr != "" && (err == nil || !strings.Contains(err.Error(), tt.errSubstr)) {
				t.Errorf("GetUserByTokenHash() error = %v, want substring %q", err, tt.errSubstr)
			}
			if !tt.wantErr && got == nil {
				t.Error("GetUserByTokenHash() expected user")
			}
		})
	}
}

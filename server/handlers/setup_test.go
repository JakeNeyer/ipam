package handlers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
)

type listUsersErrorStore struct {
	store.Storer
	err error
}

func (s listUsersErrorStore) ListUsers(*uuid.UUID) ([]*store.User, error) {
	return nil, s.err
}

func TestGetSetupStatus_EmptyStoreRequiresSetup(t *testing.T) {
	u := NewGetSetupStatusUseCase(store.NewStore(), nil)
	var out getSetupStatusOutput
	if err := u.Interact(context.Background(), struct{}{}, &out); err != nil {
		t.Fatalf("Interact: %v", err)
	}
	if !out.SetupRequired {
		t.Fatal("setup_required = false, want true when no users exist")
	}
}

func TestGetSetupStatus_ExistingUserDoesNotRequireSetup(t *testing.T) {
	s := store.NewStore()
	if err := s.CreateUser(&store.User{Email: "admin@example.com", Role: store.RoleAdmin}); err != nil {
		t.Fatal(err)
	}
	u := NewGetSetupStatusUseCase(s, nil)
	var out getSetupStatusOutput
	if err := u.Interact(context.Background(), struct{}{}, &out); err != nil {
		t.Fatalf("Interact: %v", err)
	}
	if out.SetupRequired {
		t.Fatal("setup_required = true, want false when a user exists")
	}
}

func TestGetSetupStatus_InitialAdminEmailDoesNotHideEmptyStore(t *testing.T) {
	t.Setenv("INITIAL_ADMIN_EMAIL", "admin@example.com")
	u := NewGetSetupStatusUseCase(store.NewStore(), nil)
	var out getSetupStatusOutput
	if err := u.Interact(context.Background(), struct{}{}, &out); err != nil {
		t.Fatalf("Interact: %v", err)
	}
	if !out.SetupRequired {
		t.Fatal("setup_required = false, want true when no users exist even if INITIAL_ADMIN_EMAIL is set")
	}
}

func TestGetSetupStatus_ListUsersError(t *testing.T) {
	u := NewGetSetupStatusUseCase(listUsersErrorStore{Storer: store.NewStore(), err: errors.New("connection refused")}, nil)
	var out getSetupStatusOutput
	err := u.Interact(context.Background(), struct{}{}, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), setupDBUnavailableMsg) {
		t.Fatalf("error = %v, want %q", err, setupDBUnavailableMsg)
	}
}

func TestPostSetup_ListUsersError(t *testing.T) {
	u := NewPostSetupUseCase(listUsersErrorStore{Storer: store.NewStore(), err: errors.New("connection refused")}, nil)
	var out postSetupOutput
	err := u.Interact(context.Background(), postSetupInput{Email: "admin@example.com", Password: "password123"}, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), setupDBUnavailableMsg) {
		t.Fatalf("error = %v, want %q", err, setupDBUnavailableMsg)
	}
}

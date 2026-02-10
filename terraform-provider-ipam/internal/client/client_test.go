package client

import (
	"testing"
)

func TestNew(t *testing.T) {
	_, err := New("", "token", nil)
	if err == nil {
		t.Error("expected error when baseURL is empty")
	}
	_, err = New("http://localhost", "", nil)
	if err == nil {
		t.Error("expected error when token is empty")
	}
	c, err := New("http://localhost:8080", "secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.baseURL != "http://localhost:8080" {
		t.Errorf("baseURL: got %s", c.baseURL)
	}
	if c.token != "secret" {
		t.Errorf("token: got %s", c.token)
	}
}

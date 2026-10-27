package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JakeNeyer/ipam/server/config"
	"golang.org/x/oauth2"
)

func TestNewProviderRegistry_Empty(t *testing.T) {
	r, err := NewProviderRegistry(nil)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	if r == nil {
		t.Fatal("NewProviderRegistry() = nil")
	}
	_, ok := r.Endpoint("sso")
	if ok {
		t.Error("Endpoint(sso) = true for empty registry")
	}
}

func TestNewProviderRegistry_FromConfig(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": {
			ClientID: "c", ClientSecret: "s",
			AuthURL: "https://idp/auth", TokenURL: "https://idp/token", UserInfoURL: "https://idp/userinfo",
		},
	}}}
	r, err := NewProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	e, ok := r.Endpoint("sso")
	if !ok {
		t.Fatal("Endpoint(\"sso\") = false")
	}
	if e.AuthURL != "https://idp/auth" || e.TokenURL != "https://idp/token" {
		t.Errorf("Endpoint(\"sso\") = %+v", e)
	}
}

func TestProviderRegistry_UserInfo_UnknownProvider(t *testing.T) {
	r, err := NewProviderRegistry(nil)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	profile, err := r.UserInfo(context.Background(), "unknown", &oauth2.Token{})
	if err != nil {
		t.Errorf("UserInfo(unknown) err = %v", err)
	}
	if profile != nil {
		t.Errorf("UserInfo(unknown) = %+v", profile)
	}
}

func TestProviderRegistry_Register(t *testing.T) {
	r := &ProviderRegistry{
		endpoints: make(map[string]oauth2.Endpoint),
		userInfos: make(map[string]UserInfoFetcher),
		clients:   make(map[string]*http.Client),
	}
	e := oauth2.Endpoint{AuthURL: "https://auth.example.com", TokenURL: "https://token.example.com"}
	r.Register("custom", e, nil, nil)
	got, ok := r.Endpoint("custom")
	if !ok {
		t.Fatal("Endpoint(\"custom\") = false")
	}
	if got.AuthURL != e.AuthURL || got.TokenURL != e.TokenURL {
		t.Errorf("Endpoint(\"custom\") = %+v", got)
	}
}

func TestNewProviderRegistry_BuildsHTTPClientPerProvider(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": {
			ClientID:     "c",
			ClientSecret: "s",
			AuthURL:      "https://idp/auth",
			TokenURL:     "https://idp/token",
			UserInfoURL:  "https://idp/userinfo",
		},
	}}}

	if !cfg.OAuth.Providers["sso"].Enabled() {
		t.Fatal("test provider is not enabled")
	}

	r, err := NewProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}

	client := r.HTTPClient("sso")
	if client == nil {
		t.Fatal("HTTPClient(\"sso\") = nil")
	}

	if client.Timeout != 15*time.Second {
		t.Errorf("HTTPClient(\"sso\").Timeout = %v, want %v", client.Timeout, 15*time.Second)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("HTTPClient(\"sso\").Transport = %T, want *http.Transport", client.Transport)
	}

	if transport == http.DefaultTransport {
		t.Error("HTTPClient(\"sso\").Transport = http.DefaultTransport, want cloned transport")
	}

	if transport.Proxy == nil {
		t.Error("HTTPClient(\"sso\").Transport.Proxy = nil, want default proxy behavior")
	}
}

func TestNewProviderRegistry_BuildsMTLSConfig(t *testing.T) {
	certFile, keyFile := writeTestCertKeyPair(t)

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": {
			ClientID:     "c",
			ClientSecret: "s",
			AuthURL:      "https://idp/auth",
			TokenURL:     "https://idp/token",
			UserInfoURL:  "https://idp/userinfo",
			ClientMTLS: config.OAuthClientMTLSConfig{
				Enabled:     true,
				TLSCertFile: certFile,
				TLSKeyFile:  keyFile,
				TLSVersion:  tls.VersionTLS12,
			},
		},
	}}}

	if !cfg.OAuth.Providers["sso"].Enabled() {
		t.Fatal("test provider is not enabled")
	}

	r, err := NewProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}

	client := r.HTTPClient("sso")
	if client == nil {
		t.Fatal("HTTPClient(\"sso\") = nil")
	}

	if client.Timeout != 15*time.Second {
		t.Errorf("HTTPClient(\"sso\").Timeout = %v, want %v", client.Timeout, 15*time.Second)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("HTTPClient(\"sso\").Transport = %T, want *http.Transport", client.Transport)
	}

	if transport == http.DefaultTransport {
		t.Error("HTTPClient(\"sso\").Transport = http.DefaultTransport, want cloned transport")
	}

	if transport.Proxy == nil {
		t.Error("HTTPClient(\"sso\").Transport.Proxy = nil, want default proxy behavior")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("HTTPClient(\"sso\").Transport.TLSClientConfig = nil, want configured TLS client config")
	}

	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("TLSClientConfig.MinVersion = %v, want %v", transport.TLSClientConfig.MinVersion, tls.VersionTLS12)
	}

	if len(transport.TLSClientConfig.Certificates) != 1 {
		t.Fatalf("len(TLSClientConfig.Certificates) = %d, want 1", len(transport.TLSClientConfig.Certificates))
	}

	if len(transport.TLSClientConfig.Certificates[0].Certificate) == 0 {
		t.Fatal("TLSClientConfig.Certificates[0].Certificate is empty")
	}
}

func TestNewProviderRegistry_UnknownMTLSCertFile(t *testing.T) {
	_, keyFile := writeTestCertKeyPair(t)

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": {
			ClientID:     "c",
			ClientSecret: "s",
			AuthURL:      "https://idp/auth",
			TokenURL:     "https://idp/token",
			UserInfoURL:  "https://idp/userinfo",
			ClientMTLS: config.OAuthClientMTLSConfig{
				Enabled:     true,
				TLSCertFile: "",
				TLSKeyFile:  keyFile,
				TLSVersion:  tls.VersionTLS12,
			},
		},
	}}}

	if !cfg.OAuth.Providers["sso"].Enabled() {
		t.Fatal("test provider is not enabled")
	}

	_, err := NewProviderRegistry(cfg)
	if err == nil {
		t.Fatal("NewProviderRegistry() should have failed")
	}

	if !strings.Contains(err.Error(), "TLS cert file is required") {
		t.Fatalf("NewProviderRegistry() error = %v, want missing TLS cert file error", err)
	}
}

func TestNewProviderRegistry_UnknownMTLSkeyFile(t *testing.T) {
	certFile, _ := writeTestCertKeyPair(t)

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": {
			ClientID:     "c",
			ClientSecret: "s",
			AuthURL:      "https://idp/auth",
			TokenURL:     "https://idp/token",
			UserInfoURL:  "https://idp/userinfo",
			ClientMTLS: config.OAuthClientMTLSConfig{
				Enabled:     true,
				TLSCertFile: certFile,
				TLSKeyFile:  "",
				TLSVersion:  tls.VersionTLS12,
			},
		},
	}}}

	if !cfg.OAuth.Providers["sso"].Enabled() {
		t.Fatal("test provider is not enabled")
	}

	_, err := NewProviderRegistry(cfg)
	if err == nil {
		t.Fatal("NewProviderRegistry() should have failed")
	}

	if !strings.Contains(err.Error(), "TLS key file is required") {
		t.Fatalf("NewProviderRegistry() error = %v, want missing TLS key file error", err)
	}
}

func writeTestCertKeyPair(t *testing.T) (string, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-client",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("x509.CreateCertificate() error = %v", err)
	}

	dir := t.TempDir()
	certFile := filepath.Join(dir, "tls.crt")
	keyFile := filepath.Join(dir, "tls.key")

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		t.Fatalf("os.WriteFile(certFile) error = %v", err)
	}

	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		t.Fatalf("os.WriteFile(keyFile) error = %v", err)
	}

	return certFile, keyFile
}

package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestRateLimiterBlocksFloodAndCleansUp(t *testing.T) {
	rl := NewRateLimiter(1, 3)
	defer rl.Close()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 3 burst requests from the same IP succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i+1, rec.Code)
		}
	}

	// 4th immediate request exceeds burst and must be rate-limited with 429
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("flood request status = %d, want 429 Too Many Requests", rec.Code)
	}
}

func TestEnsureTLSCertificateAndCipherSuites(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	if err := EnsureTLSCertificate(certFile, keyFile); err != nil {
		t.Fatalf("EnsureTLSCertificate error: %v", err)
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("failed to load generated X509 key pair: %v", err)
	}

	tlsCfg := NewSecureTLSConfig()
	tlsCfg.Certificates = []tls.Certificate{cert}
	if tlsCfg.MinVersion != tls.VersionTLS12 || len(tlsCfg.CipherSuites) == 0 {
		t.Fatal("expected TLS 1.2+ minimum version and explicit cipher suites")
	}

	srv := httptest.NewUnstartedServer(SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	srv.TLS = tlsCfg
	srv.StartTLS()
	defer srv.Close()

	client := srv.Client()
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("HTTPS GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HTTPS status = %d, want 200", resp.StatusCode)
	}
	if resp.Header.Get("Strict-Transport-Security") == "" {
		t.Fatal("expected Strict-Transport-Security header over HTTPS")
	}
}

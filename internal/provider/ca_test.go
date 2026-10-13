package provider

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tlsServer starts an HTTPS server with a certificate no system root trusts and writes its
// certificate to a PEM file, returning the server and the file path.
func tlsServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	file := filepath.Join(t.TempDir(), "ca.pem")
	block := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(file, block, 0o600); err != nil {
		t.Fatal(err)
	}
	return srv, file
}

func get(c *http.Client, url string) error {
	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func TestNewHTTPClient_whenCAFileSet(t *testing.T) {
	// Given a server whose certificate is in a custom CA bundle.
	srv, file := tlsServer(t)
	// When building the client with that bundle.
	c, err := NewHTTPClient(file)
	if err != nil {
		t.Fatal(err)
	}
	// Then the server certificate is trusted.
	if err := get(c, srv.URL); err != nil {
		t.Fatalf("GET with custom CA: %v", err)
	}
}

func TestNewHTTPClient_whenCAFileUnset(t *testing.T) {
	// Given the same kind of server and no custom CA bundle.
	srv, _ := tlsServer(t)
	// When building the default client.
	c, err := NewHTTPClient("")
	if err != nil {
		t.Fatal(err)
	}
	// Then the certificate is rejected.
	if err := get(c, srv.URL); err == nil {
		t.Fatal("GET without custom CA succeeded, want certificate error")
	}
}

func TestNewHTTPClient_whenCAFileHasNoCertificates(t *testing.T) {
	// Given a file that contains no PEM certificate.
	file := filepath.Join(t.TempDir(), "empty.pem")
	if err := os.WriteFile(file, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	// When building the client.
	_, err := NewHTTPClient(file)
	// Then configuration fails naming the variable.
	if err == nil || !strings.Contains(err.Error(), "WAYFINDER_CA_FILE") {
		t.Fatalf("err = %v, want error naming WAYFINDER_CA_FILE", err)
	}
}

func TestNewHTTPClient_whenCAFileMissing(t *testing.T) {
	// Given a path that does not exist.
	_, err := NewHTTPClient(filepath.Join(t.TempDir(), "missing.pem"))
	// Then configuration fails.
	if err == nil || !strings.Contains(err.Error(), "WAYFINDER_CA_FILE") {
		t.Fatalf("err = %v, want error naming WAYFINDER_CA_FILE", err)
	}
}

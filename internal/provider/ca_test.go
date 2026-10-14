package provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestNewHTTPClient_whenProfileCASet(t *testing.T) {
	// Given a server whose certificate is only in the wf CLI profile's CA PEM.
	srv, file := tlsServer(t)
	pemBytes, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	// When building the client with that PEM and no CA file.
	c, err := newHTTPClient("", pemBytes)
	if err != nil {
		t.Fatal(err)
	}
	// Then the server certificate is trusted.
	if err := get(c, srv.URL); err != nil {
		t.Fatalf("GET with profile CA: %v", err)
	}
}

// distinctTLSServer starts an HTTPS server with its own freshly generated self-signed
// certificate (httptest servers otherwise share one built-in certificate) and returns the
// server and that certificate as PEM.
func distinctTLSServer(t *testing.T) (*httptest.Server, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "wayfinder-test-" + serial.String()},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, MinVersion: tls.VersionTLS12}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestNewHTTPClient_whenCAFileAndProfileCASet(t *testing.T) {
	// Given two servers with distinct certificates: one in WAYFINDER_CA_FILE, the other only in
	// the profile PEM.
	fileSrv, filePEM := distinctTLSServer(t)
	profileSrv, profilePEM := distinctTLSServer(t)
	file := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(file, filePEM, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                  string
		caFile                string
		profile               []byte
		wantFile, wantProfile bool
	}{
		{name: "both", caFile: file, profile: profilePEM, wantFile: true, wantProfile: true},
		{name: "CA file only", caFile: file, wantFile: true},
		{name: "profile only", profile: profilePEM, wantProfile: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// When building the client from those sources.
			c, err := newHTTPClient(tc.caFile, tc.profile)
			if err != nil {
				t.Fatal(err)
			}
			// Then exactly the servers whose certificate was supplied are trusted.
			for _, s := range []struct {
				name, url string
				want      bool
			}{{"CA file server", fileSrv.URL, tc.wantFile}, {"profile server", profileSrv.URL, tc.wantProfile}} {
				if err := get(c, s.url); (err == nil) != s.want {
					t.Fatalf("GET %s: err = %v, want trusted = %v", s.name, err, s.want)
				}
			}
		})
	}
}

func TestNewHTTPClient_whenProfileCAHasNoCertificates(t *testing.T) {
	// Given a profile CA that contains no PEM certificate.
	// When building the client.
	_, err := newHTTPClient("", []byte("not a certificate"))
	// Then configuration fails with the fixed message.
	if err == nil || err.Error() != "the wf CLI profile's CA certificate contains no PEM certificates" {
		t.Fatalf("err = %v", err)
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

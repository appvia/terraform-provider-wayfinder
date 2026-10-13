// Package fakeapi is an in-process fake of the Wayfinder API used by the provider's tests.
//
// It reproduces the server semantics the provider depends on (plan reference R9), verified
// against the Wayfinder source; every behaviour carries a citation to the Wayfinder file it
// mirrors. Where the fake has to choose behaviour the source does not pin down, the comment says
// "ASSUMPTION". Objects are untyped JSON (map[string]any) driven by registered KindSpecs, so any
// manifest kind can be exercised. The fake never writes request bodies anywhere.
package fakeapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Wire constants (pkg/common/errors.go:65-72 in Wayfinder).
const (
	HeaderObjectModified      = "x-wayfinder-objectmodified"
	HeaderObjectAlreadyExists = "x-wayfinder-objectalreadyexists"
	// ManagedByLabel is the label the fake adds to every created object, standing in for
	// labels the real server adds on its own (D-CRUD: updates must keep server-added keys).
	ManagedByLabel = "wayfinder.appvia.io/managed-by-fake"
	// DefaultToken is a static bearer token accepted by every new Server.
	DefaultToken = "fake-wayfinder-token"
	// DefaultServerVersion is the release reported by /serverinfo unless changed.
	DefaultServerVersion = "v0.0.0-fake"
)

// Request is one entry of the request log. Body is only populated after RecordBodies(true),
// because request bodies may contain secrets.
type Request struct {
	Method   string
	Path     string
	RawQuery string
	Status   int
	Body     []byte
}

// Server is the fake API. All exported methods are safe for concurrent use.
type Server struct {
	// URL is the base URL of the running server (no trailing slash).
	URL string

	t   testing.TB
	srv *httptest.Server

	mu            sync.Mutex
	seq           int
	kinds         map[kindKey]*KindSpec
	objects       map[Ref]*entry
	vars          map[VarScope]*varSet
	auth          authState
	serverVersion string
	recordBodies  bool
	requests      []Request
}

// New starts a fake server that is closed when the test ends.
func New(t testing.TB) *Server {
	t.Helper()
	s := &Server{
		t:             t,
		kinds:         map[kindKey]*KindSpec{},
		objects:       map[Ref]*entry{},
		vars:          map[VarScope]*varSet{},
		auth:          newAuthState(),
		serverVersion: DefaultServerVersion,
	}
	s.srv = httptest.NewServer(s)
	s.URL = s.srv.URL
	t.Cleanup(s.srv.Close)
	return s
}

// ServeHTTP routes a request. One lock is held for the whole request, so every handler sees a
// consistent store; that serialises requests, which is fine for a test fake.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		writeError(rec, r, http.StatusBadRequest, "unreadable body")
	} else {
		s.route(rec, r, body)
	}
	logged := Request{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery, Status: rec.status}
	if s.recordBodies {
		logged.Body = body
	}
	s.requests = append(s.requests, logged)
}

const (
	resourcesPrefix = "/api/v3/resources/"
	tenantsPrefix   = "/api/v3/general/tenants/"
	tokenPath       = "/api/v3/general/login/token"
	serverInfoPath  = "/api/v3/general/serverinfo"
)

func (s *Server) route(w http.ResponseWriter, r *http.Request, body []byte) {
	path := r.URL.Path
	switch {
	case path == tokenPath && r.Method == http.MethodPost:
		// Unauthenticated by design (pkg/wfclient/refresh.go:98-100 uses Unauthenticated()).
		s.handleToken(w, r, body)
		return
	case path == serverInfoPath && r.Method == http.MethodGet:
		// ASSUMPTION: serverinfo is readable without a token (the CLI calls it before login).
		s.handleServerInfo(w)
		return
	}
	if !s.authorized(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	switch {
	case strings.HasPrefix(path, resourcesPrefix):
		s.handleResource(w, r, body)
	case strings.HasPrefix(path, tenantsPrefix):
		s.handleVars(w, r, body)
	default:
		writeError(w, r, http.StatusNotFound, "not found")
	}
}

// handleServerInfo mirrors pkg/apiserver/handlers/serverinfo.go:64-74 (types.ServerInfo).
func (s *Server) handleServerInfo(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version":            map[string]any{"release": s.serverVersion, "sha": "fake"},
		"instanceIdentifier": "fake",
		"namespace":          "wayfinder",
	})
}

// SetServerVersion sets the release reported by GET /api/v3/general/serverinfo.
func (s *Server) SetServerVersion(release string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serverVersion = release
}

// RecordBodies controls whether request bodies are kept in the request log.
func (s *Server) RecordBodies(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recordBodies = enabled
}

// Requests returns a copy of the request log.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// ResetRequests clears the request log.
func (s *Server) ResetRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = nil
}

// CountRequests counts logged requests matching method ("" = any) and path prefix ("" = any).
func (s *Server) CountRequests(method, pathPrefix string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range s.requests {
		if (method == "" || r.Method == method) && strings.HasPrefix(r.Path, pathPrefix) {
			n++
		}
	}
	return n
}

func (s *Server) nextID(prefix string) string {
	s.seq++
	return prefix + "-" + strconv.Itoa(s.seq)
}

func now() time.Time { return time.Now().UTC() }

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes the generic types.Error body (pkg/apiserver/errors/errors.go:64-77,
// pkg/apiserver/types/errors.go:20-28).
func writeError(w http.ResponseWriter, r *http.Request, code int, message string) {
	writeJSON(w, code, map[string]any{"code": code, "message": message, "verb": r.Method, "uri": r.URL.RequestURI()})
}

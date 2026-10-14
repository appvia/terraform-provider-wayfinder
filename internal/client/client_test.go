package client_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func TestCRUD_roundtrip_against_fake(t *testing.T) {
	h := newHarness(t)
	c, ctx := h.client(), t.Context()

	created, err := c.Create(ctx, envRef("dev"), envObject("dev"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	md := created["metadata"].(map[string]any)
	if md["generation"] != float64(1) || md["name"] != "dev" {
		t.Fatalf("created metadata = %v", md)
	}

	got, err := c.Get(ctx, envRef("dev"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got["spec"].(map[string]any)["stage"] = "prod"
	updated, err := c.Update(ctx, envRef("dev"), got)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if g := updated["metadata"].(map[string]any)["generation"]; g != float64(2) {
		t.Fatalf("updated generation = %v, want 2", g)
	}
	if s := updated["spec"].(map[string]any)["stage"]; s != "prod" {
		t.Fatalf("updated stage = %v", s)
	}

	if err := c.Delete(ctx, envRef("dev")); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.Get(ctx, envRef("dev")); !errors.Is(err, client.ErrNotFound) {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}
	for _, r := range h.fake.Requests() {
		if r.RawQuery != "" {
			t.Errorf("%s %s sent query %q; the client must never send query params", r.Method, r.Path, r.RawQuery)
		}
	}
}

func TestCRUD_versioned_kind(t *testing.T) {
	h := newHarness(t)
	c, ctx := h.client(), t.Context()
	ref := client.Ref{Group: "kubeplans.appvia.io", Version: "v3", APIPath: "kuberesourceplans", Name: "redis", ObjectVersion: "1.0.0", Versioned: true}
	obj := map[string]any{"metadata": map[string]any{"name": "redis", "version": "1.0.0"}, "spec": map[string]any{}}

	if _, err := c.Create(ctx, ref, obj); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := c.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if v := got["metadata"].(map[string]any)["version"]; v != "1.0.0" {
		t.Fatalf("version = %v", v)
	}
	if n := h.fake.CountRequests(http.MethodGet, "/api/v3/resources/tenants/acme/kubeplans.appvia.io/v3/kuberesourceplans/redis/versions/1.0.0"); n != 1 {
		t.Fatalf("versioned GET count = %d", n)
	}
}

func TestUpdate_stale_generation_maps_to_ErrModified(t *testing.T) {
	h := newHarness(t)
	c, ctx := h.client(), t.Context()
	h.fake.Seed(fakeEnvRef("dev"), envObject("dev"))
	got, err := c.Get(ctx, envRef("dev"))
	if err != nil {
		t.Fatal(err)
	}
	h.fake.Mutate(fakeEnvRef("dev"), func(o map[string]any) { o["spec"].(map[string]any)["stage"] = "prod" })

	_, err = c.Update(ctx, envRef("dev"), got)
	if !errors.Is(err, client.ErrModified) {
		t.Fatalf("Update = %v, want ErrModified", err)
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict || apiErr.Message != "OBJECT_MODIFIED" {
		t.Fatalf("APIError = %+v", apiErr)
	}
	if errors.Is(err, client.ErrAlreadyExists) {
		t.Fatal("objectmodified must not match ErrAlreadyExists")
	}
}

func TestCreate_existing_maps_to_ErrAlreadyExists(t *testing.T) {
	h := newHarness(t)
	c, ctx := h.client(), t.Context()
	h.fake.Seed(fakeEnvRef("dev"), envObject("dev"))

	_, err := c.Create(ctx, envRef("dev"), envObject("dev"))
	if !errors.Is(err, client.ErrAlreadyExists) {
		t.Fatalf("Create = %v, want ErrAlreadyExists", err)
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "Environment dev already exists" {
		t.Fatalf("server message not passed through: %+v", apiErr)
	}
}

func TestMissing_object_maps_to_ErrNotFound(t *testing.T) {
	h := newHarness(t)
	c, ctx := h.client(), t.Context()
	for name, call := range map[string]func() error{
		"get":    func() error { _, err := c.Get(ctx, envRef("nope")); return err },
		"update": func() error { _, err := c.Update(ctx, envRef("nope"), envObject("nope")); return err },
		"delete": func() error { return c.Delete(ctx, envRef("nope")) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, client.ErrNotFound) {
				t.Fatalf("err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestValidation_400_maps_to_ValidationError(t *testing.T) {
	h := newHarness(t)
	c, ctx := h.client(), t.Context()
	obj := envObject("dev")
	obj["spec"].(map[string]any)["stage"] = "bogus"

	_, err := c.Create(ctx, envRef("dev"), obj)
	var vErr *client.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("Create = %v, want *ValidationError", err)
	}
	want := client.FieldError{Field: "spec.stage", Code: "invalidValue", Message: "stage must be one of dev, prod"}
	if len(vErr.Fields) != 1 || vErr.Fields[0] != want || vErr.Message != "Resource validation failed" {
		t.Fatalf("ValidationError = %+v", vErr)
	}
	if !strings.Contains(err.Error(), "spec.stage") {
		t.Fatalf("error text %q lacks field path", err)
	}
}

func TestOther_errors_pass_server_message_through(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	c.TokenSource = staticToken("wrong")

	_, err := c.Get(t.Context(), envRef("dev"))
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized || apiErr.Message != "unauthorized" {
		t.Fatalf("err = %v (%+v)", err, apiErr)
	}
}

func TestGet_retries_503_then_succeeds(t *testing.T) {
	h := newHarness(t)
	h.fake.Seed(fakeEnvRef("dev"), envObject("dev"))
	h.failWith(http.MethodGet, http.StatusServiceUnavailable, 2, "")

	if _, err := h.client().Get(t.Context(), envRef("dev")); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if n := h.count(http.MethodGet); n != 3 {
		t.Fatalf("GET attempts = %d, want 3", n)
	}
}

func TestGet_retries_each_retryable_status(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			h := newHarness(t)
			h.fake.Seed(fakeEnvRef("dev"), envObject("dev"))
			h.failWith(http.MethodGet, status, 1, "")
			if _, err := h.client().Get(t.Context(), envRef("dev")); err != nil {
				t.Fatalf("Get: %v", err)
			}
		})
	}
}

func TestGet_gives_up_after_5_retries(t *testing.T) {
	h := newHarness(t)
	h.failWith(http.MethodGet, http.StatusServiceUnavailable, 10, "")

	_, err := h.client().Get(t.Context(), envRef("dev"))
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("err = %v, want 503 APIError", err)
	}
	if n := h.count(http.MethodGet); n != 6 {
		t.Fatalf("GET attempts = %d, want 6 (1 + 5 retries)", n)
	}
}

func TestGet_honours_Retry_After(t *testing.T) {
	h := newHarness(t)
	h.fake.Seed(fakeEnvRef("dev"), envObject("dev"))
	h.failWith(http.MethodGet, http.StatusTooManyRequests, 1, "1")

	start := time.Now()
	if _, err := h.client().Get(t.Context(), envRef("dev")); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Fatalf("retried after %v; Retry-After: 1 not honoured", elapsed)
	}
}

func TestGet_retry_wait_stops_on_context_cancel(t *testing.T) {
	h := newHarness(t)
	h.failWith(http.MethodGet, http.StatusServiceUnavailable, 1, "3600")
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	if _, err := h.client().Get(ctx, envRef("dev")); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestWrites_are_never_retried(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPost, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			h := newHarness(t)
			c, ctx := h.client(), t.Context()
			h.fake.Seed(fakeEnvRef("dev"), envObject("dev"))
			h.failWith(method, http.StatusServiceUnavailable, 1, "")

			var err error
			switch method {
			case http.MethodPut:
				_, err = c.Update(ctx, envRef("dev"), envObject("dev"))
			case http.MethodPost:
				_, err = c.Create(ctx, envRef("other"), envObject("other"))
			case http.MethodDelete:
				err = c.Delete(ctx, envRef("dev"))
			}
			var apiErr *client.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("err = %v, want 503 APIError", err)
			}
			if n := h.count(method); n != 1 {
				t.Fatalf("%s attempts = %d, want 1", method, n)
			}
		})
	}
}

func TestHeaders_user_agent_and_bearer_token_sent(t *testing.T) {
	h := newHarness(t)
	if _, err := h.client().Create(t.Context(), envRef("dev"), envObject("dev")); err != nil {
		t.Fatal(err)
	}
	reqs := h.requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d", len(reqs))
	}
	if ua := reqs[0].Header.Get("User-Agent"); ua != "terraform-provider-wayfinder/1.2.3" {
		t.Errorf("User-Agent = %q", ua)
	}
	if a := reqs[0].Header.Get("Authorization"); a != "Bearer "+fakeapi.DefaultToken {
		t.Errorf("Authorization = %q", a)
	}
}

func TestGet_whenCalled_sendsNoBody(t *testing.T) {
	type seen struct {
		body        string
		contentType string
	}
	got := make(chan seen, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- seen{string(b), r.Header.Get("Content-Type")}
		_, _ = io.WriteString(w, `{}`)
	}))
	t.Cleanup(srv.Close)
	c := &client.Client{BaseURL: srv.URL, Tenant: "acme", TokenSource: staticToken("t"), HTTP: srv.Client()}
	if _, err := c.Get(t.Context(), envRef("dev")); err != nil {
		t.Fatal(err)
	}
	if s := <-got; s.body != "" || s.contentType != "" {
		t.Errorf("GET sent body %q with Content-Type %q; want neither", s.body, s.contentType)
	}
}

func TestToken_fetched_per_request(t *testing.T) {
	h := newHarness(t)
	h.fake.Seed(fakeEnvRef("dev"), envObject("dev"))
	src := &countingToken{}
	c := h.client()
	c.TokenSource = src
	for range 2 {
		if _, err := c.Get(t.Context(), envRef("dev")); err != nil {
			t.Fatal(err)
		}
	}
	if src.n != 2 {
		t.Fatalf("Token calls = %d, want 2", src.n)
	}
}

type countingToken struct{ n int }

func (c *countingToken) Token(context.Context) (string, error) {
	c.n++
	return fakeapi.DefaultToken, nil
}

func TestToken_error_aborts_request(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	boom := errors.New("token exchange failed")
	c.TokenSource = tokenFunc(func(context.Context) (string, error) { return "", boom })
	if _, err := c.Get(t.Context(), envRef("dev")); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want wrapped token error", err)
	}
	if n := len(h.requests()); n != 0 {
		t.Fatalf("requests sent = %d, want 0", n)
	}
}

type tokenFunc func(context.Context) (string, error)

func (f tokenFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

func TestLogs_carry_method_path_status_but_never_body_or_token(t *testing.T) {
	h := newHarness(t)
	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(t.Context(), &buf)
	obj := envObject("dev")
	obj["spec"].(map[string]any)["password"] = "hunter2-secret"

	if _, err := h.client().Create(ctx, envRef("dev"), obj); err != nil {
		t.Fatal(err)
	}
	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil || len(entries) == 0 {
		t.Fatalf("log entries = %v, %v", entries, err)
	}
	e := entries[len(entries)-1]
	if e["method"] != "POST" || e["path"] != "/api/v3/resources/tenants/acme/workspaces/plat/org/v3/environments" || e["status"] != float64(200) {
		t.Errorf("log entry = %v", e)
	}
	for _, secret := range []string{"hunter2-secret", fakeapi.DefaultToken, "Bearer"} {
		if strings.Contains(buf.String(), secret) {
			t.Errorf("log output contains %q", secret)
		}
	}
}

func TestDoJSON_serverinfo(t *testing.T) {
	h := newHarness(t)
	var info struct {
		Version struct {
			Release string `json:"release"`
		} `json:"version"`
	}
	if err := h.client().DoJSON(t.Context(), http.MethodGet, "/api/v3/general/serverinfo", nil, &info); err != nil {
		t.Fatal(err)
	}
	if info.Version.Release != fakeapi.DefaultServerVersion {
		t.Fatalf("release = %q", info.Version.Release)
	}
}

func TestDoJSON_vars_put_and_get(t *testing.T) {
	h := newHarness(t)
	c, ctx := h.client(), t.Context()
	path := "/api/v3/general/tenants/acme/vars/variables/region"
	if err := c.DoJSON(ctx, http.MethodPut, path, map[string]any{"value": "eu-west-2"}, nil); err != nil {
		t.Fatalf("PUT: %v", err)
	}
	var item map[string]any
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &item); err != nil {
		t.Fatalf("GET: %v", err)
	}
	if item["value"] != "eu-west-2" {
		t.Fatalf("item = %v", item)
	}
}

func TestUserAgent_default_when_unset(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	c.UserAgent = ""
	_ = c.DoJSON(t.Context(), http.MethodGet, "/api/v3/general/serverinfo", nil, nil)
	if ua := h.requests()[0].Header.Get("User-Agent"); ua != "terraform-provider-wayfinder/dev" {
		t.Fatalf("User-Agent = %q", ua)
	}
}

// zeros is an endless reader of 'a' bytes used to stream oversized bodies.
type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	return len(p), nil
}

const responseLimit = 32 << 20

func newBodyServer(t *testing.T, hits *atomic.Int32, body func(w io.Writer) error) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = body(w)
	}))
	t.Cleanup(srv.Close)
	return &client.Client{BaseURL: srv.URL, Tenant: tenant, TokenSource: staticToken(fakeapi.DefaultToken), HTTP: srv.Client(), RetryBaseDelay: time.Millisecond}
}

func TestDoJSON_whenResponseExceedsLimit(t *testing.T) {
	var hits atomic.Int32
	c := newBodyServer(t, &hits, func(w io.Writer) error {
		_, err := io.CopyN(w, zeros{}, responseLimit+1)
		return err
	})
	var out map[string]any
	err := c.DoJSON(t.Context(), http.MethodGet, "/api/v3/general/serverinfo", nil, &out)
	if err == nil || !strings.Contains(err.Error(), "exceeds 33554432 bytes") {
		t.Fatalf("err = %v, want size-limit error", err)
	}
	if len(err.Error()) >= 512 {
		t.Fatalf("error too long (%d bytes)", len(err.Error()))
	}
	if n := hits.Load(); n != 1 {
		t.Fatalf("handler hits = %d, want 1 (no retry)", n)
	}
}

func TestDoJSON_whenResponseAtLimit(t *testing.T) {
	var hits atomic.Int32
	c := newBodyServer(t, &hits, func(w io.Writer) error {
		if _, err := io.WriteString(w, `"`); err != nil {
			return err
		}
		if _, err := io.CopyN(w, zeros{}, responseLimit-2); err != nil {
			return err
		}
		_, err := io.WriteString(w, `"`)
		return err
	})
	var out string
	if err := c.DoJSON(t.Context(), http.MethodGet, "/api/v3/general/serverinfo", nil, &out); err != nil {
		t.Fatalf("DoJSON at limit: %v", err)
	}
	if len(out) != responseLimit-2 {
		t.Fatalf("decoded length = %d", len(out))
	}
}

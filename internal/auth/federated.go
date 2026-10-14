package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type federatedIdentity struct {
	file, url, token, audience, tenant string
}

var (
	errStdin    = errors.New("federated_token_file / WAYFINDER_FEDERATED_TOKEN_FILE='-' (stdin) is not supported for provider authentication - set it to a token file path instead")
	errAudience = errors.New("could not determine the required audience from the server - set federated_token_request_audience / WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE explicitly")
	errSubject  = errors.New("identity token response was empty")
)

// File precedence/whitespace mirror pinned pkg/wfclient/federated.go:96-115.
// URL audience and bearer header mirror :124-186; canonical audience is defined
// by server/org/apis/org/v3/serviceaccountcredential_types.go:196-201.
func (s *cachedSource) subjectToken(ctx context.Context) (string, error) {
	ident := s.identity
	if ident.file != "" {
		if ident.file == "-" {
			return "", errStdin
		}
		body, err := os.ReadFile(ident.file)
		if err != nil {
			return "", fmt.Errorf("read federated token file: %w", err)
		}
		return strings.TrimSpace(string(body)), nil
	}
	audience := ident.audience
	if audience == "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.server+"/apiinfo", nil)
		if err != nil {
			return "", fmt.Errorf("%w: %w", errAudience, &safeError{operation: "create API info request", cause: err})
		}
		var info struct {
			UI string `json:"ui"`
		}
		if err := s.decode(req, &info); err != nil {
			return "", fmt.Errorf("%w: %w", errAudience, err)
		}
		if info.UI == "" {
			return "", fmt.Errorf("%w: server API info has no ui URL", errAudience)
		}
		audience = strings.TrimRight(info.UI, "/") + "/t/" + ident.tenant
	}
	requestURL, err := url.Parse(ident.url)
	if err != nil {
		return "", &safeError{operation: "parse identity token URL", cause: err}
	}
	query := requestURL.Query()
	query.Set("audience", audience)
	requestURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return "", &safeError{operation: "create identity token request", cause: err}
	}
	req.Header.Set("Authorization", "Bearer "+ident.token)
	var payload struct {
		Value string `json:"value"`
	}
	if err := s.decode(req, &payload); err != nil {
		return "", fmt.Errorf("fetch identity token: %w", err)
	}
	if payload.Value == "" {
		return "", errSubject
	}
	return payload.Value, nil
}

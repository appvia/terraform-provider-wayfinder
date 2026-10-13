package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

type tokenClaims struct {
	Exp    int64    `json:"exp"`
	Scopes []string `json:"scopes"`
}

var errTokenClaims = errors.New("invalid token claims")
var errBearerExpired = errors.New("the bearer token has expired; supply a new token, client credentials or a federated service account")

// Claims are read only to select the CLI exchange grant and cache expiry, not to
// authenticate a JWT. Signature verification remains the Wayfinder server's job.
func parseClaims(token string) (tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return tokenClaims{}, errTokenClaims
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return tokenClaims{}, fmt.Errorf("decode token claims: %w", errTokenClaims)
	}
	var claims tokenClaims
	if err := json.Unmarshal(body, &claims); err != nil {
		return tokenClaims{}, fmt.Errorf("parse token claims: %w", errTokenClaims)
	}
	return claims, nil
}

func (c tokenClaims) exchangeScoped() bool {
	return slices.Contains(c.Scopes, "wayfinder:auth:exchange")
}

type bearerSource struct {
	token   string
	expires int64
	now     func() time.Time
}

func (s bearerSource) Token(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("get bearer token: %w", err)
	}
	if s.expires > 0 && !s.now().Before(time.Unix(s.expires, 0)) {
		return "", errBearerExpired
	}
	return s.token, nil
}

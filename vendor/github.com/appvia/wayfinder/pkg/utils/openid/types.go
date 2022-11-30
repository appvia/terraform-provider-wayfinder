/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package openid

import (
	"context"

	"github.com/coreos/go-oidc"
)

// Authenticator provides a openid if agent
type Authenticator interface {
	// Provider returns the oidc provider
	Provider() *oidc.Provider
	// RunWithSync start the discovery grab
	RunWithSync(context.Context) error
	// Run start the discovery grab
	Run(context.Context) error
	// Verify is called to verify a token
	Verify(context.Context, string) (*oidc.IDToken, error)
}

// Config is the configuration for the service
type Config struct {
	// ClientID is the client id
	ClientID string
	// ServerURL is the openid server url
	ServerURL string
	// SkipIDCheck indicates we skip checking the issuer
	SkipClientIDCheck bool
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . Verifier
//go:generate go run golang.org/x/tools/cmd/goimports -local github.com/appvia/wayfinder -e -w .
//go:generate gofmt -s -l -w openidfakes/fake_verifier.go
type Verifier interface {
	Verify(ctx context.Context, rawIDToken string) (IDToken, error)
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . IDToken
//go:generate go run golang.org/x/tools/cmd/goimports -local github.com/appvia/wayfinder -e -w .
//go:generate gofmt -s -l -w openidfakes/fake_idtoken.go
type IDToken interface {
	Claims(v interface{}) error
}

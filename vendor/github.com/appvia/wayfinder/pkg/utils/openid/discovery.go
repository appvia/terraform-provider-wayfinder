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
	"errors"
	"sync"
	"time"

	oidc "github.com/coreos/go-oidc"

	"github.com/appvia/wayfinder/pkg/common"
)

type authImpl struct {
	sync.RWMutex
	// config is the configuration
	Config
	// provider is the oidc provider
	provider *oidc.Provider
	// verifier is the oidc token config
	verifier *oidc.IDTokenVerifier
}

// New creates and returns a authenticator
func New(config Config) (Authenticator, error) {
	if err := config.IsValid(); err != nil {
		return nil, err
	}

	return &authImpl{Config: config}, nil
}

// Provider returns the oidc provider
func (a *authImpl) Provider() *oidc.Provider {
	return a.provider
}

// Verify is called to verify the token
func (a *authImpl) Verify(ctx context.Context, token string) (*oidc.IDToken, error) {
	// @step: we lock the struct and check if the verifier has been configured yet
	verifier := a.GetVerifier()
	if verifier == nil {
		common.Log(ctx).Info("openid has not been configured yet")

		return nil, errors.New("unable to verify token")
	}

	return verifier.Verify(ctx, token)
}

// RunWithSync waits for the discovery process to occur
func (a *authImpl) RunWithSync(ctx context.Context) error {
	if err := a.Run(ctx); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)

	nctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	for {
		if a.provider != nil && a.verifier != nil {
			return nil
		}
		select {
		case <-nctx.Done():
			return errors.New("context has been cancelled")
		default:
		}

		time.Sleep(5 * time.Second)
	}
}

// Run starts the request to the discovery url
func (a *authImpl) Run(ctx context.Context) error {
	go func() {
		for {
			err := func() error {
				common.Log(ctx).WithField("server-url", a.ServerURL).
					Info("attempting to retrieve provider details via discovery url")

				// @step: attempt to retrieve the details for the discovery url
				provider, err := oidc.NewProvider(ctx, a.ServerURL)
				if err != nil {
					return err
				}

				verifier := provider.Verifier(&oidc.Config{
					ClientID:          a.ClientID,
					SkipClientIDCheck: a.SkipClientIDCheck,
					SkipExpiryCheck:   false,
				})

				common.Log(ctx).Info("openid authentication plugin successfully retrieve configuration")

				a.Lock()
				defer a.Unlock()

				a.provider = provider
				a.verifier = verifier

				return nil
			}()
			if err != nil {
				common.Log(ctx).WithError(err).Error("failed to retrieve provider configuration")

				time.Sleep(10 * time.Second)
				continue
			}

			return
		}
	}()

	return nil
}

// GetVerifier returns the internal token verifier
func (a *authImpl) GetVerifier() *oidc.IDTokenVerifier {
	a.RLock()
	defer a.RUnlock()

	return a.verifier
}

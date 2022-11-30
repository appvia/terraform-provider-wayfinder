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

package authentication

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/appvia/wayfinder/pkg/authentication/oauth"
	"github.com/appvia/wayfinder/pkg/common"
)

var (
	liveProvider *oauth.OAuthLoginProvider
	testProvider *oauth.OAuthLoginProvider
)

const (
	IDPInstallSecretName = "wayfinder-idp"
	IDPLiveSecretName    = "wayfinder-idp-live"
	IDPTestSecretName    = "wayfinder-idp-test"
)

func GetIDPProvider() *oauth.OAuthLoginProvider {
	return liveProvider
}

func GetTestIDPProvider() *oauth.OAuthLoginProvider {
	return testProvider
}

func InitialiseIDP(ctx context.Context, kubeconfig *rest.Config, stopper chan struct{}) error {
	common.Log(ctx).Info("Initializing IDP login provider")

	k8s, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubeclient to initialise IDP integration")
	}

	liveProvider, err = initialiseOAuthProvider(ctx, "IDP", k8s, stopper,
		IDPLiveSecretName,
		WithDefaultSecret(IDPInstallSecretName, common.Namespace),
	)
	if err != nil {
		return err
	}

	testProvider, err = initialiseOAuthProvider(ctx, "TEST IDP", k8s, stopper,
		IDPTestSecretName,
	)
	if err != nil {
		return err
	}

	return nil
}

func initialiseOAuthProvider(ctx context.Context, name string, k8s kubernetes.Interface, stopper chan struct{}, secretName string, opts ...SecretWatcherOption) (*oauth.OAuthLoginProvider, error) {
	provider := oauth.New()
	providerConfigurer := func(secret *v1.Secret) {
		// Use a local context, as this may no longer operate in the context of the original
		// initialisation routine
		ctx := context.Background()

		conf := oauth.DecodeSecret(secret)
		common.Log(ctx).Infof("%s configuration secret set - configuring", name)
		err := provider.Configure(context.Background(), &conf)
		if err != nil {
			common.LogWithoutContext().WithError(err).Errorf("Error encountered (re)configuring %s - integration will be disabled until this issue is resolved", name)
		}

		if provider.Enabled() {
			common.Log(ctx).Infof("%s login provider configured and enabled", name)
		} else {
			common.Log(ctx).Infof("%s login provider disabled - will be enabled when IDP secret %s/%s is created and valid", name, common.Namespace, secretName)
		}
	}

	// Prepare a watcher for the secret
	watcher := NewSecretWatcher(name, k8s, secretName, common.Namespace, providerConfigurer, opts...)
	currentSecret, err := watcher.Initialise(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise %s integration: %w", name, err)
	}

	// Do initial configuration, using any initial secret retrieved by the watcher
	providerConfigurer(currentSecret)

	// Start watching so changes to the secret cause a re-configuration
	watcher.Watch(ctx, stopper)

	return provider, nil
}

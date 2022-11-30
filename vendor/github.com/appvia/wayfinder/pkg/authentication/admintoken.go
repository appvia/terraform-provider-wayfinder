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

	"github.com/appvia/wayfinder/pkg/authentication/admintoken/tokensource"
	"github.com/appvia/wayfinder/pkg/common"
)

const (
	AdminTokenSecretName = "wayfinder-api"
)

func InitialiseAdminToken(ctx context.Context, kubeconfig *rest.Config, stopper chan struct{}) error {
	k8s, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubeclient to initialise admin token auth")
	}

	adminTokenConfigurer := func(secret *v1.Secret) {
		newToken := ""
		if secret != nil {
			if tbytes, ok := secret.Data["WF_ADMIN_TOKEN"]; ok {
				newToken = string(tbytes)
			}
		}

		if err := tokensource.SetToken(newToken); err != nil {
			common.LogWithoutContext().WithError(err).Errorf("Error encountered (re)configuring admin token - admin token login will be disabled until this is resolved")
		}

		if tokensource.Enabled() {
			common.Log(ctx).Infof("Admin token authentication enabled")
		} else {
			common.Log(ctx).Infof("Admin token authentication disabled - will be enabled if secret %s/%s is updated and valid", common.Namespace, AdminTokenSecretName)
		}
	}

	// Prepare a watcher for the secret
	watcher := NewSecretWatcher("Admin Token", k8s, AdminTokenSecretName, common.Namespace, adminTokenConfigurer)
	currentSecret, err := watcher.Initialise(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialise admin token auth: %w", err)
	}

	// Do initial configuration, using any initial secret retrieved by the watcher
	adminTokenConfigurer(currentSecret)

	// Start watching so changes to the secret cause a re-configuration
	watcher.Watch(ctx, stopper)

	return nil
}

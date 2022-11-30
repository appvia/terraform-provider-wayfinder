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
	"reflect"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/appvia/wayfinder/pkg/common"
)

type SecretWatcher struct {
	name            string
	k8s             kubernetes.Interface
	secretName      string
	secretNamespace string

	defaultSecretName      string
	defaultSecretNamespace string

	watcher SecretListener

	current *v1.Secret

	resync time.Duration
}

type SecretListener func(*v1.Secret)

type SecretWatcherOption func(*SecretWatcher)

// WithDefaultSecret adds a default secret name/namespace which will be copied as the live secret
// if no existing secret is found. This can be used (for example) to copy the values from the secret
// provided by the installer to the live secret.
var WithDefaultSecret = func(name, namespace string) SecretWatcherOption {
	return func(s *SecretWatcher) {
		s.defaultSecretName = name
		s.defaultSecretNamespace = namespace
	}
}

// WithResync sets a custom resync period for the watch (default 5 minutes)
var WithResync = func(dur time.Duration) SecretWatcherOption {
	return func(s *SecretWatcher) {
		s.resync = dur
	}
}

// NewSecretWatcher creates a new secret watcher with the given name (used only for logging), watching
// the specified secret.
func NewSecretWatcher(name string, k8s kubernetes.Interface, secretName, secretNamespace string, watcher SecretListener, opts ...SecretWatcherOption) *SecretWatcher {
	scs := &SecretWatcher{
		name:            name,
		k8s:             k8s,
		secretName:      secretName,
		secretNamespace: secretNamespace,
		resync:          5 * time.Minute,
		watcher:         watcher,
	}

	for _, o := range opts {
		o(scs)
	}

	return scs
}

func (s *SecretWatcher) Initialise(ctx context.Context) (*v1.Secret, error) {
	current, err := s.k8s.CoreV1().Secrets(s.secretNamespace).Get(ctx, s.secretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			current = nil
		} else {
			return nil, fmt.Errorf("unable to load secret to initialise: %w", err)
		}
	}

	if current == nil {
		// Check if we have a default and copy it to the live
		var err error
		// This might still be nil, but that's fine because nil is valid
		current, err = s.initialiseFromDefault(ctx)
		if err != nil {
			return nil, err
		}
	}

	s.updateIfChanged(current)
	return current, nil
}

func (s *SecretWatcher) initialiseFromDefault(ctx context.Context) (*v1.Secret, error) {
	// This is a no-op if no default provided
	if s.defaultSecretName == "" || s.defaultSecretNamespace == "" {
		return nil, nil
	}

	common.Log(ctx).Infof("Initializing %s configuration from default secret %s/%s", s.name, s.defaultSecretNamespace, s.defaultSecretName)
	def, err := s.k8s.CoreV1().Secrets(s.defaultSecretNamespace).Get(ctx, s.defaultSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Also a no-op if the specified default does not exist
			common.Log(ctx).Infof("Default %s secret %s/%s does not exist, nothing to initialize from", s.name, s.defaultSecretNamespace, s.defaultSecretName)
			return nil, nil
		}
		return nil, fmt.Errorf("unable to load specified default secret to initialise: %w", err)
	}

	current := &v1.Secret{}
	current.Namespace = s.secretNamespace
	current.Name = s.secretName
	current.Data = map[string][]byte{}
	for k, v := range def.Data {
		current.Data[k] = v
	}

	persisted, err := s.k8s.CoreV1().Secrets(s.secretNamespace).Create(ctx, current, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to create live secret from default: %w", err)
	}

	return persisted, nil
}

func (s *SecretWatcher) Watch(ctx context.Context, stopper chan struct{}) {
	f := informers.NewSharedInformerFactoryWithOptions(s.k8s, s.resync, informers.WithNamespace(s.secretNamespace))
	si := f.Core().V1().Secrets()
	si.Informer().AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret := obj.(*v1.Secret)
			if secret.Name != s.secretName {
				return
			}
			s.updateIfChanged(secret)

		},
		UpdateFunc: func(objOld interface{}, objNew interface{}) {
			secret := objNew.(*v1.Secret)
			if secret.Name != s.secretName {
				return
			}
			s.updateIfChanged(secret)

		},
		DeleteFunc: func(obj interface{}) {
			secret := obj.(*v1.Secret)
			if secret.Name != s.secretName {
				return
			}
			s.updateIfChanged(nil)
		},
	}, s.resync)

	go si.Informer().Run(stopper)
}

func (s *SecretWatcher) updateIfChanged(secret *v1.Secret) {
	// If we have a nil secret, empty current and notify any watchers
	if secret == nil {
		s.current = nil

		if s.watcher != nil {
			s.watcher(s.current)
		}

		return
	}

	// If we have a non-nil secret, update current if it's different to the current secret
	if s.current == nil || !reflect.DeepEqual(secret.Data, s.current.Data) {
		s.current = secret

		if s.watcher != nil {
			s.watcher(s.current)
		}
	}
}

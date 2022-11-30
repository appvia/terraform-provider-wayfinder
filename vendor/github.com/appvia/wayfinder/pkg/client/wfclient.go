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

package client

import (
	"context"
	"fmt"

	"github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/client/config"
	"github.com/appvia/wayfinder/pkg/utils/retry"
)

// NewWFClient provides a Wayfinder client styled after the lovely simple interface of controller-manager's client.Client.
func NewWFClient(cfg *config.Config, options ...OptionFunc) (WFClient, error) {
	// Get underlying scratchy client
	c, err := New(cfg, options...)
	if err != nil {
		return nil, err
	}

	return NewWFClientForClient(c), nil
}

// NewWFClientForClient wraps the provided client in a WFClient, for a nice migration path
func NewWFClientForClient(c Interface) WFClient {
	return &wfClient{c}
}

type wfClient struct {
	c Interface
}

func (s *wfClient) Get(ctx context.Context, key ObjectKey, obj Object) error {
	req := s.c.Request()
	if key.Workspace != "" {
		req = req.Workspace(key.Workspace)
	}
	return req.Resource(obj.APIPath()).
		Name(key.Name).
		Result(obj).
		Get().
		Error()
}

func (s *wfClient) List(ctx context.Context, list ObjectList, opts ...ListOption) error {
	o := GetListOpts(opts)
	req := s.c.Request()
	if o.InWorkspace != "" {
		req = req.Workspace(o.InWorkspace)
	}
	return req.Resource(list.ObjectType().APIPath()).
		Result(list).
		Get().
		Error()
}

func (s *wfClient) Create(ctx context.Context, obj Object, opts ...CreateOption) error {
	o := GetCreateOpts(opts)
	req := s.c.Request()
	if obj.GetNamespace() != "" {
		req = req.Workspace(v1alpha1.Workspace(obj))
	}
	if o.DryRun {
		req = req.Parameters(DryRunParameter())
	}
	return req.Resource(obj.APIPath()).
		Payload(obj).
		Result(obj).
		Post().
		Error()
}

func (s *wfClient) Delete(ctx context.Context, obj Object, opts ...DeleteOption) error {
	o := GetDeleteOptions(opts)
	req := s.c.Request()
	if obj.GetNamespace() != "" {
		req = req.Workspace(v1alpha1.Workspace(obj))
	}
	req = req.Resource(obj.APIPath()).
		Name(obj.GetName()).
		Result(obj)
	if o.DryRun {
		req = req.Parameters(DryRunParameter())
	}
	if o.Orphan {
		req = req.Parameters(OrphanParameter())
	}
	if o.Cascade {
		req = req.Parameters(CascadeParameter())
	}
	if o.Force {
		req = req.Parameters(ForceParameter())
	}
	return req.Delete().Error()
}

func (s *wfClient) Update(ctx context.Context, obj Object, opts ...UpdateOption) error {
	uo := GetUpdateOpts(opts)
	var lastErr error
	// Retry a few times if an object modified error occurs
	err := retry.Retry(ctx, 3, false, 0, func() (bool, error) {
		req := s.c.Request()
		if obj.GetNamespace() != "" {
			req = req.Workspace(v1alpha1.Workspace(obj))
		}
		if uo.DryRun {
			req = req.Parameters(DryRunParameter())
		}
		if uo.Force {
			req = req.Parameters(ForceParameter())
		}

		if err := req.Resource(obj.APIPath()).
			Name(obj.GetName()).
			Payload(obj).
			Result(obj).
			Update().
			Error(); err != nil {
			if !IsObjectModified(err) || uo.NoRetryOnConflict {
				return false, err
			}
			// For an object modified error, let's see if we can get the updated object from the
			// server and check if the generation is the same - if it is, it's just a status update
			// so we can safely update the resource version and retry.
			lastErr = err
			upd := obj.Clone()
			if gerr := s.Get(ctx, ObjectKeyFromObject(obj), upd); gerr != nil {
				return false, fmt.Errorf("failed to retrieve updated version of the object after an object modified conflict: %w", gerr)
			}

			if upd.GetGeneration() != obj.GetGeneration() {
				// Generation changed so it is not safe to do a silent retry of this one,
				// just throw the original error.
				return false, err
			}

			// Generation the same - i.e. spec not updated, so it is (reasonably) safe to retry
			obj.SetResourceVersion(upd.GetResourceVersion())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		if retry.IsRetryFailed(err) && lastErr != nil {
			return lastErr
		}
		return err
	}
	return nil
}

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

package common

import (
	"context"
	"errors"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	// ErrIgnore is used to used to stop the ensure chain
	ErrIgnore = errors.New("resource is being ignored")
	// DefaultEnsureHandler is the default sequential runner for a list of ensure functions
	DefaultEnsureHandler = EnsureRunner{}
	// AggregateEnsureHandler runs all ensure funcs and returns an aggregated error
	AggregateEnsureHandler = AggregateEnsureRunner{}
)

// EnsureFunc defines a method to ensure a state
type EnsureFunc func(ctx context.Context) (reconcile.Result, error)

// EnsureRunner provides a wrapper for running ensure funcs
type EnsureRunner struct{}

// AggregateEnsureRunner provides a wrapper for running ensure funcs
type AggregateEnsureRunner struct{}

// RequeueImmediate should be used anywhere we wish an immediate / ASAP requeue to be performed
var RequeueImmediate = reconcile.Result{RequeueAfter: 5 * time.Millisecond}

// Run is a generic handler for running the ensure methods
func (e *EnsureRunner) Run(ctx context.Context, ensures []EnsureFunc) (reconcile.Result, error) {
	for _, x := range ensures {
		result, err := x(ctx)
		if err != nil {
			// Always requeue on conflict errors
			if kerrors.IsConflict(err) {
				return reconcile.Result{Requeue: true}, nil
			}

			// Skip the rest of the ensure funcs if anything returns ErrIgnore
			if err == ErrIgnore {
				return reconcile.Result{}, nil
			}

			return reconcile.Result{}, err
		}
		if result.Requeue || result.RequeueAfter > 0 {
			return result, nil
		}
	}

	return reconcile.Result{}, nil
}

// IsResultRequeuing returns true if the reconcile.Result is asking to requeue
func IsResultRequeuing(result reconcile.Result) bool {
	return result.Requeue || result.RequeueAfter > 0
}

// Run is a generic handler for running the ensure methods
func (e *AggregateEnsureRunner) Run(ctx context.Context, ensures []EnsureFunc) (reconcile.Result, error) {
	aggregatedResult := reconcile.Result{}
	var aggregatedError error
	for _, x := range ensures {
		result, err := x(ctx)
		if err != nil {
			switch {
			case err == ErrIgnore:
				return reconcile.Result{}, nil
			case kerrors.IsConflict(err):
				aggregatedResult = reconcile.Result{Requeue: true}
			default:
				aggregatedError = err
			}
		}
		switch {
		case result.Requeue:
			aggregatedResult.Requeue = true
		case aggregatedResult.RequeueAfter == 0:
			aggregatedResult.RequeueAfter = result.RequeueAfter
		case result.RequeueAfter < aggregatedResult.RequeueAfter:
			aggregatedResult.RequeueAfter = result.RequeueAfter
		}
	}

	if aggregatedResult.RequeueAfter > 0 {
		aggregatedResult.Requeue = false
	}

	return aggregatedResult, aggregatedError
}

// EnsureWithRetry will call an ensure function until it is complete or an error
func EnsureWithRetry(ctx context.Context, fn EnsureFunc) error {
	for {
		result, err := fn(ctx)
		if err != nil {
			if err == ErrIgnore {
				return nil
			}

			return err
		}
		switch {
		case result.Requeue:
		case result.RequeueAfter > 0:
			time.Sleep(result.RequeueAfter)
		default:
			return nil
		}
	}
}

/*
// On this spot, 17/12/2021, a brave recursive method fought valiantly and was slain in battle
//  with the iterite elect.
func (e *EnsureRunner) Run(ctx context.Context, ensures []EnsureFunc) (restail reconcile.Result, errtail error) {
	if len(ensures) == 0 {
		return reconcile.Result{}, nil
	}
	reshead, errhead := ensures[0](ctx)
	switch {
	case errhead == ErrIgnore:
		return reconcile.Result{}, nil
	case e.RunAll:
		defer func() {
			switch {
			case errhead != nil:
				errtail = errhead
			case reshead.Requeue || reshead.RequeueAfter > 0:
				restail = reshead
			}
		}()
	case kerrors.IsConflict(errhead):
		return reconcile.Result{Requeue: true}, nil
	case errhead != nil:
		return reconcile.Result{}, errhead
	case reshead.Requeue || reshead.RequeueAfter > 0:
		return reshead, nil
	}

	// I know a song that'll get on your nerves...
	return e.Run(ctx, ensures[1:])
}
*/

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

package retry

import (
	"context"
	"errors"
	"time"

	"github.com/appvia/wayfinder/pkg/utils/sleep"
)

var (
	// ErrTimeout indicates the operation has been timed out
	ErrTimeout = errors.New("operation has timed out")
	// ErrCancelled indicates the operaton was cancelled
	ErrCancelled = errors.New("operation has been cancelled")
)

// Forever is called on interval for eternity or until cancelled
func Forever(ctx context.Context, interval time.Duration, method func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			method()
		case <-ctx.Done():
			return
		}
		if sleep.Sleep(ctx, interval) {
			return
		}
	}
}

// WaitUntilComplete calls the condition on every interval and check for true, nil. An error indicates a hard error and exits
func WaitUntilComplete(ctx context.Context, timeout time.Duration, interval time.Duration, condition func() (bool, error)) error {
	nc, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if interval <= 0 {
		return errors.New("invalid interval")
	}

	for {
		select {
		case <-nc.Done():
			return ErrCancelled
		default:
		}

		if done, err := condition(); err != nil {
			return err
		} else if done {
			return nil
		}

		if sleep.Sleep(ctx, interval) {
			return ErrCancelled
		}
	}
}

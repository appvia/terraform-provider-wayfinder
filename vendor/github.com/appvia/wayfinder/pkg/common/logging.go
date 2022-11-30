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
	"fmt"
	"strings"

	// This is imported to provide a reasonable default logger, however see
	// pkg/cmd/defaults.go for the main instantiation of the logger in the Wayfinder
	// codebase.
	//
	// No usage of logging should import logrus directly; the few places where it
	// is imported for logging configuration purposes have been reviewed and
	// deemed acceptable.
	//
	// nolint:depguard
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

// Logger represents Wayfinder's standard logging interface
type Logger interface {
	logrus.Ext1FieldLogger
}

// Default to logrus for backwards compat for now
var defaultLogger Logger = logrus.StandardLogger()

var logProvider = func() Logger {
	return defaultLogger
}

// SetLogProvider allows the current logging provider to be set
func SetLogProvider(lf func() Logger) {
	logProvider = lf
}

type logContexts int

const (
	logContextController logContexts = iota
	logContextResource
	logContextAPI
	logContextUser
	eventContextRecorder
)

// WithController creates a sub-context with the specified controller added
func WithController(ctx context.Context, ctrl string) context.Context {
	return context.WithValue(ctx, logContextController, ctrl)
}

// GetController pulls the controller out of an existing context
// This isn't safe, only use it with a controller's context
func GetController(ctx context.Context) string {
	return ctx.Value(logContextController).(string)
}

// WithResource creates a sub-context with the specified resource added
func WithResource(ctx context.Context, nn types.NamespacedName) context.Context {
	return context.WithValue(ctx, logContextResource, nn)
}

// WithControllerResource creates a sub-context with the specified controller and resource added
// Shortcut for calling WithController and WithResource separately as we use this a lot
func WithControllerResource(ctx context.Context, ctrl string, nn types.NamespacedName) context.Context {
	ctx = WithController(ctx, ctrl)
	ctx = WithResource(ctx, nn)
	return ctx
}

// WithRecorder adds the specified event recorder into the context
func WithRecorder(ctx context.Context, recorder record.EventRecorder) context.Context {
	return context.WithValue(ctx, eventContextRecorder, recorder)
}

type apiCall struct {
	path   string
	method string
}

// WithAPI adds the URL and methods to the context key values, returning a new context
func WithAPI(ctx context.Context, path string, method string) context.Context {
	return context.WithValue(ctx, logContextAPI, apiCall{path, method})
}

// WithUser add the username to the context key values, returning a new context
func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, logContextUser, user)
}

// LogWithoutContext should ONLY be used where there is no relevant context - you should prefer
// calling Log(ctx) and add contexts in where they are missing to using this.
func LogWithoutContext() Logger {
	return logProvider()
}

// Log gets a logger to use, initialised with any relevant values from the context
func Log(ctx context.Context) Logger {
	log := logProvider()
	if ctx != nil {
		if ctrl, ok := ctx.Value(logContextController).(string); ok {
			log = log.WithField("controller", ctrl)
		}
		if nn, ok := ctx.Value(logContextResource).(types.NamespacedName); ok {
			log = log.WithField("resource", nn.String())
		}
		if a, ok := ctx.Value(logContextAPI).(apiCall); ok {
			log = log.WithField("api", fmt.Sprintf("%s:%s", a.method, a.path))
		}
		if user, ok := ctx.Value(logContextUser).(string); ok {
			log = log.WithField("user", user)
		}
	}
	return log
}

// Recorder returns the event recorder registered with the context. It will log a warning and return
// a fake event recorder if no recorder is available in the context.
func Recorder(ctx context.Context) record.EventRecorder {
	if rec, ok := ctx.Value(eventContextRecorder).(record.EventRecorder); ok {
		return rec
	}
	Log(ctx).Warn("Event Recorder requested from context when no event recorder has been registered - no events will be logged")
	return record.NewFakeRecorder(10)
}

// ProgressReporter echos the CLI's logging interface, used to report progress of a multi-step
// process
type ProgressReporter interface {
	// Step should be used to indicate a progress step has commenced
	Step(message string, args ...interface{})
	// Success should be used to indicate an operation has successfully completed
	Success(message string, args ...interface{})
	// Warn should be used to indicate a warning that the user might need to be aware of, but has
	// not caused a failure
	Warn(message string, args ...interface{})
	// Error should be used to indicate a failure has occurred meaning the operation the user
	// requested has not completed successfully
	Error(message string, args ...interface{})
}

func LoggingProgressReporter(ctx context.Context) ProgressReporter {
	return &loggingProgressReporter{l: Log(ctx)}
}

type loggingProgressReporter struct {
	l Logger
}

func (l *loggingProgressReporter) Step(message string, args ...interface{}) {
	l.l.Debugf(message, args...)
}
func (l *loggingProgressReporter) Success(message string, args ...interface{}) {
	l.l.Infof(message, args...)
}
func (l *loggingProgressReporter) Warn(message string, args ...interface{}) {
	l.l.Warnf(message, args...)
}
func (l *loggingProgressReporter) Error(message string, args ...interface{}) {
	l.l.Errorf(message, args...)
}

// NewStringProgressReporter returns a progress reporter that concatenates the reported statuses
// into a single multi-line string. Call .Out() to get the resulting string. It will also log the
// output as per LoggingProgressReporter if log is true
func NewStringProgressReporter(ctx context.Context, log bool) *StringProgressReporter {
	logger := &StringProgressReporter{
		sb: strings.Builder{},
	}
	if log {
		logger.l = Log(ctx)
	}
	return logger
}

type StringProgressReporter struct {
	sb strings.Builder
	l  Logger
}

func (l *StringProgressReporter) Step(message string, args ...interface{}) {
	l.sb.WriteString(fmt.Sprintf("* "+message+"\n", args...))
	if l.l != nil {
		l.l.Debugf(message, args...)
	}
}
func (l *StringProgressReporter) Success(message string, args ...interface{}) {
	l.sb.WriteString(fmt.Sprintf("✓ "+message+"\n", args...))
	if l.l != nil {
		l.l.Infof(message, args...)
	}
}
func (l *StringProgressReporter) Warn(message string, args ...interface{}) {
	l.sb.WriteString(fmt.Sprintf("! "+message+"\n", args...))
	if l.l != nil {
		l.l.Warnf(message, args...)
	}
}
func (l *StringProgressReporter) Error(message string, args ...interface{}) {
	l.sb.WriteString(fmt.Sprintf("X "+message+"\n", args...))
	if l.l != nil {
		l.l.Errorf(message, args...)
	}
}

func (l *StringProgressReporter) Out() string {
	return l.sb.String()
}

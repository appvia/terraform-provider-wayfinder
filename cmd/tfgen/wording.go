// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"regexp"
)

// wordingRules rewrite upstream descriptions that are written for Wayfinder operators into
// wording that suits SaaS users of the provider. They run, in order, on every description
// the generator emits. Keep each rule narrow and add a reason.
var wordingRules = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	// ExternalIdentity federated audience fields: Wayfinder is SaaS, there is no "instance".
	{regexp.MustCompile(`this instance's canonical Wayfinder audience`), "the canonical Wayfinder audience"},
}

// forbiddenWording is checked after wordingRules. Generation fails on a match so new
// upstream wording is reviewed (and given a rule) instead of reaching the docs.
// `WF_` is matched case-sensitively: lower-case `wf_agent` is an attribute name.
var forbiddenWording = regexp.MustCompile(`(?i:varset|your administrator|this instance|wayfinder.{0,20}helm values|helm values.{0,20}wayfinder)|WF_[A-Z]`)

func reword(text string) (string, error) {
	for _, rule := range wordingRules {
		text = rule.pattern.ReplaceAllString(text, rule.replacement)
	}
	if match := forbiddenWording.FindString(text); match != "" {
		return "", fmt.Errorf("user-facing text contains forbidden wording %q; add a rule to cmd/tfgen/wording.go", match)
	}
	return text, nil
}

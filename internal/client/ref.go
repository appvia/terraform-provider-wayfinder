// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// resourcesRoot is the prefix of every resource route (R8, pkg/wfclient/apiurlmanager.go:147-216).
const resourcesRoot = "/api/v3/resources"

// Ref identifies a Wayfinder resource object (or, without Name, its collection).
type Ref struct {
	Group, Version, APIPath string
	// Workspace and Environment select the scope; both empty means tenant scope.
	Workspace, Environment string
	Name                   string
	// ObjectVersion is metadata.version of a versioned kind; required for item routes when
	// Versioned is true and forbidden otherwise.
	ObjectVersion string
	Versioned     bool
}

// errInvalidRef is wrapped by every Ref validation failure.
var errInvalidRef = errors.New("invalid resource reference")

// CollectionPath returns the escaped collection route:
// /api/v3/resources/tenants/{t}[/workspaces/{ws}[/envs/{env}]]/{group}/{version}/{apiPath}.
func (r Ref) CollectionPath(tenant string) (string, error) {
	switch {
	case tenant == "":
		return "", fmt.Errorf("%w: tenant is empty", errInvalidRef)
	case r.Group == "" || r.Version == "" || r.APIPath == "":
		return "", fmt.Errorf("%w: group, version and apiPath are required", errInvalidRef)
	case r.Environment != "" && r.Workspace == "":
		return "", fmt.Errorf("%w: environment %q needs a workspace", errInvalidRef, r.Environment)
	}

	segs := []string{"tenants", tenant}
	if r.Workspace != "" {
		segs = append(segs, "workspaces", r.Workspace)
	}

	if r.Environment != "" {
		segs = append(segs, "envs", r.Environment)
	}

	return join(append(segs, r.Group, r.Version, r.APIPath)), nil
}

// ItemPath returns the escaped item route: the collection path plus /{name}, plus
// /versions/{version} for versioned kinds.
func (r Ref) ItemPath(tenant string) (string, error) {
	col, err := r.CollectionPath(tenant)
	if err != nil {
		return "", err
	}

	switch {
	case r.Name == "":
		return "", fmt.Errorf("%w: name is empty", errInvalidRef)
	case r.Versioned && r.ObjectVersion == "":
		return "", fmt.Errorf("%w: versioned %s %q needs a version", errInvalidRef, r.APIPath, r.Name)
	case !r.Versioned && r.ObjectVersion != "":
		return "", fmt.Errorf("%w: %s is not versioned but version %q was given", errInvalidRef, r.APIPath, r.ObjectVersion)
	}

	item := col + "/" + url.PathEscape(r.Name)
	if r.Versioned {
		item += "/versions/" + url.PathEscape(r.ObjectVersion)
	}

	return item, nil
}

func join(segs []string) string {
	var b strings.Builder
	b.WriteString(resourcesRoot)

	for _, s := range segs {
		b.WriteByte('/')
		b.WriteString(url.PathEscape(s))
	}

	return b.String()
}

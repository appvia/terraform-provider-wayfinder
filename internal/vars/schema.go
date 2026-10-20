// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package vars

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	keyDescription = "Name of the key. Must start with a letter or underscore and contain only " +
		"letters, digits, underscores and dashes. Changing it replaces the resource."
	workspaceDescription = "Workspace the key belongs to. Omit for a tenant-wide key. " +
		"Changing it replaces the resource."
	environmentDescription = "Environment (within `workspace`) the key belongs to. Requires `workspace`. " +
		"Omit for a workspace-wide key. Changing it replaces the resource."
	instanceDescription = "Stack instance (within `environment`) the key belongs to. Requires `workspace` and " +
		"`environment`. Omit for an environment-wide key. Changing it replaces the resource."
	idDescription = "Identifier `[workspace/[environment/[instance/]]]key`, also used for import."
	keyFormat     = "must start with a letter or underscore and contain only letters, digits, underscores and dashes"
)

var segmentPattern = regexp.MustCompile(`^[^/]+$`)

func keyValidators() []validator.String {
	return []validator.String{stringvalidator.RegexMatches(keyPattern, keyFormat)}
}

func workspaceValidators() []validator.String {
	return []validator.String{stringvalidator.RegexMatches(segmentPattern, "must be a non-empty name without '/'")}
}

func environmentValidators() []validator.String {
	return append(workspaceValidators(), stringvalidator.AlsoRequires(path.MatchRoot("workspace")))
}

func instanceValidators() []validator.String {
	return append(workspaceValidators(),
		stringvalidator.AlsoRequires(path.MatchRoot("workspace"), path.MatchRoot("environment")))
}

// scopeAttributes are the resource attributes shared by wayfinder_variable and wayfinder_secret.
func scopeAttributes() map[string]schema.Attribute {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}

	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: idDescription,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"key": schema.StringAttribute{
			Required: true, MarkdownDescription: keyDescription,
			PlanModifiers: replace, Validators: keyValidators(),
		},
		"workspace": schema.StringAttribute{
			Optional: true, MarkdownDescription: workspaceDescription,
			PlanModifiers: replace, Validators: workspaceValidators(),
		},
		"environment": schema.StringAttribute{
			Optional: true, MarkdownDescription: environmentDescription,
			PlanModifiers: replace, Validators: environmentValidators(),
		},
		"instance": schema.StringAttribute{
			Optional: true, MarkdownDescription: instanceDescription,
			PlanModifiers: replace, Validators: instanceValidators(),
		},
	}
}

// dataSourceScopeAttributes are scopeAttributes for the data source schema.
func dataSourceScopeAttributes() map[string]dsschema.Attribute {
	return map[string]dsschema.Attribute{
		"id":          dsschema.StringAttribute{Computed: true, MarkdownDescription: idDescription},
		"key":         dsschema.StringAttribute{Required: true, MarkdownDescription: "Name of the key to read.", Validators: keyValidators()},
		"workspace":   dsschema.StringAttribute{Optional: true, MarkdownDescription: "Workspace to read from. Omit for a tenant-wide key.", Validators: workspaceValidators()},
		"environment": dsschema.StringAttribute{Optional: true, MarkdownDescription: "Environment (within `workspace`) to read from. Requires `workspace`.", Validators: environmentValidators()},
		"instance":    dsschema.StringAttribute{Optional: true, MarkdownDescription: "Stack instance (within `environment`) to read from. Requires `workspace` and `environment`.", Validators: instanceValidators()},
	}
}

// keyOf builds the scoped key from Terraform values (null means "not set").
func keyOf(key, workspace, environment, instance types.String) scopedKey {
	return scopedKey{
		Workspace: workspace.ValueString(), Environment: environment.ValueString(),
		Instance: instance.ValueString(), Key: key.ValueString(),
	}
}

// optional maps an empty string to null so imported state matches configuration.
func optional(s string) types.String {
	if s == "" {
		return types.StringNull()
	}

	return types.StringValue(s)
}

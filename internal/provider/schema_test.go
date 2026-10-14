package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
)

func TestProviderSchema_whenCLIAttributesDeclared(t *testing.T) {
	// Given the provider schema.
	attrs := providerSchema().Attributes

	// When looking up the CLI session attributes.
	useCLI, okUseCLI := attrs["use_cli"].(schema.BoolAttribute)
	cliProfile, okCLIProfile := attrs["cli_profile"].(schema.StringAttribute)

	// Then use_cli is an optional bool and cli_profile an optional string.
	if !okUseCLI || !useCLI.Optional {
		t.Fatalf("use_cli = %#v, want optional BoolAttribute", attrs["use_cli"])
	}
	if !okCLIProfile || !cliProfile.Optional {
		t.Fatalf("cli_profile = %#v, want optional StringAttribute", attrs["cli_profile"])
	}
}

func TestProviderSchema_whenDescriptionsNameEnvironmentVariable(t *testing.T) {
	// Given the provider schema.
	attrs := providerSchema().Attributes

	// When reading the description of each CLI session attribute.
	for name, env := range map[string]string{"use_cli": "WAYFINDER_USE_CLI", "cli_profile": "WAYFINDER_CLI_PROFILE"} {
		var desc string
		switch a := attrs[name].(type) {
		case schema.BoolAttribute:
			desc = a.MarkdownDescription
		case schema.StringAttribute:
			desc = a.MarkdownDescription
		default:
			t.Fatalf("%s has unexpected attribute type %T", name, attrs[name])
		}

		// Then it ends with the environment variable sentence.
		if want := "Can also be set with the `" + env + "` environment variable."; !strings.HasSuffix(desc, want) {
			t.Fatalf("%s description = %q, want suffix %q", name, desc, want)
		}
	}
}

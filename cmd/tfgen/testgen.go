package main

import (
	"fmt"
	"go/format"
	"strconv"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

// renderTests emits zz_resources_test.go: one plan-stability case per typed kind, driven by the
// hand-written harness in internal/generated (package generated_test). Each case carries the
// kind's fixture configuration (required fields plus every write-only field), the unique
// write-only literals that must never reach plan or state, and a skip reason when no
// validator-compliant fixture could be synthesized.
func renderTests(kinds []manifest.Kind) ([]byte, error) {
	var b strings.Builder
	b.WriteString(header + "package generated_test\n\nvar generatedCases = []generatedCase{\n")
	for _, k := range kinds {
		name := typeName(k)
		identity, id := exampleIdentity(k)
		s := &synth{typ: name, fixture: true}
		config := fmt.Sprintf("resource %q \"example\" {\n%s%s}\n", name, identity, s.body(k.Spec, true, "  ", ""))
		fmt.Fprintf(&b, "{Type: %q, Group: %q, Kind: %q, ImportID: %q,\nConfig: %s,\n", name, k.Group, k.Kind, id, quote(config))
		if len(s.secrets) > 0 {
			fmt.Fprintf(&b, "Secrets: %s,\nWriteOnly: %s,\n", stringSlice(s.secrets), stringSlice(s.writeOnly))
		}
		if len(s.problems) > 0 {
			fmt.Fprintf(&b, "Skip: %q,\n", "cannot synthesize a minimal fixture: "+strings.Join(s.problems, "; "))
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n")
	return format.Source([]byte(b.String()))
}

func quote(s string) string {
	if strings.Contains(s, "`") {
		return strconv.Quote(s)
	}
	return "`" + s + "`"
}

func stringSlice(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = strconv.Quote(item)
	}
	return "[]string{" + strings.Join(quoted, ", ") + "}"
}

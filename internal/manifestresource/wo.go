package manifestresource

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"sigs.k8s.io/yaml"
)

type woProblem struct {
	attr, summary, detail string
}

var (
	yamlLinePattern = regexp.MustCompile(`line (\d+)`)
	woForbiddenKeys = []string{"apiVersion", "kind", "metadata", "status"}
)

// The YAML parser's error is never copied into a problem: it can echo the secret value.
func parseWriteOnly(text string) (map[string]any, []woProblem) {
	docs := 0
	for _, doc := range documentSep.Split(text, -1) {
		if strings.TrimSpace(doc) != "" {
			docs++
		}
	}
	if docs > 1 {
		return nil, []woProblem{{
			attr:    "manifest_wo",
			summary: "Invalid manifest_wo",
			detail:  fmt.Sprintf("manifest_wo holds %d YAML documents; it must be one object.", docs),
		}}
	}
	raw, err := yaml.YAMLToJSON([]byte(text))
	if err != nil {
		detail := "manifest_wo is not valid YAML or JSON"
		if m := yamlLinePattern.FindStringSubmatch(err.Error()); m != nil {
			detail += " (line " + m[1] + ")"
		}
		return nil, []woProblem{{attr: "manifest_wo", summary: "Invalid manifest_wo", detail: detail}}
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return nil, []woProblem{{
			attr:    "manifest_wo",
			summary: "Invalid manifest_wo",
			detail:  "manifest_wo must be one YAML or JSON object.",
		}}
	}
	obj, _ = withoutNulls(obj).(map[string]any)
	var problems []woProblem
	for _, key := range woForbiddenKeys {
		if _, set := obj[key]; set {
			problems = append(problems, woProblem{
				attr:    "manifest_wo",
				summary: "Invalid manifest_wo",
				detail:  fmt.Sprintf("manifest_wo must not set %s; set it in manifest.", key),
			})
		}
	}
	return obj, problems
}

// Package analyze compares user-supplied values against a chart's defaults
// and finds settings that are redundant or unknown to the chart.
package analyze

import (
	"reflect"
	"regexp"
	"sort"
)

// Kind classifies a finding.
type Kind string

const (
	// Redundant means the value equals the chart default.
	Redundant Kind = "redundant"
	// Unknown means the key does not exist in the chart defaults.
	Unknown Kind = "unknown"
)

// Finding is a single reported key.
type Finding struct {
	Path  string `json:"path"`
	Kind  Kind   `json:"-"`
	Value any    `json:"value"`
}

// Analyze walks user and reports every leaf that equals the corresponding
// value in defaults, and every key that defaults doesn't define. Keys below
// an empty or null default map are free-form (e.g. podAnnotations) and are
// never reported as unknown. Findings are sorted by path.
func Analyze(user, defaults map[string]any) []Finding {
	var out []Finding
	walk("", user, defaults, &out)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func walk(prefix string, user, defaults map[string]any, out *[]Finding) {
	for k, uv := range user {
		path := join(prefix, k)
		if uv == nil {
			// null removes a default in Helm; that's intentional, not cruft.
			continue
		}
		dv, exists := defaults[k]
		if !exists {
			*out = append(*out, Finding{Path: path, Kind: Unknown, Value: uv})
			continue
		}
		um, uok := uv.(map[string]any)
		dm, dok := dv.(map[string]any)
		switch {
		case uok && dok:
			if len(dm) == 0 {
				// Free-form map; contents can't be checked.
				if len(um) == 0 {
					*out = append(*out, Finding{Path: path, Kind: Redundant, Value: uv})
				}
				continue
			}
			walk(path, um, dm, out)
		case reflect.DeepEqual(uv, dv):
			*out = append(*out, Finding{Path: path, Kind: Redundant, Value: uv})
		}
	}
}

var plainKey = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// join appends key to a dotted path, quoting keys that aren't plain
// identifiers (e.g. annotation names containing dots).
func join(prefix, key string) string {
	if !plainKey.MatchString(key) {
		return prefix + `["` + key + `"]`
	}
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

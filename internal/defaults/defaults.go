// Package defaults computes the effective default values of a chart,
// including the defaults of all its subcharts.
package defaults

import (
	"helm.sh/helm/v3/pkg/chart"
)

// Build returns the effective default values of c as seen from the parent
// chart's values.yaml: subchart defaults are placed under the dependency's
// alias (or name), with the parent's overrides taking precedence. Globals of
// all subcharts are merged into the top-level "global" key.
//
// Dependency conditions and tags are intentionally ignored, so values for
// disabled subcharts are still recognized.
func Build(c *chart.Chart) map[string]any {
	out := deepCopy(c.Values)

	for _, sub := range c.Dependencies() {
		subDefaults := Build(sub)
		for _, key := range keysFor(c, sub) {
			if parent, ok := out[key].(map[string]any); ok {
				out[key] = merge(parent, deepCopy(subDefaults))
			} else if _, exists := out[key]; !exists {
				out[key] = deepCopy(subDefaults)
			}
			// A non-map parent value (e.g. null) is kept as-is.

			if g, ok := subDefaults["global"].(map[string]any); ok {
				pg, _ := out["global"].(map[string]any)
				if pg == nil {
					pg = map[string]any{}
				}
				out["global"] = merge(pg, deepCopy(g))
			}
		}
	}
	return out
}

// keysFor returns the value keys under which sub is configured in parent:
// one per matching dependency entry (alias or name). Subcharts without a
// dependency entry are keyed by their chart name.
func keysFor(parent *chart.Chart, sub *chart.Chart) []string {
	var keys []string
	if parent.Metadata != nil {
		for _, d := range parent.Metadata.Dependencies {
			if d.Name != sub.Name() {
				continue
			}
			if d.Alias != "" {
				keys = append(keys, d.Alias)
			} else {
				keys = append(keys, d.Name)
			}
		}
	}
	if len(keys) == 0 {
		keys = []string{sub.Name()}
	}
	return keys
}

// merge fills dst with keys from src that dst doesn't have, recursing into
// nested maps. Values in dst win. dst is modified and returned.
func merge(dst, src map[string]any) map[string]any {
	for k, sv := range src {
		dv, exists := dst[k]
		if !exists {
			dst[k] = sv
			continue
		}
		dm, dok := dv.(map[string]any)
		sm, sok := sv.(map[string]any)
		if dok && sok {
			dst[k] = merge(dm, sm)
		}
	}
	return dst
}

func deepCopy(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = copyValue(v)
	}
	return out
}

func copyValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return deepCopy(t)
	case []any:
		s := make([]any, len(t))
		for i, e := range t {
			s[i] = copyValue(e)
		}
		return s
	default:
		return v
	}
}

// Package analyze compares user-supplied values against a chart's defaults
// and finds settings that are redundant or unknown to the chart.
package analyze

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
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

// Options tunes the comparison.
type Options struct {
	// FreeForm lists paths, in the syntax findings use (a.b, a["x.y"]),
	// that are treated as free-form maps: keys at or below them are never
	// reported as unknown. A "*" segment matches any single key.
	FreeForm []string
}

// Validate reports the first malformed FreeForm path, if any.
func (o Options) Validate() error {
	_, err := o.patterns()
	return err
}

func (o Options) patterns() ([][]string, error) {
	patterns := make([][]string, 0, len(o.FreeForm))
	for _, p := range o.FreeForm {
		segs, err := parsePath(p)
		if err != nil {
			return nil, fmt.Errorf("free-form path %q: %w", p, err)
		}
		patterns = append(patterns, segs)
	}
	return patterns, nil
}

// Analyze walks user and reports every leaf that equals the corresponding
// value in defaults, and every key that defaults doesn't define. Keys below
// an empty or null default map, or at or below a path in opts.FreeForm, are
// free-form (e.g. podAnnotations) and are never reported as unknown.
// Findings are sorted by path. A malformed FreeForm path is an error.
func Analyze(user, defaults map[string]any, opts Options) ([]Finding, error) {
	patterns, err := opts.patterns()
	if err != nil {
		return nil, err
	}
	w := walker{patterns: patterns}
	w.walk("", nil, false, user, defaults)
	sort.Slice(w.out, func(i, j int) bool { return w.out[i].Path < w.out[j].Path })
	return w.out, nil
}

type walker struct {
	patterns [][]string
	out      []Finding
}

func (w *walker) walk(prefix string, segs []string, free bool, user, defaults map[string]any) {
	for k, uv := range user {
		path := join(prefix, k)
		if uv == nil {
			// null removes a default in Helm; that's intentional, not cruft.
			continue
		}
		childSegs := append(segs[:len(segs):len(segs)], k)
		childFree := free || matchAny(childSegs, w.patterns)
		dv, exists := defaults[k]
		if !exists {
			if !childFree {
				w.out = append(w.out, Finding{Path: path, Kind: Unknown, Value: uv})
			}
			continue
		}
		um, uok := uv.(map[string]any)
		dm, dok := dv.(map[string]any)
		switch {
		case uok && dok:
			if len(dm) == 0 {
				// Free-form map; contents can't be checked.
				if len(um) == 0 {
					w.out = append(w.out, Finding{Path: path, Kind: Redundant, Value: uv})
				}
				continue
			}
			w.walk(path, childSegs, childFree, um, dm)
		case reflect.DeepEqual(uv, dv):
			w.out = append(w.out, Finding{Path: path, Kind: Redundant, Value: uv})
		}
	}
}

// matchAny reports whether segs matches one of patterns segment by segment;
// a "*" pattern segment matches any key.
func matchAny(segs []string, patterns [][]string) bool {
outer:
	for _, p := range patterns {
		if len(p) != len(segs) {
			continue
		}
		for i := range p {
			if p[i] != "*" && p[i] != segs[i] {
				continue outer
			}
		}
		return true
	}
	return false
}

// parsePath splits a path in the syntax produced by join into its keys,
// e.g. `a["x.y"].b` into [a x.y b].
func parsePath(s string) ([]string, error) {
	var segs []string
	for i := 0; ; {
		var seg string
		if strings.HasPrefix(s[i:], `["`) {
			end := strings.Index(s[i+2:], `"]`)
			if end < 0 {
				return nil, errors.New(`unterminated ["`)
			}
			seg = s[i+2 : i+2+end]
			i += end + 4
		} else {
			end := strings.IndexAny(s[i:], ".[")
			if end < 0 {
				end = len(s) - i
			}
			seg = s[i : i+end]
			i += end
		}
		if seg == "" {
			return nil, errors.New("empty segment")
		}
		segs = append(segs, seg)
		switch {
		case i == len(s):
			return segs, nil
		case s[i] == '[':
		case s[i] == '.' && i+1 < len(s) && s[i+1] != '[':
			i++
		default:
			return nil, fmt.Errorf("unexpected %q at offset %d", s[i], i)
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

package analyze

import (
	"reflect"
	"testing"

	"sigs.k8s.io/yaml"
)

func parse(t *testing.T, s string) map[string]any {
	t.Helper()
	m := map[string]any{}
	if err := yaml.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestAnalyze(t *testing.T) {
	defaults := `
replicas: 1
ratio: 1.0
image: {repository: nginx, tag: "1.25"}
ports: [80, 443]
annotations: {}
extra: null
resources: {limits: {cpu: 1}}
labels: {app: x}
groups: {web: {labels: {app: x}}, db: {labels: {app: y}}}
`
	tests := []struct {
		name     string
		user     string
		want     []Finding
		freeForm []string
	}{
		{"changed scalar", `replicas: 2`, nil, nil},
		{"redundant scalar", `replicas: 1`, []Finding{{"replicas", Redundant, float64(1)}}, nil},
		{"int equals float", `ratio: 1`, []Finding{{"ratio", Redundant, float64(1)}}, nil},
		{"redundant list", `ports: [80, 443]`, []Finding{{"ports", Redundant, []any{float64(80), float64(443)}}}, nil},
		{"reordered list", `ports: [443, 80]`, nil, nil},
		{"nested redundant", `image: {tag: "1.25"}`, []Finding{{"image.tag", Redundant, "1.25"}}, nil},
		{"unknown top-level", `foo: bar`, []Finding{{"foo", Unknown, "bar"}}, nil},
		{"unknown nested", `image: {tagg: x}`, []Finding{{"image.tagg", Unknown, "x"}}, nil},
		{"free-form map", `annotations: {a.b/c: d}`, nil, nil},
		{"empty free-form map", `annotations: {}`, []Finding{{"annotations", Redundant, map[string]any{}}}, nil},
		{"null default", `extra: {anything: goes}`, nil, nil},
		{"null user value", `replicas: null`, nil, nil},
		{"type mismatch", `resources: none`, nil, nil},
		{"quoted path", `image: {"a.b": 1}`, []Finding{{`image["a.b"]`, Unknown, float64(1)}}, nil},
		{"sorted", "zz: 1\nreplicas: 1", []Finding{
			{"replicas", Redundant, float64(1)},
			{"zz", Unknown, float64(1)},
		}, nil},
		{"non-empty map without free-form", `labels: {team: a}`, []Finding{{"labels.team", Unknown, "a"}}, nil},
		{"free-form suppresses unknown", `labels: {team: a}`, nil, []string{"labels"}},
		{"free-form keeps redundant", `labels: {app: x, team: a}`, []Finding{{"labels.app", Redundant, "x"}}, []string{"labels"}},
		{"free-form nested unknown", `labels: {a: {b: 1}}`, nil, []string{"labels"}},
		{"free-form path itself unknown", `custom: {a: 1}`, nil, []string{"custom"}},
		{"free-form leaves siblings", "labels: {team: a}\nfoo: bar", []Finding{{"foo", Unknown, "bar"}}, []string{"labels"}},
		{"wildcard", `groups: {web: {labels: {team: a}}, db: {labels: {team: b}}}`, nil, []string{"groups.*.labels"}},
		{"wildcard is one segment", `groups: {web: {labels: {team: a}}}`,
			[]Finding{{"groups.web.labels.team", Unknown, "a"}}, []string{"*.labels"}},
		{"quoted free-form path", `image: {"a.b": {c: 1}, "a.c": 1}`,
			[]Finding{{`image["a.c"]`, Unknown, float64(1)}}, []string{`image["a.b"]`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Analyze(parse(t, tt.user), parse(t, defaults), Options{FreeForm: tt.freeForm})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v\nwant %#v", got, tt.want)
			}
		})
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		in      string
		want    []string
		wantErr bool
	}{
		{"a", []string{"a"}, false},
		{"a.b.c", []string{"a", "b", "c"}, false},
		{"*.labels", []string{"*", "labels"}, false},
		{`a["x.y"].b`, []string{"a", "x.y", "b"}, false},
		{`["x.y"]`, []string{"x.y"}, false},
		{`a["x"]["y"]`, []string{"a", "x", "y"}, false},
		{"", nil, true},
		{"a.", nil, true},
		{"a..b", nil, true},
		{".a", nil, true},
		{`a["x`, nil, true},
		{`a[""]`, nil, true},
		{`a["x"]b`, nil, true},
	}
	for _, tt := range tests {
		got, err := parsePath(tt.in)
		if (err != nil) != tt.wantErr || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parsePath(%q) = %q, %v; want %q, err=%v", tt.in, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestAnalyzeBadFreeForm(t *testing.T) {
	if _, err := Analyze(nil, nil, Options{FreeForm: []string{`a["b`}}); err == nil {
		t.Fatal("want error for malformed path")
	}
}

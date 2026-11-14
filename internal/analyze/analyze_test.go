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
`
	tests := []struct {
		name string
		user string
		want []Finding
	}{
		{"changed scalar", `replicas: 2`, nil},
		{"redundant scalar", `replicas: 1`, []Finding{{"replicas", Redundant, float64(1)}}},
		{"int equals float", `ratio: 1`, []Finding{{"ratio", Redundant, float64(1)}}},
		{"redundant list", `ports: [80, 443]`, []Finding{{"ports", Redundant, []any{float64(80), float64(443)}}}},
		{"reordered list", `ports: [443, 80]`, nil},
		{"nested redundant", `image: {tag: "1.25"}`, []Finding{{"image.tag", Redundant, "1.25"}}},
		{"unknown top-level", `foo: bar`, []Finding{{"foo", Unknown, "bar"}}},
		{"unknown nested", `image: {tagg: x}`, []Finding{{"image.tagg", Unknown, "x"}}},
		{"free-form map", `annotations: {a.b/c: d}`, nil},
		{"empty free-form map", `annotations: {}`, []Finding{{"annotations", Redundant, map[string]any{}}}},
		{"null default", `extra: {anything: goes}`, nil},
		{"null user value", `replicas: null`, nil},
		{"type mismatch", `resources: none`, nil},
		{"quoted path", `image: {"a.b": 1}`, []Finding{{`image["a.b"]`, Unknown, float64(1)}}},
		{"sorted", "zz: 1\nreplicas: 1", []Finding{
			{"replicas", Redundant, float64(1)},
			{"zz", Unknown, float64(1)},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Analyze(parse(t, tt.user), parse(t, defaults))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v\nwant %#v", got, tt.want)
			}
		})
	}
}

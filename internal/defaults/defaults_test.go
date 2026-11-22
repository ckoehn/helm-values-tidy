package defaults

import (
	"testing"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestBuild(t *testing.T) {
	c, err := loader.Load("../../testdata/parent")
	if err != nil {
		t.Fatal(err)
	}
	d := chartutil.Values(Build(c))

	tests := []struct {
		path string
		want any
	}{
		{"replicaCount", float64(1)},
		{"db.port", float64(5433)},          // parent overrides subchart default
		{"db.user", "admin"},                // subchart default under alias
		{"db.enabled", false},               // parent-only key
		{"global.region", "eu"},             // parent global
		{"global.storageClass", "standard"}, // subchart global
		{"db.labels.app", "child"},          // subchart map default under alias
	}
	for _, tt := range tests {
		got, err := d.PathValue(tt.path)
		if err != nil {
			t.Errorf("%s: %v", tt.path, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s = %v, want %v", tt.path, got, tt.want)
		}
	}

	if _, ok := d["child"]; ok {
		t.Error("aliased subchart should not appear under its chart name")
	}
	if ns, _ := d["nodeSelector"].(map[string]any); ns["kubernetes.io/os"] != "linux" {
		t.Errorf("nodeSelector = %v, want kubernetes.io/os: linux", d["nodeSelector"])
	}
	if _, ok := d["extra"]; !ok {
		t.Error("null default key should be kept")
	}
}

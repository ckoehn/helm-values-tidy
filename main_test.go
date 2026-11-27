package main

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	withVersion := func(v string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: v}}
	}
	tests := []struct {
		name    string
		ldflags string
		info    *debug.BuildInfo
		want    string
	}{
		{"ldflags wins", "v0.1.0", withVersion("v0.0.9"), "v0.1.0"},
		{"module version", "", withVersion("v0.1.0"), "v0.1.0"},
		{"devel", "", withVersion("(devel)"), "dev"},
		{"empty module version", "", withVersion(""), "dev"},
		{"no build info", "", nil, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.ldflags, tt.info); got != tt.want {
				t.Errorf("resolveVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunPrintsVersion(t *testing.T) {
	for _, flag := range []string{"-V", "--tool-version"} {
		var out bytes.Buffer
		// No CHART or -f needed: the version flag short-circuits validation.
		if err := run([]string{flag}, &out); err != nil {
			t.Fatalf("run(%s): %v", flag, err)
		}
		if !strings.HasPrefix(out.String(), "helm-values-tidy ") {
			t.Errorf("run(%s) output = %q", flag, out.String())
		}
	}
}

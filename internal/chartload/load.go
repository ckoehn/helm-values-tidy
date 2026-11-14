// Package chartload resolves a chart reference (local path, repo/name, URL
// or oci://) and loads it, including its packaged subcharts.
package chartload

import (
	"fmt"
	"io"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
)

// Options controls how a chart reference is resolved.
type Options struct {
	Version string
	RepoURL string
	// Log receives download progress output. Defaults to stderr.
	Log io.Writer
}

// Load locates the chart named by ref (downloading it if necessary) and loads it.
func Load(ref string, opts Options) (*chart.Chart, error) {
	if opts.Log == nil {
		opts.Log = os.Stderr
	}
	settings := cli.New()

	rc, err := registry.NewClient(
		registry.ClientOptWriter(opts.Log),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
		registry.ClientOptEnableCache(true),
	)
	if err != nil {
		return nil, fmt.Errorf("creating registry client: %w", err)
	}

	// Install is only used as a vehicle for ChartPathOptions, whose registry
	// client can't be set directly.
	inst := action.NewInstall(&action.Configuration{RegistryClient: rc})
	inst.Version = opts.Version
	inst.RepoURL = opts.RepoURL

	path, err := inst.LocateChart(ref, settings)
	if err != nil {
		return nil, fmt.Errorf("locating chart %q: %w", ref, err)
	}
	c, err := loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("loading chart %q: %w", path, err)
	}
	return c, nil
}

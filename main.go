// Command helm-values-tidy reports settings in Helm values files that either equal
// the chart's defaults or aren't defined by the chart at all.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	flag "github.com/spf13/pflag"
	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/ckoehn/helm-values-tidy/internal/analyze"
	"github.com/ckoehn/helm-values-tidy/internal/chartload"
	"github.com/ckoehn/helm-values-tidy/internal/defaults"
	"github.com/ckoehn/helm-values-tidy/internal/report"
)

// version is set at build time by release builds with
// -ldflags "-X main.version=v1.2.3". When empty, it is taken from the module
// build info (see toolVersion).
var version string

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("helm-values-tidy", flag.ContinueOnError)
	fs.SortFlags = false
	fs.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "helm-values-tidy %s\n\nUsage: helm-values-tidy [flags] CHART\n\n"+
			"CHART is a local directory, a .tgz, repo/name, a URL or oci://...\n\nFlags:\n", toolVersion())
		fs.PrintDefaults()
	}
	var (
		valueFiles []string
		version    string
		printVer   bool
		repoURL    string
		output     string
		freeForm   []string
	)
	fs.StringArrayVarP(&valueFiles, "values", "f", nil, "values `file` (repeatable, merged in order)")
	fs.StringVar(&version, "version", "", "chart version constraint")
	fs.StringVar(&repoURL, "repo", "", "chart repository `URL`")
	fs.StringVarP(&output, "output", "o", "text", "output `format`: text or json")
	fs.StringArrayVar(&freeForm, "free-form", nil, "treat `PATH` as a free-form map: keys at or below it are never unknown\n(repeatable, * matches one key, e.g. '*.labels')")
	fs.BoolVarP(&printVer, "tool-version", "V", false, "print the helm-values-tidy version and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if printVer {
		_, err := fmt.Fprintln(stdout, "helm-values-tidy", toolVersion())
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return errors.New("exactly one CHART argument is required")
	}
	if len(valueFiles) == 0 {
		return errors.New("at least one values file (-f) is required")
	}
	if output != "text" && output != "json" {
		return fmt.Errorf("unknown output format %q", output)
	}
	opts := analyze.Options{FreeForm: freeForm}
	if err := opts.Validate(); err != nil {
		return err
	}

	user := map[string]any{}
	for _, f := range valueFiles {
		vals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			return fmt.Errorf("reading %s: %w", f, err)
		}
		user = mergeOver(user, vals.AsMap())
	}

	c, err := chartload.Load(fs.Arg(0), chartload.Options{Version: version, RepoURL: repoURL})
	if err != nil {
		return err
	}

	findings, err := analyze.Analyze(user, defaults.Build(c), opts)
	if err != nil {
		return err
	}
	r := report.New(c.Name()+"@"+c.Metadata.Version, findings)
	if output == "json" {
		return report.WriteJSON(stdout, r)
	}
	return report.WriteText(stdout, r)
}

// toolVersion returns the version of this binary: the one set with -ldflags,
// else the module version recorded by the Go toolchain (set by
// `go install ...@v1.2.3`, or derived from VCS for local builds), else "dev".
func toolVersion() string {
	info, _ := debug.ReadBuildInfo()
	return resolveVersion(version, info)
}

func resolveVersion(ldflags string, info *debug.BuildInfo) string {
	if ldflags != "" {
		return ldflags
	}
	if info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// mergeOver deep-merges src into dst, with src winning, like Helm does for
// successive -f files.
func mergeOver(dst, src map[string]any) map[string]any {
	for k, sv := range src {
		if sm, ok := sv.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				dst[k] = mergeOver(dm, sm)
				continue
			}
		}
		dst[k] = sv
	}
	return dst
}

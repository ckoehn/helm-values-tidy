// Command helm-values-tidy reports settings in Helm values files that either equal
// the chart's defaults or aren't defined by the chart at all.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/ckoehn/helm-values-tidy/internal/analyze"
	"github.com/ckoehn/helm-values-tidy/internal/chartload"
	"github.com/ckoehn/helm-values-tidy/internal/defaults"
	"github.com/ckoehn/helm-values-tidy/internal/report"
)

type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("helm-values-tidy", flag.ContinueOnError)
	fs.Usage = func() {
		_, _ = fmt.Fprint(fs.Output(), "Usage: helm-values-tidy [flags] CHART\n\n"+
			"CHART is a local directory, a .tgz, repo/name, a URL or oci://...\n\nFlags:\n")
		fs.PrintDefaults()
	}
	var (
		valueFiles stringSlice
		version    string
		repoURL    string
		output     string
	)
	fs.Var(&valueFiles, "f", "values file (repeatable, merged in order)")
	fs.Var(&valueFiles, "values", "values file (repeatable, merged in order)")
	fs.StringVar(&version, "version", "", "chart version constraint")
	fs.StringVar(&repoURL, "repo", "", "chart repository URL")
	fs.StringVar(&output, "o", "text", "output format: text or json")
	fs.StringVar(&output, "output", "text", "output format: text or json")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
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

	r := report.New(c.Name()+"@"+c.Metadata.Version, analyze.Analyze(user, defaults.Build(c)))
	if output == "json" {
		return report.WriteJSON(os.Stdout, r)
	}
	return report.WriteText(os.Stdout, r)
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

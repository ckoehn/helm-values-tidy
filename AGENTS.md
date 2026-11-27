# AGENTS.md

Guidance for coding agents working in this repository.

## What this is

`helm-values-tidy` is a Go CLI that compares a user's Helm values file(s)
against a chart's effective defaults and reports:

- **Redundant** – keys whose value equals the chart default
- **Unknown** – keys the chart doesn't define

```
helm-values-tidy [-f values.yaml ...] [--version V] [--repo URL] [-o text|json] [--free-form PATH ...] CHART
```

`CHART` can be a local directory, `.tgz`, `repo/name`, URL or `oci://…`.

## Layout

| Path | Purpose |
| --- | --- |
| `main.go` | Flag parsing (`spf13/pflag`, already a Helm dependency: `-x` short, `--long` long), merging `-f` files, wiring |
| `internal/chartload` | Resolve and load a chart via the Helm v3 SDK (`action.ChartPathOptions.LocateChart`, `loader.Load`) |
| `internal/defaults` | Build the effective default tree, including subcharts and globals |
| `internal/analyze` | Recursive comparison producing sorted `Finding`s |
| `internal/report` | Text and JSON rendering |
| `testdata/parent` | Test chart with an aliased subchart (`child` as `db`) |
| `testdata/values.yaml` | Values exercising every rule; comments mark expected results |
| `testdata/nginx-values.yaml` | Values for `make demo-remote` (bitnami/nginx at `REMOTE_VERSION`); comments mark expected results |
| `.goreleaser.yaml` | Prebuilt binaries for releases (linux/darwin/windows × amd64/arm64); `main.version` set via ldflags |
| `.github/workflows/release.yml` | On `v*` tags: reruns `ci.yml`, runs GoReleaser, attests build provenance |
| `scripts/release-notes.sh` | Extracts a version's `CHANGELOG.md` section as release notes |

## Commands

Use the Makefile (`make help` lists everything):

- `make check` – vet, golangci-lint, tests and gofmt check; run before finishing a change
- `make test` / `make cover`
- `make demo` – run against `testdata/parent` with `DEMO_FLAGS` (`--free-form`
  paths); output should match the comments in `testdata/values.yaml`
- `make demo-remote` – run `testdata/nginx-values.yaml` against bitnami/nginx
  pinned to `REMOTE_VERSION` (needs network); output should match that file's
  comments. If you bump the version, re-check the comments and the README
  "Example" section, which shows this output.
- `make build` – binary in `bin/`
- `make snapshot` / `make release-check` – GoReleaser dry run into `dist/` /
  config validation (GoReleaser pinned via `GORELEASER_VERSION`, keep in sync
  with `release.yml`)
- `make release-notes VERSION=X.Y.Z` – the release notes for a version

## Comparison rules (keep these stable)

- Key missing from defaults → Unknown, **except** below a default that is an
  empty map `{}` or `null` (free-form maps like `podAnnotations`) → ignored.
- Both maps → recurse. Empty user map over empty default map → Redundant.
- Paths given with `--free-form` (`analyze.Options.FreeForm`; same syntax as
  finding paths, `*` matches one key) behave like an empty default map at and
  below them: never Unknown, but Redundant is still reported.
- User value `null` → ignored (Helm's "remove default" idiom).
- Otherwise `reflect.DeepEqual`; lists compare as a whole, order matters.
- Type mismatch (map vs scalar) → not reported.
- Subchart defaults live under the dependency alias (or name); parent values
  override them. Subchart `global`s are merged into top-level `global`
  (parent wins). Conditions/tags are deliberately **not** applied.

## Conventions

- All YAML is parsed with `sigs.k8s.io/yaml` (chart loader and
  `chartutil.ReadValuesFile` do this), so numbers are `float64` on both
  sides. Don't introduce another YAML parser for values, or int/float
  comparisons break.
- Report data goes to stdout; anything else (download progress, warnings)
  goes to stderr so `-o json` stays parseable.
- Findings don't change the exit code; only errors exit 1.
- Use the Helm **v3** SDK (`helm.sh/helm/v3`), even if the local `helm` CLI is v4.
- Behaviour changes need a case in `internal/analyze/analyze_test.go` (table
  test) or `internal/defaults/defaults_test.go`, and `testdata/values.yaml`
  comments updated if the demo output changes.
- User-visible changes (flags, comparison rules, output) get an entry under
  `## [Unreleased]` in `CHANGELOG.md` (Keep a Changelog sections: Added,
  Changed, Fixed, Removed). Releasing is described in the README
  ("Releasing"); don't tag or push releases unless asked.
- Match existing style: small packages, doc comments on exported identifiers,
  no extra dependencies without good reason.

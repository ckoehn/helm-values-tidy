# helm-values-tidy

[![CI](https://github.com/ckoehn/helm-values-tidy/actions/workflows/ci.yml/badge.svg)](https://github.com/ckoehn/helm-values-tidy/actions/workflows/ci.yml)

Find the dead weight in your Helm values files.

`helm-values-tidy` compares your `values.yaml` against the defaults of the chart
it's used with and reports:

- **Redundant** settings – values that are exactly the chart default and can be removed
- **Unknown** settings – keys the chart doesn't define at all (typos, leftovers from older chart versions)

It works with local and remote charts and understands subcharts, aliases and globals.

## Installation

### Prebuilt binaries

Download the archive for your platform (Linux, macOS or Windows; amd64 or arm64)
from the [releases page](https://github.com/ckoehn/helm-values-tidy/releases),
unpack it and put `helm-values-tidy` somewhere on your `PATH`. For example:

```sh
VERSION=0.1.0 OS=linux ARCH=amd64   # OS: linux or darwin, ARCH: amd64 or arm64
curl -fsSLO "https://github.com/ckoehn/helm-values-tidy/releases/download/v${VERSION}/helm-values-tidy_${VERSION}_${OS}_${ARCH}.tar.gz"
tar -xzf "helm-values-tidy_${VERSION}_${OS}_${ARCH}.tar.gz" helm-values-tidy
sudo install helm-values-tidy /usr/local/bin/
```

Each release has a `checksums.txt` and build provenance attestations, so you can
check a download before using it:

```sh
sha256sum --ignore-missing -c checksums.txt        # macOS: shasum -a 256 --ignore-missing -c checksums.txt
gh attestation verify helm-values-tidy_*.tar.gz --repo ckoehn/helm-values-tidy
```

The macOS binaries aren't signed by Apple. If you downloaded one with a browser and
macOS refuses to run it, remove the quarantine flag:
`xattr -d com.apple.quarantine helm-values-tidy`.

### With Go

Requires Go 1.27 or newer.

```sh
go install github.com/ckoehn/helm-values-tidy@latest
```

Or from a checkout of this repository:

```sh
make install     # installs into $GOBIN (or $GOPATH/bin)
# or
make build       # builds ./bin/helm-values-tidy
```

## Usage

```
helm-values-tidy [flags] CHART

  -f, --values FILE     values file (repeatable, merged in order like helm)
      --version VER     chart version constraint
      --repo URL        chart repository URL
  -o, --output FMT      text (default) or json
      --free-form PATH  treat PATH as a free-form map: keys at or below it are
                        never unknown (repeatable, * matches one key)
  -V, --tool-version    print the helm-values-tidy version and exit
```

Short flags take one dash (`-f`), long flags two (`--values`). Flags may also
come after `CHART`. `--version` is the *chart* version; use `-V` for the version
of helm-values-tidy itself.

`CHART` can be anything `helm install` accepts:

```sh
helm-values-tidy -f values.yaml ./my-chart                    # local directory
helm-values-tidy -f values.yaml my-chart-1.2.3.tgz            # packaged chart
helm-values-tidy -f values.yaml bitnami/nginx --version 25.2.0  # repo from `helm repo add`
helm-values-tidy -f values.yaml nginx --repo https://charts.bitnami.com/bitnami
helm-values-tidy -f values.yaml oci://registry-1.docker.io/bitnamicharts/nginx
```

Repository and registry credentials are read from your regular Helm configuration.

### Example

Using [`testdata/nginx-values.yaml`](testdata/nginx-values.yaml) (this is what
`make demo-remote` runs):

```console
$ helm-values-tidy -f testdata/nginx-values.yaml --version 25.2.16 \
    --free-form service.ports oci://registry-1.docker.io/bitnamicharts/nginx
Chart: nginx@25.2.16

Redundant (equal to chart default): 5
  containerPorts.http = 8080
  image.repository = bitnami/nginx
  livenessProbe.periodSeconds = 10
  service.ports.http = 80
  service.type = LoadBalancer

Unknown (not defined by chart): 2
  legacyKey = 1
  readinessProbe.peroidSeconds = 5
```

With `-o json` (shortened):

```json
{
  "chart": "nginx@25.2.16",
  "redundant": [
    { "path": "containerPorts.http", "value": 8080 },
    { "path": "image.repository", "value": "bitnami/nginx" },
    ...
  ],
  "unknown": [
    { "path": "legacyKey", "value": 1 },
    { "path": "readinessProbe.peroidSeconds", "value": 5 }
  ]
}
```

Report data goes to stdout and download progress to stderr, so the JSON output
can be piped straight into `jq`. The exit code is `0` on success, regardless of
findings, and `1` on errors.

## How values are compared

| Situation | Result |
| --- | --- |
| Value equals the chart default | Redundant |
| Key doesn't exist in the chart defaults | Unknown |
| Key below a default that is `{}` or `null` (e.g. `podAnnotations`, `nodeSelector`) | Ignored – free-form map |
| Key at or below a `--free-form` path that the chart doesn't define | Ignored – free-form map |
| Your value is `null` | Ignored – Helm's way of removing a default |
| Map vs. scalar type mismatch | Ignored |

- Lists are compared as a whole; order matters.
- `1` and `1.0` are considered equal.
- Keys containing dots are shown quoted, e.g. `podAnnotations["prometheus.io/scrape"]`.

### Free-form maps with defaults

Some charts ship a *non-empty* default for a map you're meant to extend, e.g.
`nodeSelector: {kubernetes.io/os: linux}` or `commonLabels: {app: foo}`. Every key
you add there would show up as unknown. Mark such maps with `--free-form`:

```sh
helm-values-tidy -f values.yaml --free-form nodeSelector --free-form '*.labels' ./my-chart
```

- Keys at or below a free-form path are never reported as unknown, at any depth.
- Keys that equal the chart default are still reported as redundant.
- Paths use the same syntax as the output: `a.b`, and `a["x.y"]` for keys with dots.
- A `*` segment matches exactly one key, so `*.labels` matches `db.labels` but
  not `db.web.labels` or the top-level `labels`.
- The flag can be repeated. A malformed path is an error.

### Subcharts

- Subchart values are checked under the dependency's alias (or name).
- If the parent chart overrides a subchart default, the override is the default.
- `global` values from all subcharts are recognized.
- Dependency `condition`s and `tags` are ignored, so settings for currently
  disabled subcharts are not reported as unknown.

## Development

```sh
make help        # list all targets
make check       # vet, lint, tests, gofmt check
make test        # unit tests
make cover       # tests with coverage summary
make demo        # run against the bundled test chart (with DEMO_FLAGS)
make demo-remote # run against a pinned bitnami/nginx (needs network)
make build       # binary in ./bin
make snapshot    # release archives in ./dist (GoReleaser dry run)
```

`testdata/parent` is a small chart with an aliased subchart, and
`testdata/values.yaml` exercises every comparison rule – its comments describe the
expected result for each key when run with the `DEMO_FLAGS` from the Makefile
(`--free-form nodeSelector --free-form '*.labels'`). Run `make demo DEMO_FLAGS=` to
see the same file without them.

`make demo-remote` checks `testdata/nginx-values.yaml` against
`bitnami/nginx` pinned to `REMOTE_VERSION`, with `REMOTE_FLAGS`
(`--free-form service.ports`). That file's comments describe the expected
results for that chart version. When you bump `REMOTE_VERSION`, re-check them and the
"Example" output above.

See [CHANGELOG.md](CHANGELOG.md) for notable changes and
[AGENTS.md](AGENTS.md) for architecture notes
and conventions.

### Releasing

Releases are built by the [Release workflow](.github/workflows/release.yml) when a
`v*` tag is pushed. It reruns CI, builds the binaries with GoReleaser
([`.goreleaser.yaml`](.goreleaser.yaml)) and publishes a GitHub release whose notes
are the version's section from `CHANGELOG.md`.

1. Run `make check`, `make demo`, `make demo-remote` and `make snapshot`, and try a
   binary from `dist/`.
2. In `CHANGELOG.md`, rename `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD`, add a
   new empty `## [Unreleased]` above it and update the links at the bottom.
   `make release-notes VERSION=X.Y.Z` shows what the release page will say (the
   workflow fails if that section is missing).
3. Commit, push to `main` and wait for CI.
4. Tag and push: `git tag -a vX.Y.Z -m vX.Y.Z && git push origin vX.Y.Z`.
5. Check the release page, `go install github.com/ckoehn/helm-values-tidy@vX.Y.Z`
   and `helm-values-tidy -V`.

If the workflow fails, delete the tag (`git push --delete origin vX.Y.Z` and
`git tag -d vX.Y.Z`) and any partial release, fix the problem and tag again. Tags
like `vX.Y.Z-rc.1` are published as pre-releases.

When bumping GoReleaser, change `GORELEASER_VERSION` in the Makefile and `version`
in the workflow together.

## License

[MIT](LICENSE)

# helm-values-tidy

[![CI](https://github.com/ckoehn/helm-values-tidy/actions/workflows/ci.yml/badge.svg)](https://github.com/ckoehn/helm-values-tidy/actions/workflows/ci.yml)

Find the dead weight in your Helm values files.

`helm-values-tidy` compares your `values.yaml` against the defaults of the chart
it's used with and reports:

- **Redundant** settings – values that are exactly the chart default and can be removed
- **Unknown** settings – keys the chart doesn't define at all (typos, leftovers from older chart versions)

It works with local and remote charts and understands subcharts, aliases and globals.

## Installation

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

  -f, -values FILE    values file (repeatable, merged in order like helm)
  -version VER        chart version constraint
  -repo URL           chart repository URL
  -o, -output FMT     text (default) or json
```

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

```console
$ helm-values-tidy -f values.yaml oci://registry-1.docker.io/bitnamicharts/nginx
Chart: nginx@25.2.14

Redundant (equal to chart default): 4
  containerPorts.http = 8080
  containerPorts.https = 8443
  replicaCount = 1
  service.type = LoadBalancer

Unknown (not defined by chart): 2
  nonExistent = foo
  service.typo = true
```

With `-o json`:

```json
{
  "chart": "nginx@25.2.14",
  "redundant": [{ "path": "replicaCount", "value": 1 }],
  "unknown": [{ "path": "nonExistent", "value": "foo" }]
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
| Your value is `null` | Ignored – Helm's way of removing a default |
| Map vs. scalar type mismatch | Ignored |

- Lists are compared as a whole; order matters.
- `1` and `1.0` are considered equal.
- Keys containing dots are shown quoted, e.g. `podAnnotations["prometheus.io/scrape"]`.

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
make demo        # run against the bundled test chart
make build       # binary in ./bin
```

`testdata/parent` is a small chart with an aliased subchart, and
`testdata/values.yaml` exercises every comparison rule – its comments describe the
expected result for each key. See [AGENTS.md](AGENTS.md) for architecture notes
and conventions.

## License

[MIT](LICENSE)

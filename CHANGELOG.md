# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Report values that equal the chart default (**redundant**) and keys the chart
  doesn't define (**unknown**), as text or JSON (`-o, --output`).
- Load charts from a local directory, `.tgz`, `repo/name`, URL or `oci://`, with
  `--version` and `--repo`.
- Merge multiple values files (`-f, --values`, repeatable) the same way Helm does.
- Check subchart values (including aliases) and subchart `global`s.
- Treat maps whose chart default is `{}` or `null` as free-form, so their keys are
  never reported as unknown.
- `--free-form PATH` (repeatable) to treat a map as free-form even when its chart
  default isn't empty. Keys at or below it are never unknown, but values that equal
  the default are still reported as redundant. A `*` segment matches any single key,
  e.g. `--free-form '*.labels'`.
- `-V, --tool-version` prints the helm-values-tidy version (`--version` is
  already the chart version). The version is also shown in `--help`.
- Prebuilt binaries for Linux, macOS and Windows (amd64 and arm64) attached to
  each GitHub release, with checksums and build provenance attestations.

### Changed

- Command-line flags follow the usual convention: short flags take one dash (`-f`,
  `-o`) and long flags take two (`--values`, `--output`, `--version`, `--repo`).
  Flags may also come after `CHART`. The single-dash long forms (e.g. `-values`) are
  no longer accepted.

[Unreleased]: https://github.com/ckoehn/helm-values-tidy/commits/main

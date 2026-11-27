#!/bin/sh
# Print the CHANGELOG.md section for a version, without its heading, for use
# as GitHub release notes. Fails if the section is missing or empty, so a
# release can't go out before `## [Unreleased]` was renamed.
#
# Usage: scripts/release-notes.sh 0.1.0 [CHANGELOG.md]
set -eu

version=${1:?usage: $0 VERSION [CHANGELOG]}
version=${version#v}
changelog=${2:-CHANGELOG.md}

notes=$(awk -v v="$version" '
	/^## \[/ { if (found) exit; if (index($0, "## [" v "]") == 1) { found = 1; next } }
	/^\[[^]]+\]: / { if (found) exit }
	found { print }
' "$changelog" | sed -e '/./,$!d')

if [ -z "$(printf '%s' "$notes" | tr -d '[:space:]')" ]; then
	echo "error: no section for version $version in $changelog" >&2
	exit 1
fi
printf '%s\n' "$notes"

#!/usr/bin/env bash

set -euo pipefail

usage() {
  printf 'usage: %s <git-ref> <full-commit> <repository>\n' "${0##*/}" >&2
  exit 2
}

[[ "$#" -eq 3 ]] || usage

git_ref="$1"
commit="$2"
repository="$3"

[[ "${commit}" =~ ^[0-9a-f]{40}$ ]] || {
  printf 'invalid commit: expected a full lowercase 40-character SHA\n' >&2
  exit 1
}
[[ "${repository}" == "harmonycloud/saola-cli" ]] || {
  printf 'invalid repository: expected harmonycloud/saola-cli\n' >&2
  exit 1
}

case "${git_ref}" in
  refs/heads/main)
    channel="dev"
    version="dev-${commit:0:12}"
    ;;
  refs/tags/v*)
    version="${git_ref#refs/tags/}"
    semver='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?$'
    [[ "${version}" =~ ${semver} ]] || {
      printf 'invalid release tag: expected vMAJOR.MINOR.PATCH with optional SemVer prerelease\n' >&2
      exit 1
    }
    channel="stable"
    ;;
  *)
    printf 'invalid git ref: only refs/heads/main and SemVer tags are releasable\n' >&2
    exit 1
    ;;
esac

source_date_epoch="${SOURCE_DATE_EPOCH:-}"
if [[ -z "${source_date_epoch}" ]]; then
  source_date_epoch="$(git show -s --format=%ct "${commit}" 2>/dev/null)" || {
    printf 'SOURCE_DATE_EPOCH is unset and commit timestamp cannot be resolved\n' >&2
    exit 1
  }
fi
[[ "${source_date_epoch}" =~ ^(0|[1-9][0-9]*)$ ]] || {
  printf 'invalid SOURCE_DATE_EPOCH: expected a non-negative integer\n' >&2
  exit 1
}

if build_date="$(date -u -d "@${source_date_epoch}" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null)"; then
  :
elif build_date="$(date -u -r "${source_date_epoch}" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null)"; then
  :
else
  printf 'unable to format SOURCE_DATE_EPOCH\n' >&2
  exit 1
fi

printf 'channel=%s\n' "${channel}"
printf 'version=%s\n' "${version}"
printf 'commit=%s\n' "${commit}"
printf 'build_date=%s\n' "${build_date}"

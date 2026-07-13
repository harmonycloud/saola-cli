#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
helper="${script_dir}/release-metadata.sh"
workflow="${script_dir}/../.github/workflows/release.yml"
commit="0123456789abcdef0123456789abcdef01234567"
epoch="1782812176"

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local output="$1"
  local expected="$2"
  [[ "${output}" == *"${expected}"* ]] || fail "expected output to contain ${expected}, got: ${output}"
}

assert_rejected() {
  if "${helper}" "$@" >/dev/null 2>&1; then
    fail "expected rejection for arguments: $*"
  fi
}

main_output="$(SOURCE_DATE_EPOCH="${epoch}" "${helper}" refs/heads/main "${commit}" harmonycloud/saola-cli)"
assert_contains "${main_output}" "channel=dev"
assert_contains "${main_output}" "version=dev-0123456789ab"
assert_contains "${main_output}" "commit=${commit}"
assert_contains "${main_output}" "build_date=2026-06-30T09:36:16Z"

tag_output="$(SOURCE_DATE_EPOCH="${epoch}" "${helper}" refs/tags/v1.2.3-rc.1 "${commit}" harmonycloud/saola-cli)"
assert_contains "${tag_output}" "channel=stable"
assert_contains "${tag_output}" "version=v1.2.3-rc.1"

repeat_output="$(SOURCE_DATE_EPOCH="${epoch}" "${helper}" refs/heads/main "${commit}" harmonycloud/saola-cli)"
[[ "${main_output}" == "${repeat_output}" ]] || fail "metadata output is not deterministic"

assert_rejected refs/heads/dev "${commit}" harmonycloud/saola-cli
assert_rejected refs/tags/latest "${commit}" harmonycloud/saola-cli
assert_rejected refs/tags/v1.2 "${commit}" harmonycloud/saola-cli
assert_rejected refs/heads/main 0123456789abcdef harmonycloud/saola-cli
assert_rejected refs/heads/main "${commit}" 'harmonycloud/saola-cli;echo-owned'
assert_rejected refs/heads/main "${commit}" other/saola-cli
if SOURCE_DATE_EPOCH=not-an-epoch "${helper}" refs/heads/main "${commit}" harmonycloud/saola-cli >/dev/null 2>&1; then
  fail "expected rejection for an invalid SOURCE_DATE_EPOCH"
fi

grep -Fq -- '--draft' "${workflow}" || fail "stable releases must be created as drafts"
grep -Fq 'gh release download' "${workflow}" || fail "existing assets must be downloaded before comparison"
grep -Fq 'cmp -s' "${workflow}" || fail "existing assets must be compared byte-for-byte"
grep -Fq 'cosign verify-blob' "${workflow}" || fail "existing signature bundles must be cryptographically verified"
grep -Fq -- '--certificate-identity' "${workflow}" || fail "bundle verification must bind the workflow identity"
grep -Fq 'sigstore.json' "${workflow}" || fail "bundle replay handling is missing"
grep -Fq 'canonical_sbom' "${workflow}" || fail "existing SBOM is not treated as the canonical release asset"
grep -Fq 'spdxVersion' "${workflow}" || fail "canonical SBOM structure is not validated"
grep -Fq 'sign-blob --yes --bundle' "${workflow}" || fail "missing canonical SBOM bundle cannot be regenerated"
grep -Fq -- '--draft=false' "${workflow}" || fail "draft releases must be published only after verification"
if grep -Fq -- '--clobber' "${workflow}"; then
  fail "stable release assets must never be overwritten with --clobber"
fi

uses_count="$(grep -Ec '^[[:space:]]+(- )?uses:' "${workflow}")"
pinned_uses_count="$(grep -Ec '^[[:space:]]+(- )?uses: [^@[:space:]]+@[0-9a-f]{40}[[:space:]]+# v[^[:space:]]+$' "${workflow}")"
[[ "${uses_count}" -gt 0 && "${uses_count}" -eq "${pinned_uses_count}" ]] || {
  fail "all workflow actions must be pinned to a full 40-character commit SHA with a version comment"
}

printf 'PASS: release metadata contract\n'

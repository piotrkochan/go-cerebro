#!/usr/bin/env bash

set -euo pipefail

tag="${1:-}"
output="${2:-dist/release-notes.md}"
changes_file="${CHANGES_FILE:-CHANGES.md}"
cliff_config="${GIT_CLIFF_CONFIG:-cliff.toml}"

if [[ ! "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.+)?$ ]]; then
  echo "usage: $0 vX.Y.Z[-prerelease] [output]" >&2
  exit 2
fi
if ! git rev-parse --verify --quiet "refs/tags/${tag}" >/dev/null; then
  echo "tag ${tag} does not exist" >&2
  exit 2
fi
if [[ ! -f "${changes_file}" ]]; then
  echo "${changes_file} does not exist" >&2
  exit 2
fi
if [[ ! -f "${cliff_config}" ]]; then
  echo "${cliff_config} does not exist" >&2
  exit 2
fi
if ! command -v git-cliff >/dev/null; then
  echo "git-cliff is required" >&2
  exit 2
fi

base_version="${tag%%-*}"
release_date="$(git for-each-ref --format='%(creatordate:short)' "refs/tags/${tag}")"
manual_notes="$(mktemp)"
chore_notes="$(mktemp)"
raw_chore_notes="$(mktemp)"
trap 'rm -f "${manual_notes}" "${chore_notes}" "${raw_chore_notes}"' EXIT

awk -v version="${base_version}" '
  BEGIN {
    heading = "## " version
    found = 0
    copy = 0
  }
  index($0, heading) == 1 &&
    (length($0) == length(heading) || substr($0, length(heading) + 1, 3) == " - ") {
    found = 1
    copy = 1
    next
  }
  copy && /^## / {
    exit
  }
  copy {
    if ($0 == "") {
      blank_lines++
      next
    }
    while (blank_lines > 0) {
      print ""
      blank_lines--
    }
    print
  }
  END {
    if (!found) {
      exit 3
    }
  }
' "${changes_file}" >"${manual_notes}" || {
  echo "release section ${base_version} was not found in ${changes_file}" >&2
  exit 2
}

previous_stable_tag=""
while IFS= read -r candidate; do
  if [[ "${candidate}" == "${tag}" || "${candidate}" == *-* ]]; then
    continue
  fi
  if [[ "${candidate}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    previous_stable_tag="${candidate}"
    break
  fi
done < <(git tag --merged "${tag}" --sort=-version:refname)

commit_range="${tag}"
if [[ -n "${previous_stable_tag}" ]]; then
  commit_range="${previous_stable_tag}..${tag}"
fi

cliff_args=(--config "${cliff_config}")
if [[ "${tag}" == *-* ]]; then
  cliff_args+=(--tag "${tag}")
fi
git cliff "${cliff_args[@]}" "${commit_range}" >"${raw_chore_notes}"
awk '
  NF {
    while (blank_lines > 0 && emitted) {
      print ""
      blank_lines--
    }
    print
    emitted = 1
    next
  }
  emitted {
    blank_lines++
  }
' "${raw_chore_notes}" >"${chore_notes}"

mkdir -p "$(dirname "${output}")"
{
  printf '## %s - %s\n' "${tag}" "${release_date}"
  cat "${manual_notes}"
  if [[ -s "${chore_notes}" ]]; then
    printf '\n'
    cat "${chore_notes}"
  fi
} >"${output}"

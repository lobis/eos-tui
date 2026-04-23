#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/publish-patch.sh [options]

Compute the next patch semver tag from the latest vMAJOR.MINOR.PATCH tag,
optionally run local checks, create an annotated tag, and push it.

Options:
  --remote <name>     Git remote to push to (default: origin)
  --no-push           Create the tag locally but do not push it
  --skip-checks       Skip local go build/go test preflight checks
  --dry-run           Print the actions without changing anything
  -h, --help          Show this help

Examples:
  scripts/publish-patch.sh
  scripts/publish-patch.sh --dry-run
  scripts/publish-patch.sh --no-push
EOF
}

REMOTE="origin"
NO_PUSH=0
SKIP_CHECKS=0
DRY_RUN=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --remote)
      [[ $# -ge 2 ]] || { echo "missing value for --remote" >&2; exit 1; }
      REMOTE="$2"
      shift 2
      ;;
    --no-push)
      NO_PUSH=1
      shift
      ;;
    --skip-checks)
      SKIP_CHECKS=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "working tree is not clean; commit or stash changes before publishing" >&2
  exit 1
fi

if [[ -n "$(git ls-files --others --exclude-standard)" ]]; then
  echo "untracked files present; clean up before publishing" >&2
  exit 1
fi

latest_tag="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-version:refname | head -n 1)"
if [[ -z "$latest_tag" ]]; then
  echo "no existing semver tags found (expected vMAJOR.MINOR.PATCH)" >&2
  exit 1
fi

if [[ ! "$latest_tag" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
  echo "latest tag $latest_tag does not match vMAJOR.MINOR.PATCH" >&2
  exit 1
fi

major="${BASH_REMATCH[1]}"
minor="${BASH_REMATCH[2]}"
patch="${BASH_REMATCH[3]}"
next_patch=$((patch + 1))
next_tag="v${major}.${minor}.${next_patch}"

if git rev-parse -q --verify "refs/tags/${next_tag}" >/dev/null; then
  echo "tag ${next_tag} already exists" >&2
  exit 1
fi

echo "latest tag: ${latest_tag}"
echo "next tag:   ${next_tag}"

if [[ "$SKIP_CHECKS" -eq 0 ]]; then
  echo "+ go build ./..."
  if [[ "$DRY_RUN" -eq 0 ]]; then
    go build ./...
  fi

  echo "+ go test ./..."
  if [[ "$DRY_RUN" -eq 0 ]]; then
    go test ./...
  fi
fi

echo "+ git tag -a ${next_tag} -m 'Release ${next_tag}'"
if [[ "$DRY_RUN" -eq 0 ]]; then
  git tag -a "${next_tag}" -m "Release ${next_tag}"
fi

if [[ "$NO_PUSH" -eq 0 ]]; then
  echo "+ git push ${REMOTE} ${next_tag}"
  if [[ "$DRY_RUN" -eq 0 ]]; then
    git push "${REMOTE}" "${next_tag}"
  fi
else
  echo "tag created locally only"
fi

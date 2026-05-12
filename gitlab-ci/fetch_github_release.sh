#!/usr/bin/env bash
set -euo pipefail

GITHUB_REPO="${GITHUB_REPO:-cern-eos/eos-tui}"
TAG="${CI_COMMIT_TAG:-${RELEASE_TAG:-}}"

if [ -z "${TAG}" ]; then
    echo "ERROR: no tag to fetch."
    echo "Set RELEASE_TAG=vX.Y.Z when triggering this pipeline manually."
    exit 1
fi

if [[ ! "${TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "ERROR: tag '${TAG}' does not look like vMAJOR.MINOR.PATCH."
    exit 1
fi

OUT_DIR="github-release"
RAW_BASE="https://raw.githubusercontent.com/${GITHUB_REPO}/${TAG}"
API_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/tags/${TAG}"

CURL_RETRY=(
    --fail --silent --show-error --location
    --connect-timeout 30
    --max-time 600
    --retry 8
    --retry-delay 5
    --retry-max-time 600
    --retry-all-errors
)

curl_github() {
    local args=("${CURL_RETRY[@]}")
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        args+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
    fi
    curl "${args[@]}" "$@"
}

run_with_retry() {
    local attempt=1
    local max=5
    local delay=15
    until "$@"; do
        if [ "${attempt}" -ge "${max}" ]; then
            echo "ERROR: command failed after ${attempt} attempts: $*"
            return 1
        fi
        echo "Attempt ${attempt} failed; sleeping ${delay}s before retry..." >&2
        sleep "${delay}"
        attempt=$((attempt + 1))
        delay=$((delay * 2))
    done
}

fetch_release_metadata() {
    curl_github \
        -H "Accept: application/vnd.github+json" \
        "${API_URL}" -o "${OUT_DIR}/release.json"
}

download_asset() {
    local url="$1"
    local name="$2"
    curl_github \
        -H "Accept: application/octet-stream" \
        "${url}" -o "${OUT_DIR}/${name}.part"
    mv "${OUT_DIR}/${name}.part" "${OUT_DIR}/${name}"
}

download_raw() {
    local path="$1"
    local name="$2"
    curl "${CURL_RETRY[@]}" \
        "${RAW_BASE}/${path}" -o "${OUT_DIR}/${name}.part"
    mv "${OUT_DIR}/${name}.part" "${OUT_DIR}/${name}"
}

rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"

echo "Fetching release metadata for ${GITHUB_REPO} ${TAG}"
run_with_retry fetch_release_metadata

ASSETS=()
while IFS= read -r asset; do
    ASSETS+=("${asset}")
done < <(jq -r '.assets[] | "\(.name)\t\(.browser_download_url)"' "${OUT_DIR}/release.json")
if [ "${#ASSETS[@]}" -eq 0 ]; then
    echo "ERROR: no release assets found for ${TAG}"
    exit 1
fi

for entry in "${ASSETS[@]}"; do
    name="${entry%%	*}"
    url="${entry#*	}"
    echo "Downloading asset: ${name}"
    run_with_retry download_asset "${url}" "${name}"
done

# LICENSE / README aren't release assets; fetch from the tagged source tree.
echo "Downloading LICENSE and README at ${TAG}"
run_with_retry download_raw "LICENSE" "LICENSE"
run_with_retry download_raw "README.md" "README.md"

if [ -f "${OUT_DIR}/SHA256SUMS.txt" ]; then
    echo "Verifying SHA256SUMS.txt"
    # Older releases may contain a checksum entry for SHA256SUMS.txt itself.
    # That entry is not stable because the file changes when the entry is added.
    (cd "${OUT_DIR}" && grep -vE '(^|[[:space:]])SHA256SUMS\.txt$' SHA256SUMS.txt | sha256sum -c -)
else
    echo "WARNING: SHA256SUMS.txt not present in release; skipping checksum verification."
fi

echo "Fetched assets:"
ls -la "${OUT_DIR}"

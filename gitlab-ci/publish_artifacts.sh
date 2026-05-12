#!/usr/bin/env bash
# Layout under ${STCI_ROOT_PATH}/${EOS_CODENAME}/:
#   el-{9,10}/{x86_64,aarch64}/  RPM repos with createrepo_c metadata
#   releases/<tag>/              flat copy of every release asset + LICENSE/README

set -euo pipefail

STCI_ROOT_PATH="${STCI_ROOT_PATH:-/eos/project/s/storage-ci/www}"
EOS_CODENAME="${EOS_CODENAME:-eos-tui}"
SRC_DIR="${SRC_DIR:-github-release}"
TAG="${CI_COMMIT_TAG:-${RELEASE_TAG:-}}"

if [ -z "${TAG}" ]; then
    echo "ERROR: no release tag set (CI_COMMIT_TAG / RELEASE_TAG)"
    exit 1
fi

if [ ! -d "${SRC_DIR}" ]; then
    echo "ERROR: ${SRC_DIR} not found; run fetch_github_release.sh first."
    exit 1
fi

ROOT="${STCI_ROOT_PATH}/${EOS_CODENAME}"
RELEASES_DIR="${ROOT}/releases/${TAG}"

echo "Publishing eos-tui ${TAG} under ${ROOT}"

shopt -s nullglob
for rpm in "${SRC_DIR}"/*.rpm; do
    name="$(basename "${rpm}")"
    case "${name}" in
        *.el9.x86_64.rpm)   dest="${ROOT}/el-9/x86_64" ;;
        *.el9.aarch64.rpm)  dest="${ROOT}/el-9/aarch64" ;;
        *.el10.x86_64.rpm)  dest="${ROOT}/el-10/x86_64" ;;
        *.el10.aarch64.rpm) dest="${ROOT}/el-10/aarch64" ;;
        *)
            echo "WARNING: unrecognized RPM '${name}', skipping"
            continue
            ;;
    esac
    mkdir -p "${dest}"
    cp -f "${rpm}" "${dest}/"
    echo "  -> ${dest}/${name}"
done
shopt -u nullglob

for dist in el-9 el-10; do
    for arch in x86_64 aarch64; do
        repo="${ROOT}/${dist}/${arch}"
        if [ -d "${repo}" ]; then
            echo "createrepo_c ${repo}"
            createrepo_c -q --update "${repo}" || createrepo_c -q "${repo}"
        fi
    done
done

mkdir -p "${RELEASES_DIR}"
for f in "${SRC_DIR}"/*; do
    name="$(basename "${f}")"
    [ "${name}" = "release.json" ] && continue
    cp -f "${f}" "${RELEASES_DIR}/${name}"
done

echo "Published assets in ${RELEASES_DIR}:"
ls -la "${RELEASES_DIR}"

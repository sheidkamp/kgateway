#!/usr/bin/env bash
# shellcheck disable=SC1000-SC9999

# Verifies the copy-by-digest primitive used by the release workflow's publish job:
# copying a staged multi-arch image to its published tag must preserve the exact bytes.
#
# The release pipeline stages images under a non-semver tag, runs the gate, then promotes
# the validated image to its semver tag with `docker buildx imagetools create`, asserting
# the published digest equals the staged digest. This script exercises that same copy and
# assertion against a throwaway local registry:2, so the load-bearing claim can be checked
# in seconds without GitHub or ghcr.
#
# Requirements: docker with buildx. Spins up (and tears down) a local registry:2 container.

set -o errexit
set -o pipefail
set -o nounset

REGISTRY_PORT="${REGISTRY_PORT:-5000}"
REGISTRY_HOST="localhost:${REGISTRY_PORT}"
REGISTRY_NAME="kgw-promo-verify-reg"
IMAGE="${REGISTRY_HOST}/kgateway"
STAGE_TAG="stage-test"
PUBLISH_TAG="v0.0.0-test"

cleanup() {
    docker rm -f "${REGISTRY_NAME}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# The digest of a manifest is the sha256 of its raw bytes; --raw returns exactly those.
# This mirrors the digest() helper in the release workflow's publish job.
digest() {
    docker buildx imagetools inspect "$1" --raw | sha256sum | awk '{print "sha256:"$1}'
}

echo "==> Starting local registry:2 on ${REGISTRY_HOST}"
cleanup
docker run -d -p "${REGISTRY_PORT}:5000" --name "${REGISTRY_NAME}" registry:2 >/dev/null
# Give the registry a moment to accept connections.
until docker buildx imagetools inspect "${REGISTRY_HOST}/nothing:nothing" >/dev/null 2>&1 \
    || curl -sf "http://${REGISTRY_HOST}/v2/" >/dev/null 2>&1; do
    sleep 1
done

echo "==> Staging a multi-arch manifest at ${IMAGE}:${STAGE_TAG}"
# A minimal multi-arch image is enough; only the manifest bytes matter for this check.
docker buildx build --platform linux/amd64,linux/arm64 \
    -t "${IMAGE}:${STAGE_TAG}" --push - <<'DOCKERFILE'
FROM alpine:3.17.6
DOCKERFILE

echo "==> Promoting staged image to ${IMAGE}:${PUBLISH_TAG} (copy by digest)"
src="${IMAGE}:${STAGE_TAG}"
dst="${IMAGE}:${PUBLISH_TAG}"
staged="$(digest "$src")"
docker buildx imagetools create -t "$dst" "${src}@${staged}"
published="$(digest "$dst")"

echo "    staged:    ${staged}"
echo "    published: ${published}"
if [[ "$published" != "$staged" ]]; then
    echo "FAIL: promotion did not preserve the digest (${published} != ${staged})" >&2
    echo "      buildx re-serialized the manifest index; the publish job would fail closed here." >&2
    exit 1
fi

echo "PASS: copy-by-digest preserved the manifest (${published})"

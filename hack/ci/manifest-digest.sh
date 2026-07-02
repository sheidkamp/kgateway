#!/usr/bin/env bash
# shellcheck disable=SC1000-SC9999

# Prints the sha256 digest of an image manifest's raw bytes for the given ref.
#
# The release workflow uses this in two places that MUST agree: the stage job captures
# each staged image's digest, and the publish job recomputes it to detect drift before
# promoting and to confirm the promoted tag preserved the exact bytes. Keeping the
# computation in one script prevents the two callers from diverging.
set -o errexit
set -o pipefail
set -o nounset

if [[ $# -ne 1 ]]; then
    echo "usage: $(basename "$0") <image-ref>" >&2
    exit 2
fi

# The digest of a manifest is the sha256 of its raw bytes; --raw returns exactly those.
docker buildx imagetools inspect "$1" --raw | sha256sum | awk '{print "sha256:"$1}'

#!/usr/bin/env bash
# Quick local/CI image build helper. Use --save for low-resource servers.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

IMAGE_TAG="${IMAGE_TAG:-sub2api:latest}"
DOCKERFILE="${DOCKERFILE:-${REPO_ROOT}/Dockerfile}"
SAVE_ARCHIVE=""

usage() {
    cat <<'EOF'
Usage:
  bash deploy/build_image.sh [--tag IMAGE] [--dockerfile FILE] [--save ARCHIVE.tar.gz]

Examples:
  IMAGE_TAG=sub2api-hot:0.1.129 bash deploy/build_image.sh --save /tmp/sub2api-hot.tar.gz
  bash deploy/build_image.sh --tag sub2api-hot:local --dockerfile Dockerfile.prebuilt
EOF
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --tag)
            [ "$#" -ge 2 ] || { echo "--tag requires an image tag" >&2; exit 1; }
            IMAGE_TAG="$2"
            shift 2
            ;;
        --dockerfile)
            [ "$#" -ge 2 ] || { echo "--dockerfile requires a path" >&2; exit 1; }
            DOCKERFILE="$2"
            shift 2
            ;;
        --save)
            [ "$#" -ge 2 ] || { echo "--save requires an archive path" >&2; exit 1; }
            SAVE_ARCHIVE="$2"
            shift 2
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

docker build -t "$IMAGE_TAG" \
    --build-arg GOPROXY=https://goproxy.cn,direct \
    --build-arg GOSUMDB=sum.golang.google.cn \
    -f "$DOCKERFILE" \
    "$REPO_ROOT"

if [ -n "$SAVE_ARCHIVE" ]; then
    case "$SAVE_ARCHIVE" in
        *.tar.gz|*.tgz) ;;
        *) SAVE_ARCHIVE="${SAVE_ARCHIVE}.tar.gz" ;;
    esac

    tmp_tar="$(mktemp "${TMPDIR:-/tmp}/sub2api-image.XXXXXX.tar")"
    rm -f "$tmp_tar" "$SAVE_ARCHIVE"
    docker save -o "$tmp_tar" "$IMAGE_TAG"
    gzip -c "$tmp_tar" > "$SAVE_ARCHIVE"
    rm -f "$tmp_tar"
    printf 'Saved %s\n' "$SAVE_ARCHIVE"
fi

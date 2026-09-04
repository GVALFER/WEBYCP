#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)
ALLOW_DIRTY=false

log() {
    printf '%s\n' "WEBYCP release: $*"
}

fail() {
    printf '%s\n' "WEBYCP release: $*" >&2
    exit 1
}

usage() {
    printf '%s\n' "Usage: $0 VERSION [--allow-dirty]"
}

[ "$#" -ge 1 ] || {
    usage >&2
    exit 1
}

VERSION=$1
shift

while [ "$#" -gt 0 ]; do
    case "$1" in
        --allow-dirty)
            ALLOW_DIRTY=true
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            usage >&2
            fail "unknown option: $1"
            ;;
    esac
    shift
done

printf '%s\n' "$VERSION" |
    grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$' ||
    fail "version must use SemVer without a leading v, for example 1.0.0"

for command in docker go node npm gzip; do
    command -v "$command" >/dev/null 2>&1 || fail "required command is missing: $command"
done

NODE_MAJOR=$(node -p 'Number(process.versions.node.split(".")[0])')
[ "$NODE_MAJOR" -ge 24 ] || fail "Node.js 24 or newer is required"

COMMIT=${WEBYCP_RELEASE_COMMIT:-unknown}
SOURCE_EPOCH=${SOURCE_DATE_EPOCH:-}

if command -v git >/dev/null 2>&1 &&
    git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    COMMIT=${WEBYCP_RELEASE_COMMIT:-$(git -C "$ROOT_DIR" rev-parse --short=12 HEAD)}
    SOURCE_EPOCH=${SOURCE_DATE_EPOCH:-$(git -C "$ROOT_DIR" log -1 --format=%ct)}

    if ! git -C "$ROOT_DIR" diff --quiet --ignore-submodules -- ||
        ! git -C "$ROOT_DIR" diff --cached --quiet --ignore-submodules -- ||
        [ -n "$(git -C "$ROOT_DIR" ls-files --others --exclude-standard)" ]; then
        [ "$ALLOW_DIRTY" = true ] ||
            fail "working tree is not clean; commit the release or pass --allow-dirty for local testing"
        COMMIT="${COMMIT}-dirty"
    fi
fi

[ -n "$SOURCE_EPOCH" ] ||
    fail "SOURCE_DATE_EPOCH is required when building outside a Git worktree"

case "$COMMIT" in
    *[!0-9A-Za-z._-]*) fail "release commit contains unsupported characters" ;;
esac
case "$SOURCE_EPOCH" in
    ''|*[!0-9]*) fail "SOURCE_DATE_EPOCH must be a non-negative integer" ;;
esac

OUTPUT_DIR=${WEBYCP_RELEASE_DIR:-$ROOT_DIR/dist}
case "$OUTPUT_DIR" in
    /*) ;;
    *) OUTPUT_DIR="$ROOT_DIR/$OUTPUT_DIR" ;;
esac
RELEASE_NAME="webycp-$VERSION-linux-amd64"
ARCHIVE_NAME="$RELEASE_NAME.tar.gz"
CHECKSUM_NAME="$ARCHIVE_NAME.sha256"
STAGE_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/webycp-release.XXXXXX")
RELEASE_DIR="$STAGE_ROOT/$RELEASE_NAME"
ARCHIVE_TMP="$OUTPUT_DIR/.$ARCHIVE_NAME.tmp.$$"
CHECKSUM_TMP="$OUTPUT_DIR/.$CHECKSUM_NAME.tmp.$$"

cleanup() {
    rm -rf -- "$STAGE_ROOT"
    rm -f -- "$ARCHIVE_TMP" "$CHECKSUM_TMP"
}

trap cleanup EXIT
trap 'exit 1' HUP INT TERM

export COPYFILE_DISABLE=1

cd "$ROOT_DIR"

log "Installing locked frontend dependencies"
npm --prefix "$ROOT_DIR/web" ci --prefer-offline --no-audit --no-fund

log "Running repository checks"
make -C "$ROOT_DIR" check
make -C "$ROOT_DIR" security

mkdir -p "$RELEASE_DIR/bin" "$RELEASE_DIR/docs"

LDFLAGS="-s -w -X github.com/GVALFER/WEBYCP/internal/buildinfo.Version=$VERSION -X github.com/GVALFER/WEBYCP/internal/buildinfo.Commit=$COMMIT"

log "Building static Linux amd64 binaries"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -buildvcs=false \
    -trimpath \
    -ldflags "$LDFLAGS" \
    -o "$RELEASE_DIR/bin/webycp-server" \
    ./cmd/webycp-server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -buildvcs=false \
    -trimpath \
    -ldflags "$LDFLAGS" \
    -o "$RELEASE_DIR/bin/webycp-agent" \
    ./cmd/webycp-agent

log "Building the Linux amd64 Next.js standalone runtime"
docker buildx build \
    --platform linux/amd64 \
    --output "type=local,dest=$RELEASE_DIR" \
    --file "$ROOT_DIR/web/Dockerfile.release" \
    "$ROOT_DIR/web"
check_binary() {
    binary=$1
    magic=$(od -An -j0 -N4 -t x1 "$binary" | tr -d ' \n')
    machine=$(od -An -j18 -N2 -t x1 "$binary" | tr -d ' \n')
    [ "$magic" = 7f454c46 ] && [ "$machine" = 3e00 ] ||
        fail "not a Linux amd64 binary: $binary"
}
check_binary "$RELEASE_DIR/runtime/node"
mkdir -p \
    "$RELEASE_DIR/packaging/nginx" \
    "$RELEASE_DIR/packaging/systemd" \
    "$RELEASE_DIR/packaging/ubuntu"
cp "$ROOT_DIR/packaging/README.md" "$RELEASE_DIR/packaging/"
cp \
    "$ROOT_DIR/packaging/nginx/panel-bootstrap.conf" \
    "$ROOT_DIR/packaging/nginx/webycp.conf" \
    "$RELEASE_DIR/packaging/nginx/"
cp \
    "$ROOT_DIR/packaging/systemd/webycp-agent.service" \
    "$ROOT_DIR/packaging/systemd/webycp-server.service" \
    "$ROOT_DIR/packaging/systemd/webycp-web.service" \
    "$RELEASE_DIR/packaging/systemd/"
cp \
    "$ROOT_DIR/packaging/ubuntu/agent.env.example" \
    "$ROOT_DIR/packaging/ubuntu/install.sh" \
    "$ROOT_DIR/packaging/ubuntu/server.env.example" \
    "$ROOT_DIR/packaging/ubuntu/upgrade.sh" \
    "$ROOT_DIR/packaging/ubuntu/web.env.example" \
    "$RELEASE_DIR/packaging/ubuntu/"
cp "$ROOT_DIR/docs/operations.md" "$RELEASE_DIR/docs/"
cp \
    "$ROOT_DIR/.gitleaks.toml" \
    "$ROOT_DIR/LICENSE" \
    "$ROOT_DIR/NOTICE" \
    "$ROOT_DIR/README.md" \
    "$RELEASE_DIR/"
printf '%s\n' "$VERSION" >"$RELEASE_DIR/VERSION"

find "$RELEASE_DIR" -type d -exec chmod 0755 {} +
find "$RELEASE_DIR" -type f -exec chmod 0644 {} +
chmod 0755 \
    "$RELEASE_DIR/bin/webycp-agent" \
    "$RELEASE_DIR/bin/webycp-server" \
    "$RELEASE_DIR/runtime/node" \
    "$RELEASE_DIR/packaging/ubuntu/install.sh" \
    "$RELEASE_DIR/packaging/ubuntu/upgrade.sh"

mkdir -p "$OUTPUT_DIR"

GNU_TAR=
for candidate in gtar tar; do
    if command -v "$candidate" >/dev/null 2>&1 &&
        "$candidate" --version 2>/dev/null | grep -q 'GNU tar'; then
        GNU_TAR=$candidate
        break
    fi
done

log "Creating deterministic archive"
if [ -n "$GNU_TAR" ]; then
    "$GNU_TAR" \
        --sort=name \
        --mtime="@$SOURCE_EPOCH" \
        --owner=0 \
        --group=0 \
        --numeric-owner \
        --format=gnu \
        -C "$STAGE_ROOT" \
        -cf - "$RELEASE_NAME" |
        gzip -n >"$ARCHIVE_TMP"
elif command -v docker >/dev/null 2>&1; then
    docker run --rm \
        --user "$(id -u):$(id -g)" \
        -e RELEASE_NAME="$RELEASE_NAME" \
        -e SOURCE_EPOCH="$SOURCE_EPOCH" \
        -v "$STAGE_ROOT:/stage:ro" \
        -v "$OUTPUT_DIR:/output" \
        ubuntu@sha256:33ceb71981b602c1a7443a53469e4dba065f7503eab3078a2d7a57a2ab987517 \
        sh -eu -c 'tar --sort=name --mtime="@$SOURCE_EPOCH" --owner=0 --group=0 --numeric-owner --format=gnu -C /stage -cf - "$RELEASE_NAME" | gzip -n >"/output/.$RELEASE_NAME.tar.gz.tmp"'
    mv -f -- "$OUTPUT_DIR/.$ARCHIVE_NAME.tmp" "$ARCHIVE_TMP"
else
    fail "GNU tar or Docker is required to create a deterministic archive"
fi

mv -f -- "$ARCHIVE_TMP" "$OUTPUT_DIR/$ARCHIVE_NAME"

"$ROOT_DIR/scripts/security.sh" \
    --artifact "$OUTPUT_DIR/$ARCHIVE_NAME"

if command -v sha256sum >/dev/null 2>&1; then
    CHECKSUM=$(sha256sum "$OUTPUT_DIR/$ARCHIVE_NAME" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    CHECKSUM=$(shasum -a 256 "$OUTPUT_DIR/$ARCHIVE_NAME" | awk '{print $1}')
else
    fail "sha256sum or shasum is required"
fi

printf '%s  %s\n' "$CHECKSUM" "$ARCHIVE_NAME" >"$CHECKSUM_TMP"
mv -f -- "$CHECKSUM_TMP" "$OUTPUT_DIR/$CHECKSUM_NAME"

log "Created $OUTPUT_DIR/$ARCHIVE_NAME"
log "SHA-256: $CHECKSUM"

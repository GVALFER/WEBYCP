#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)
GITLEAKS_VERSION=v8.30.1
GOVULNCHECK_VERSION=v1.7.0
OSV_IMAGE=ghcr.io/google/osv-scanner@sha256:5116601dedc01c1c580eb92371883ec052fc4c13c3fbc109d621a63ac416d475
MODE=all
ARTIFACT=

log() {
    printf '%s\n' "WEBYCP security: $*"
}

fail() {
    printf '%s\n' "WEBYCP security: $*" >&2
    exit 1
}

usage() {
    printf '%s\n' "Usage: $0 [--artifact FILE|--test-secret-scan]"
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --artifact)
            [ "$#" -ge 2 ] || fail "--artifact requires a file"
            MODE=artifact
            ARTIFACT=$2
            shift 2
            ;;
        --test-secret-scan)
            MODE=test-secret
            shift
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
done

run_gitleaks() {
    go run "github.com/zricethezav/gitleaks/v8@$GITLEAKS_VERSION" \
        --config "$ROOT_DIR/.gitleaks.toml" "$@"
}

scan_secrets() {
    log "Scanning Git history for secrets"
    run_gitleaks git \
        --redact \
        --no-banner \
        --no-color \
        --log-level=warn \
        --timeout=120 \
        "$ROOT_DIR"

    log "Scanning the working tree for secrets"
    run_gitleaks dir \
        --redact \
        --no-banner \
        --no-color \
        --log-level=warn \
        --timeout=120 \
        "$ROOT_DIR"
}

scan_artifact() {
    [ -f "$ARTIFACT" ] && [ ! -L "$ARTIFACT" ] ||
        fail "artifact must be a regular file: $ARTIFACT"
    log "Scanning release artifact for secrets"
    run_gitleaks dir \
        --redact \
        --no-banner \
        --no-color \
        --log-level=warn \
        --max-archive-depth=2 \
        --timeout=120 \
        "$ARTIFACT"
}

audit_frontend() {
    report=$(mktemp "${TMPDIR:-/tmp}/webycp-npm-audit.XXXXXX")
    attempt=1
    while [ "$attempt" -le 2 ]; do
        if npm --prefix "$ROOT_DIR/web" audit \
            --omit=dev \
            --audit-level=high \
            --fetch-timeout=30000 \
            --fetch-retries=1 \
            --fetch-retry-mintimeout=1000 \
            --fetch-retry-maxtimeout=5000 >"$report" 2>&1; then
            cat "$report"
            rm -f -- "$report"
            return
        fi

        if ! grep -Eiq \
            'audit endpoint returned an error|network timeout|ENOAUDIT|EAI_AGAIN|ECONNRESET|ECONNREFUSED|ENETUNREACH|ETIMEDOUT|socket hang up|429 Too Many Requests|5[0-9][0-9] (Internal Server Error|Service Unavailable|Gateway)' \
            "$report"; then
            cat "$report" >&2
            rm -f -- "$report"
            return 1
        fi

        [ "$attempt" -lt 2 ] || break
        log "npm audit network failure; retrying ($attempt/2)"
        attempt=$((attempt + 1))
        sleep 2
    done

    cat "$report" >&2
    rm -f -- "$report"
    command -v docker >/dev/null 2>&1 ||
        fail "npm audit is unavailable and Docker is required for the OSV fallback"
    docker info >/dev/null 2>&1 ||
        fail "npm audit is unavailable and Docker is not running for the OSV fallback"

    log "npm audit is unavailable; checking the lockfile with OSV Scanner"
    docker run --rm \
        -v "$ROOT_DIR:/src:ro" \
        "$OSV_IMAGE" \
        scan source -L /src/web/package-lock.json --verbosity=error
}

test_secret_scan() {
    fixture=$(mktemp -d "${TMPDIR:-/tmp}/webycp-secret-test.XXXXXX")
    trap 'rm -rf -- "$fixture"' EXIT HUP INT TERM
    prefix=ghp_
    suffix=A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8
    printf 'GITHUB_TOKEN=%s%s\n' "$prefix" "$suffix" >"$fixture/leaked.env"

    if run_gitleaks dir \
        --redact \
        --no-banner \
        --no-color \
        --log-level=error \
        "$fixture" >/dev/null 2>&1; then
        fail "secret scanner accepted the leak fixture"
    fi

    rm -rf -- "$fixture"
    trap - EXIT HUP INT TERM
    log "Secret scanner rejection test passed"
}

scan_all() {
    log "Checking Go call paths for known vulnerabilities"
    go run "golang.org/x/vuln/cmd/govulncheck@$GOVULNCHECK_VERSION" \
        ./cmd/... ./internal/...

    log "Checking production frontend dependencies"
    audit_frontend

    scan_secrets
    test_secret_scan
}

cd "$ROOT_DIR"
case "$MODE" in
    all) scan_all ;;
    artifact) scan_artifact ;;
    test-secret) test_secret_scan ;;
esac

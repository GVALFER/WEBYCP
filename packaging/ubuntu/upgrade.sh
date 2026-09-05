#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
SOURCE_DIR=$(CDPATH='' cd -- "$SCRIPT_DIR/../.." && pwd)
BACKUP_ROOT=/var/lib/webycp/upgrades
MODE=upgrade
CHECK_ONLY=false
RECOVERY_DIR=
BACKUP_DIR=
WEB_STAGE=
WEB_OLD=
STOPPED=false
CHANGED=false
DONE=false

log() {
    printf '%s\n' "WEBYCP upgrade: $*"
}

fail() {
    printf '%s\n' "WEBYCP upgrade: $*" >&2
    exit 1
}

# shellcheck source=packaging/ubuntu/powerdns.sh
. "$SCRIPT_DIR/powerdns.sh"

usage() {
    printf '%s\n' "Usage: $0 [--source DIR] [--check]"
    printf '%s\n' "       $0 --recover BACKUP_DIR"
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --source)
            [ "$#" -ge 2 ] || fail "--source requires a directory"
            [ -d "$2" ] || fail "source directory does not exist: $2"
            SOURCE_DIR=$(CDPATH='' cd -- "$2" && pwd)
            shift 2
            ;;
        --check)
            CHECK_ONLY=true
            shift
            ;;
        --recover)
            [ "$#" -ge 2 ] || fail "--recover requires a backup directory"
            MODE=recover
            RECOVERY_DIR=$2
            shift 2
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

[ "$MODE" != recover ] || [ "$CHECK_ONLY" = false ] ||
    fail "--check and --recover cannot be used together"

require_root() {
    [ "$(id -u)" -eq 0 ] || fail "the upgrade must run as root"
}

check_host() {
    [ -r /etc/os-release ] || fail "cannot identify the operating system"
    # shellcheck disable=SC1091
    . /etc/os-release
    if [ "${ID:-}" != ubuntu ] || [ "${VERSION_ID:-}" != 24.04 ]; then
        fail "Ubuntu 24.04 is required"
    fi
    [ "$(dpkg --print-architecture)" = amd64 ] || fail "amd64 is required"
    [ -d /run/systemd/system ] || fail "systemd is not running"
}

check_file() {
    path=$1
    label=$2
    [ -f "$path" ] && [ ! -L "$path" ] ||
        fail "$label must be a regular file: $path"
}

check_dir() {
    path=$1
    label=$2
    [ -d "$path" ] && [ ! -L "$path" ] ||
        fail "$label must be a directory: $path"
}

check_binary() {
    binary=$1
    magic=$(od -An -j0 -N4 -t x1 "$binary" | tr -d ' \n')
    machine=$(od -An -j18 -N2 -t x1 "$binary" | tr -d ' \n')
    if [ "$magic" != 7f454c46 ] || [ "$machine" != 3e00 ]; then
        fail "not a Linux amd64 binary: $binary"
    fi
}

read_version() {
    version_file=$1
    check_file "$version_file" "version marker"
    version=$(sed -n '1p' "$version_file")
    [ "$(wc -l <"$version_file" | tr -d ' ')" -eq 1 ] ||
        fail "version marker must contain exactly one line: $version_file"
    printf '%s\n' "$version" |
        grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$' ||
        fail "invalid release version: $version"
    printf '%s\n' "$version"
}

check_release() {
    for path in \
        bin/webycp-agent \
        bin/webycp-server \
        runtime/node \
        web/server.js \
        packaging/systemd/webycp-agent.service \
        packaging/systemd/webycp-server.service \
        packaging/systemd/webycp-web.service \
        packaging/ubuntu/powerdns.sh \
        packaging/ubuntu/web.env.example \
        VERSION
    do
        check_file "$SOURCE_DIR/$path" "release file"
    done
    check_dir "$SOURCE_DIR/web" "release web build"
    check_dir "$SOURCE_DIR/web/.next/static" "release Next.js static build"
    if find "$SOURCE_DIR/web" -type l -print -quit | grep -q .; then
        fail "release web build must not contain symlinks"
    fi
    check_binary "$SOURCE_DIR/bin/webycp-agent"
    check_binary "$SOURCE_DIR/bin/webycp-server"
    check_binary "$SOURCE_DIR/runtime/node"

    RELEASE_VERSION=$(read_version "$SOURCE_DIR/VERSION")
    for binary in webycp-agent webycp-server; do
        binary_version=$("$SOURCE_DIR/bin/$binary" --version | awk '{print $2}')
        [ "$binary_version" = "$RELEASE_VERSION" ] ||
            fail "$binary version does not match VERSION: $binary"
    done
}

check_install() {
    check_file /usr/lib/webycp/webycp-agent "installed Agent"
    check_file /usr/lib/webycp/webycp-server "installed Server"
    check_file /usr/lib/systemd/system/webycp-agent.service "installed Agent unit"
    check_file /usr/lib/systemd/system/webycp-server.service "installed Server unit"
    check_file /etc/webycp/server.env "Server environment"
    check_dir /usr/share/webycp/web "installed web build"
    check_dir /var/lib/webycp/server "Server state"
    check_file /etc/nginx/webycp/sites-available/panel.conf "panel Nginx configuration"
}

wait_for_socket() {
    attempts=0
    while [ "$attempts" -lt 20 ]; do
        if systemctl is-active --quiet webycp-agent &&
            [ -S /run/webycp/agent.sock ]; then
            return
        fi
        attempts=$((attempts + 1))
        sleep 1
    done
    return 1
}

wait_for_server() {
    attempts=0
    while [ "$attempts" -lt 20 ]; do
        if systemctl is-active --quiet webycp-server &&
            curl --fail --silent --show-error \
                http://127.0.0.1:8080/api/v1/health >/dev/null 2>&1; then
            return
        fi
        attempts=$((attempts + 1))
        sleep 1
    done
    return 1
}

wait_for_web() {
    attempts=0
    while [ "$attempts" -lt 20 ]; do
        if systemctl is-active --quiet webycp-web &&
            curl --fail --silent --show-error \
                http://127.0.0.1:3000/login >/dev/null 2>&1; then
            return
        fi
        attempts=$((attempts + 1))
        sleep 1
    done
    return 1
}

check_live_install() {
    systemctl is-active --quiet webycp-agent || fail "Agent is not active"
    systemctl is-active --quiet webycp-server || fail "Server is not active"
    wait_for_socket || fail "Agent socket is not ready"
    wait_for_server || fail "Server health endpoint is not ready"
    if [ -f /usr/lib/systemd/system/webycp-web.service ]; then
        wait_for_web || fail "Web frontend is not ready"
    fi
    if [ -f "$POWERDNS_CONFIG" ]; then
        webycp_wait_powerdns || fail "PowerDNS is not ready"
    fi
    /usr/sbin/nginx -t || fail "Nginx configuration is invalid"
}

check_backup() {
    resolved=$(readlink -f -- "$RECOVERY_DIR") ||
        fail "backup directory does not exist: $RECOVERY_DIR"
    case "$resolved" in
        "$BACKUP_ROOT"/*) ;;
        *) fail "backup must be inside $BACKUP_ROOT" ;;
    esac
    RECOVERY_DIR=$resolved
    check_dir "$RECOVERY_DIR" "backup"
    check_file "$RECOVERY_DIR/bin/webycp-agent" "backup Agent"
    check_file "$RECOVERY_DIR/bin/webycp-server" "backup Server"
    check_file "$RECOVERY_DIR/systemd/webycp-agent.service" "backup Agent unit"
    check_file "$RECOVERY_DIR/systemd/webycp-server.service" "backup Server unit"
    check_file "$RECOVERY_DIR/VERSION" "backup version"
    check_dir "$RECOVERY_DIR/web" "backup web build"
    if [ ! -f "$RECOVERY_DIR/web/server.js" ] &&
        [ ! -f "$RECOVERY_DIR/web/index.html" ]; then
        fail "backup web build has no Next.js server or legacy index"
    fi
    check_dir "$RECOVERY_DIR/server" "backup Server state"
    check_dir "$RECOVERY_DIR/powerdns" "backup PowerDNS state"
    if [ -f "$RECOVERY_DIR/powerdns/PRESENT" ]; then
        check_file "$RECOVERY_DIR/powerdns/webycp.conf" "backup PowerDNS configuration"
        check_file "$RECOVERY_DIR/powerdns/powerdns.key" "backup PowerDNS key"
        check_file "$RECOVERY_DIR/powerdns/webycp.sqlite3" "backup PowerDNS database"
    elif [ ! -f "$RECOVERY_DIR/powerdns/ABSENT" ]; then
        fail "backup PowerDNS state marker is missing"
    fi
    RECOVERY_VERSION=$(read_version "$RECOVERY_DIR/VERSION")
}

install_atomic() {
    source=$1
    target=$2
    mode=$3
    owner=$4
    group=$5
    temporary=$(mktemp "${target}.new.XXXXXX")
    if ! install -o "$owner" -g "$group" -m "$mode" "$source" "$temporary"; then
        rm -f -- "$temporary"
        return 1
    fi
    mv -f -- "$temporary" "$target"
}

prepare_web() {
    source=$1
    WEB_STAGE=$(mktemp -d /usr/share/webycp/.web-upgrade.XXXXXX) || return 1
    cp -a "$source/." "$WEB_STAGE/" || return 1
    chown -R root:root "$WEB_STAGE" || return 1
    find "$WEB_STAGE" -type d -exec chmod 0755 {} + || return 1
    find "$WEB_STAGE" -type f -exec chmod 0644 {} + || return 1
    if [ ! -f "$WEB_STAGE/server.js" ] &&
        [ ! -f "$WEB_STAGE/index.html" ]; then
        return 1
    fi
}

current_version() {
    /usr/lib/webycp/webycp-server --version | awk '{print $2}'
}

stop_services() {
    STOPPED=true
    if [ -f /usr/lib/systemd/system/webycp-web.service ]; then
        systemctl stop webycp-web
    fi
    systemctl stop webycp-server
    systemctl stop webycp-agent
    if [ -f "$POWERDNS_CONFIG" ]; then
        systemctl stop pdns
    fi
}

start_services() {
    expected=$1
    systemctl daemon-reload || return 1
    if [ -f "$POWERDNS_CONFIG" ]; then
        webycp_start_powerdns || return 1
    fi
    systemctl start webycp-agent || return 1
    wait_for_socket || return 1
    systemctl start webycp-server || return 1
    wait_for_server || return 1
    if [ -f /usr/lib/systemd/system/webycp-web.service ]; then
        systemctl enable webycp-web >/dev/null || return 1
        systemctl start webycp-web || return 1
        wait_for_web || return 1
    fi
    /usr/sbin/nginx -t || return 1
    systemctl reload nginx || return 1
    actual=$(current_version) || return 1
    [ "$actual" = "$expected" ]
}

create_backup() {
    label=$1
    version=$(current_version)
    install -d -o root -g root -m 0700 "$BACKUP_ROOT"
    BACKUP_DIR=$(mktemp -d "$BACKUP_ROOT/$label-$(date -u +%Y%m%dT%H%M%SZ).XXXXXX")
    chmod 0700 "$BACKUP_DIR"
    install -d -o root -g root -m 0700 \
        "$BACKUP_DIR/bin" "$BACKUP_DIR/config" "$BACKUP_DIR/nginx" \
        "$BACKUP_DIR/powerdns" "$BACKUP_DIR/systemd"
    cp -a /usr/lib/webycp/webycp-agent "$BACKUP_DIR/bin/"
    cp -a /usr/lib/webycp/webycp-server "$BACKUP_DIR/bin/"
    cp -a /usr/lib/systemd/system/webycp-agent.service "$BACKUP_DIR/systemd/"
    cp -a /usr/lib/systemd/system/webycp-server.service "$BACKUP_DIR/systemd/"
    if [ -f /usr/lib/webycp/node ]; then
        cp -a /usr/lib/webycp/node "$BACKUP_DIR/bin/"
    fi
    if [ -f /usr/lib/systemd/system/webycp-web.service ]; then
        cp -a /usr/lib/systemd/system/webycp-web.service "$BACKUP_DIR/systemd/"
    fi
    if [ -f /etc/webycp/web.env ]; then
        cp -a /etc/webycp/web.env "$BACKUP_DIR/config/"
    fi
    cp -a /etc/nginx/webycp/sites-available/panel.conf "$BACKUP_DIR/nginx/"
    cp -a /usr/share/webycp/web "$BACKUP_DIR/web"
    cp -a /var/lib/webycp/server "$BACKUP_DIR/server"
    if [ -f "$POWERDNS_CONFIG" ]; then
        cp -a "$POWERDNS_CONFIG" "$BACKUP_DIR/powerdns/webycp.conf"
        cp -a "$POWERDNS_KEY" "$BACKUP_DIR/powerdns/powerdns.key"
        cp -a "$POWERDNS_DATABASE" "$BACKUP_DIR/powerdns/webycp.sqlite3"
        : >"$BACKUP_DIR/powerdns/PRESENT"
    else
        : >"$BACKUP_DIR/powerdns/ABSENT"
    fi
    printf '%s\n' "$version" >"$BACKUP_DIR/VERSION"
    chmod 0600 "$BACKUP_DIR/VERSION"
}

replace_web() {
    WEB_OLD=$(mktemp -d /usr/share/webycp/.web-previous.XXXXXX)
    rmdir "$WEB_OLD"
    mv /usr/share/webycp/web "$WEB_OLD"
    mv "$WEB_STAGE" /usr/share/webycp/web
    WEB_STAGE=
}

install_files() {
    bin_dir=$1
    unit_dir=$2
    version_file=$3
    node=$4
    web_unit=$5
    CHANGED=true
    install_atomic "$bin_dir/webycp-agent" /usr/lib/webycp/webycp-agent 0755 root root
    install_atomic "$bin_dir/webycp-server" /usr/lib/webycp/webycp-server 0755 root root
    install_atomic "$node" /usr/lib/webycp/node 0755 root root
    install_atomic \
        "$unit_dir/webycp-agent.service" \
        /usr/lib/systemd/system/webycp-agent.service 0644 root root
    install_atomic \
        "$unit_dir/webycp-server.service" \
        /usr/lib/systemd/system/webycp-server.service 0644 root root
    install_atomic \
        "$web_unit" \
        /usr/lib/systemd/system/webycp-web.service 0644 root root
    install_atomic "$version_file" /usr/lib/webycp/VERSION 0644 root root
    replace_web
}

ensure_web_config() {
    if [ -L /etc/webycp/web.env ]; then
        return 1
    fi
    if [ -e /etc/webycp/web.env ]; then
        [ -f /etc/webycp/web.env ] || return 1
        chown root:webycp /etc/webycp/web.env || return 1
        chmod 0640 /etc/webycp/web.env || return 1
        return
    fi
    install -o root -g webycp -m 0640 \
        "$SOURCE_DIR/packaging/ubuntu/web.env.example" /etc/webycp/web.env
}

switch_panel_port() {
    port=$1
    panel=/etc/nginx/webycp/sites-available/panel.conf
    check_file "$panel" "panel Nginx configuration"
    [ "$(head -n 1 "$panel")" = "# Managed by WEBYCP." ] || return 1
    stage=$(mktemp /etc/nginx/webycp/sites-available/.panel.XXXXXX) || return 1
    sed -E \
        "s#proxy_pass http://127\\.0\\.0\\.1:(3000|8080);#proxy_pass http://127.0.0.1:$port;#g" \
        "$panel" >"$stage" || {
        rm -f -- "$stage"
        return 1
    }
    if ! grep -q "proxy_pass http://127.0.0.1:$port;" "$stage"; then
        rm -f -- "$stage"
        return 1
    fi
    chown root:root "$stage" || return 1
    chmod 0644 "$stage" || return 1
    mv -f -- "$stage" "$panel"
    /usr/sbin/nginx -t
}

run_migrations() {
    runuser -u webycp -- /bin/sh -c \
        'set -a; . /etc/webycp/server.env; exec /usr/lib/webycp/webycp-server migrate'
}

restore_state() {
    snapshot=$1
    state_stage=$(mktemp -d /var/lib/webycp/.server-restore.XXXXXX) || return 1
    cp -a "$snapshot/server/." "$state_stage/" || return 1
    chown --reference="$snapshot/server" "$state_stage" || return 1
    chmod --reference="$snapshot/server" "$state_stage" || return 1
    failed=$(mktemp -d /var/lib/webycp/.server-failed.XXXXXX) || return 1
    rmdir "$failed" || return 1
    mv /var/lib/webycp/server "$failed" || return 1
    mv "$state_stage" /var/lib/webycp/server || return 1
    rm -rf -- "$failed"
}

restore_powerdns() {
    snapshot=$1
    systemctl stop pdns >/dev/null 2>&1 || true
    if [ -f "$snapshot/powerdns/PRESENT" ]; then
        install -d -o root -g root -m 0755 /etc/powerdns/pdns.d || return 1
        install -d -o root -g webycp -m 0750 /etc/webycp || return 1
        install -d -o pdns -g pdns -m 0750 /var/lib/powerdns || return 1
        install_atomic \
            "$snapshot/powerdns/webycp.conf" \
            "$POWERDNS_CONFIG" 0640 root pdns || return 1
        install_atomic \
            "$snapshot/powerdns/powerdns.key" \
            "$POWERDNS_KEY" 0600 root root || return 1
        install_atomic \
            "$snapshot/powerdns/webycp.sqlite3" \
            "$POWERDNS_DATABASE" 0640 pdns pdns || return 1
        return
    fi
    if [ -f "$snapshot/powerdns/ABSENT" ]; then
        if [ -f "$POWERDNS_CONFIG" ] &&
            [ "$(sed -n '1p' "$POWERDNS_CONFIG")" = "# Managed by WEBYCP." ]; then
            rm -f -- "$POWERDNS_CONFIG" "$POWERDNS_KEY" "$POWERDNS_DATABASE"
        fi
        systemctl disable pdns >/dev/null 2>&1 || true
        if dpkg-query -W -f='${Status}' pdns-server 2>/dev/null |
            grep -q '^install ok installed$'; then
            DEBIAN_FRONTEND=noninteractive apt-get remove -y -qq \
                pdns-backend-sqlite3 pdns-server || return 1
        fi
        return
    fi
    return 1
}

restore_snapshot() {
    snapshot=$1
    systemctl stop webycp-web webycp-server webycp-agent >/dev/null 2>&1 || true
    install_atomic \
        "$snapshot/bin/webycp-agent" \
        /usr/lib/webycp/webycp-agent 0755 root root || return 1
    install_atomic \
        "$snapshot/bin/webycp-server" \
        /usr/lib/webycp/webycp-server 0755 root root || return 1
    install_atomic \
        "$snapshot/systemd/webycp-agent.service" \
        /usr/lib/systemd/system/webycp-agent.service 0644 root root || return 1
    install_atomic \
        "$snapshot/systemd/webycp-server.service" \
        /usr/lib/systemd/system/webycp-server.service 0644 root root || return 1
    if [ -f "$snapshot/bin/node" ] &&
        [ -f "$snapshot/systemd/webycp-web.service" ]; then
        install_atomic \
            "$snapshot/bin/node" \
            /usr/lib/webycp/node 0755 root root || return 1
        install_atomic \
            "$snapshot/systemd/webycp-web.service" \
            /usr/lib/systemd/system/webycp-web.service 0644 root root || return 1
        if [ -f "$snapshot/config/web.env" ]; then
            install_atomic \
                "$snapshot/config/web.env" \
                /etc/webycp/web.env 0640 root webycp || return 1
        fi
    else
        systemctl disable webycp-web >/dev/null 2>&1 || true
        rm -f -- \
            /etc/webycp/web.env \
            /usr/lib/systemd/system/webycp-web.service \
            /usr/lib/webycp/node
    fi
    install_atomic \
        "$snapshot/VERSION" \
        /usr/lib/webycp/VERSION 0644 root root || return 1

    if [ -z "$WEB_STAGE" ] &&
        { [ -z "$WEB_OLD" ] || [ ! -d "$WEB_OLD" ]; }; then
        prepare_web "$snapshot/web" || return 1
    fi
    failed_web=$(mktemp -d /usr/share/webycp/.web-failed.XXXXXX) || return 1
    rmdir "$failed_web" || return 1
    mv /usr/share/webycp/web "$failed_web" || return 1
    if [ -n "$WEB_OLD" ] && [ -d "$WEB_OLD" ]; then
        mv "$WEB_OLD" /usr/share/webycp/web || return 1
        WEB_OLD=
    else
        mv "$WEB_STAGE" /usr/share/webycp/web || return 1
        WEB_STAGE=
    fi
    rm -rf -- "$failed_web"
    if [ -f "$snapshot/nginx/panel.conf" ]; then
        install_atomic \
            "$snapshot/nginx/panel.conf" \
            /etc/nginx/webycp/sites-available/panel.conf 0644 root root || return 1
        /usr/sbin/nginx -t || return 1
    elif [ -f "$snapshot/web/server.js" ]; then
        switch_panel_port 3000 || return 1
    else
        switch_panel_port 8080 || return 1
    fi
    restore_state "$snapshot" || return 1
    restore_powerdns "$snapshot" || return 1
}

cleanup() {
    status=$?
    trap - EXIT HUP INT TERM
    set +e

    if [ "$DONE" != true ] && [ "$STOPPED" = true ]; then
        if [ "$CHANGED" = true ] && [ -n "$BACKUP_DIR" ]; then
            log "Upgrade failed; restoring $BACKUP_DIR"
            backup_version=$(sed -n '1p' "$BACKUP_DIR/VERSION")
            if restore_snapshot "$BACKUP_DIR" &&
                start_services "$backup_version"; then
                log "Rollback complete"
            else
                log "Rollback needs manual recovery from $BACKUP_DIR"
            fi
        else
            previous=$(current_version)
            start_services "$previous" ||
                log "Services need manual recovery"
        fi
    fi

    [ -z "$WEB_STAGE" ] || rm -rf -- "$WEB_STAGE"
    if [ "$DONE" = true ] && [ -n "$WEB_OLD" ]; then
        rm -rf -- "$WEB_OLD"
    fi
    exit "$status"
}

trap cleanup EXIT
trap 'exit 1' HUP INT TERM

main() {
    require_root
    check_host
    check_install

    if [ "$MODE" = recover ]; then
        check_backup
        prepare_web "$RECOVERY_DIR/web"
        stop_services
        create_backup "before-recovery"
        CHANGED=true
        restore_snapshot "$RECOVERY_DIR"
        start_services "$RECOVERY_VERSION"
        DONE=true
        log "Recovery complete"
        log "Previous state backup: $BACKUP_DIR"
        return
    fi

    check_release
    (
        set -a
        # shellcheck disable=SC1091
        . /etc/webycp/server.env
        "$SOURCE_DIR/bin/webycp-server" check-schema
    ) || fail "database schema is incompatible; no services or release files were changed"
    webycp_check_powerdns
    check_live_install
    log "Preflight passed for $RELEASE_VERSION"
    if [ "$CHECK_ONLY" = true ]; then
        DONE=true
        return
    fi

    prepare_web "$SOURCE_DIR/web"
    stop_services
    create_backup "before-$RELEASE_VERSION"
    CHANGED=true
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    webycp_install_powerdns
    webycp_configure_powerdns
    install_files \
        "$SOURCE_DIR/bin" \
        "$SOURCE_DIR/packaging/systemd" \
        "$SOURCE_DIR/VERSION" \
        "$SOURCE_DIR/runtime/node" \
        "$SOURCE_DIR/packaging/systemd/webycp-web.service"
    ensure_web_config
    switch_panel_port 3000
    run_migrations
    start_services "$RELEASE_VERSION"
    DONE=true
    log "Upgrade to $RELEASE_VERSION complete"
    log "Recovery backup: $BACKUP_DIR"
}

main

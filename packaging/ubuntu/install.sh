#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
SOURCE_DIR=$(CDPATH='' cd -- "$SCRIPT_DIR/../.." && pwd)
START_SERVICES=true

log() {
    printf '%s\n' "WEBYCP: $*"
}

fail() {
    printf '%s\n' "WEBYCP: $*" >&2
    exit 1
}

usage() {
    printf '%s\n' "Usage: $0 [--source DIR] [--no-start]"
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --source)
            [ "$#" -ge 2 ] || fail "--source requires a directory"
            [ -d "$2" ] || fail "source directory does not exist: $2"
            SOURCE_DIR=$(CDPATH='' cd -- "$2" && pwd)
            shift 2
            ;;
        --no-start)
            START_SERVICES=false
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

require_root() {
    [ "$(id -u)" -eq 0 ] || fail "the installer must run as root"
}

check_host() {
    [ -r /etc/os-release ] || fail "cannot identify the operating system"
    # shellcheck disable=SC1091
    . /etc/os-release
    if [ "${ID:-}" != "ubuntu" ] || [ "${VERSION_ID:-}" != "24.04" ]; then
        fail "Ubuntu 24.04 is required"
    fi
    [ "$(dpkg --print-architecture)" = "amd64" ] || fail "amd64 is required"
}

check_binary() {
    binary=$1
    magic=$(od -An -j0 -N4 -t x1 "$binary" | tr -d ' \n')
    machine=$(od -An -j18 -N2 -t x1 "$binary" | tr -d ' \n')
    if [ "$magic" != "7f454c46" ] || [ "$machine" != "3e00" ]; then
        fail "not a Linux amd64 binary: $binary"
    fi
}

check_source() {
    for path in \
        bin/webycp-agent \
        bin/webycp-server \
        runtime/node \
        web/server.js \
        packaging/nginx/panel-bootstrap.conf \
        packaging/nginx/webycp.conf \
        packaging/systemd/webycp-agent.service \
        packaging/systemd/webycp-server.service \
        packaging/systemd/webycp-web.service \
        packaging/ubuntu/agent.env.example \
        packaging/ubuntu/server.env.example \
        packaging/ubuntu/upgrade.sh \
        packaging/ubuntu/web.env.example \
        VERSION
    do
        [ -f "$SOURCE_DIR/$path" ] || fail "release file is missing: $path"
    done
    check_binary "$SOURCE_DIR/bin/webycp-agent"
    check_binary "$SOURCE_DIR/bin/webycp-server"
    check_binary "$SOURCE_DIR/runtime/node"
    [ -d "$SOURCE_DIR/web/.next/static" ] ||
        fail "release directory is missing: web/.next/static"
}

install_packages() {
    log "Installing Ubuntu packages"
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq --no-install-recommends \
        ca-certificates \
        certbot \
        cron \
        curl \
        mysql-server \
        nginx \
        openssl \
        php8.3-cli \
        php8.3-fpm \
        php8.3-mysql
}

ensure_identity() {
    if group_entry=$(getent group webycp); then
        group_gid=$(printf '%s' "$group_entry" | cut -d: -f3)
        [ "$group_gid" -lt 1000 ] || fail "existing webycp group is not a system group"
    else
        groupadd --system webycp
        group_gid=$(getent group webycp | cut -d: -f3)
    fi

    if user_entry=$(getent passwd webycp); then
        user_uid=$(printf '%s' "$user_entry" | cut -d: -f3)
        user_gid=$(printf '%s' "$user_entry" | cut -d: -f4)
        user_home=$(printf '%s' "$user_entry" | cut -d: -f6)
        user_shell=$(printf '%s' "$user_entry" | cut -d: -f7)
        if [ "$user_uid" -ge 1000 ] ||
            [ "$user_gid" != "$group_gid" ] ||
            [ "$user_home" != "/var/lib/webycp/server" ] ||
            [ "$user_shell" != "/usr/sbin/nologin" ]; then
            fail "existing webycp user does not match the required service identity"
        fi
    else
        useradd \
            --system \
            --gid webycp \
            --home-dir /var/lib/webycp/server \
            --no-create-home \
            --shell /usr/sbin/nologin \
            --comment "WEBYCP server" \
            webycp
    fi
}

prepare_directories() {
    install -d -o root -g root -m 0755 /usr/lib/webycp
    install -d -o root -g root -m 0755 /usr/share/webycp
    install -d -o root -g root -m 0755 /usr/share/webycp/web
    install -d -o root -g webycp -m 0750 /etc/webycp
    install -d -o root -g root -m 0700 /etc/webycp/bootstrap
    install -d -o root -g root -m 0755 /var/lib/webycp
    install -d -o webycp -g webycp -m 0700 /var/lib/webycp/server
    install -d -o root -g root -m 0700 /home/.webycp-trash
    install -d -o root -g webycp -m 0755 /var/lib/webycp/acme
    install -d -o root -g root -m 0700 /var/backups/webycp
    install -d -o root -g root -m 0755 /etc/nginx/webycp/sites-available
    install -d -o root -g root -m 0755 /etc/nginx/webycp/sites-enabled
}

install_atomic() {
    source=$1
    target=$2
    mode=$3
    owner=$4
    group=$5
    temporary=$(mktemp "${target}.new.XXXXXX")
    trap 'rm -f -- "$temporary"' EXIT HUP INT TERM
    install -o "$owner" -g "$group" -m "$mode" "$source" "$temporary"
    mv -f -- "$temporary" "$target"
    trap - EXIT HUP INT TERM
}

install_release() {
    log "Installing WEBYCP release files"
    install_atomic "$SOURCE_DIR/bin/webycp-agent" /usr/lib/webycp/webycp-agent 0755 root root
    install_atomic "$SOURCE_DIR/bin/webycp-server" /usr/lib/webycp/webycp-server 0755 root root
    install_atomic "$SOURCE_DIR/runtime/node" /usr/lib/webycp/node 0755 root root
    install_atomic "$SOURCE_DIR/VERSION" /usr/lib/webycp/VERSION 0644 root root

    cp -a "$SOURCE_DIR/web/." /usr/share/webycp/web/
    chown -R root:root /usr/share/webycp/web
    find /usr/share/webycp/web -type d -exec chmod 0755 {} +
    find /usr/share/webycp/web -type f -exec chmod 0644 {} +

    install_atomic \
        "$SOURCE_DIR/packaging/systemd/webycp-agent.service" \
        /usr/lib/systemd/system/webycp-agent.service 0644 root root
    install_atomic \
        "$SOURCE_DIR/packaging/systemd/webycp-server.service" \
        /usr/lib/systemd/system/webycp-server.service 0644 root root
    install_atomic \
        "$SOURCE_DIR/packaging/systemd/webycp-web.service" \
        /usr/lib/systemd/system/webycp-web.service 0644 root root
}

ensure_config() {
    source=$1
    target=$2
    mode=$3
    owner=$4
    group=$5
    if [ -L "$target" ]; then
        fail "configuration path must not be a symlink: $target"
    fi
    if [ -e "$target" ]; then
        [ -f "$target" ] || fail "configuration path is not a regular file: $target"
        chown "$owner:$group" "$target"
        chmod "$mode" "$target"
        return
    fi
    install -o "$owner" -g "$group" -m "$mode" "$source" "$target"
}

install_config() {
    ensure_config \
        "$SOURCE_DIR/packaging/ubuntu/agent.env.example" \
        /etc/webycp/agent.env 0600 root root
    ensure_config \
        "$SOURCE_DIR/packaging/ubuntu/server.env.example" \
        /etc/webycp/server.env 0640 root webycp
    ensure_config \
        "$SOURCE_DIR/packaging/ubuntu/web.env.example" \
        /etc/webycp/web.env 0640 root webycp
}

ensure_bootstrap_certificate() {
    certificate=/etc/webycp/bootstrap/tls.crt
    key=/etc/webycp/bootstrap/tls.key
    if [ -e "$certificate" ] || [ -e "$key" ]; then
        if [ ! -f "$certificate" ] || [ -L "$certificate" ] ||
            [ ! -f "$key" ] || [ -L "$key" ]; then
            fail "bootstrap TLS paths must be regular files"
        fi
        chmod 0644 "$certificate"
        chmod 0600 "$key"
        chown root:root "$certificate" "$key"
        return
    fi

    stage=$(mktemp -d /etc/webycp/bootstrap/.tls.XXXXXX)
    trap 'rm -rf -- "$stage"' EXIT HUP INT TERM
    openssl req \
        -x509 \
        -nodes \
        -newkey rsa:2048 \
        -sha256 \
        -days 30 \
        -subj /CN=webycp-bootstrap \
        -keyout "$stage/tls.key" \
        -out "$stage/tls.crt" \
        >/dev/null 2>&1
    install -o root -g root -m 0600 "$stage/tls.key" "$key"
    install -o root -g root -m 0644 "$stage/tls.crt" "$certificate"
    rm -rf -- "$stage"
    trap - EXIT HUP INT TERM
}

install_nginx() {
    include=/etc/nginx/conf.d/webycp.conf
    panel=/etc/nginx/webycp/sites-available/panel.conf
    link=/etc/nginx/webycp/sites-enabled/panel.conf
    include_created=false
    panel_created=false
    link_created=false

    if [ -L "$include" ]; then
        fail "Nginx include must not be a symlink: $include"
    elif [ -e "$include" ]; then
        if [ ! -f "$include" ] ||
            ! cmp -s "$SOURCE_DIR/packaging/nginx/webycp.conf" "$include"; then
            fail "Nginx include exists with unexpected contents: $include"
        fi
    else
        install -o root -g root -m 0644 "$SOURCE_DIR/packaging/nginx/webycp.conf" "$include"
        include_created=true
    fi

    if [ -L "$panel" ]; then
        fail "panel Nginx configuration must not be a symlink: $panel"
    elif [ -e "$panel" ]; then
        if [ ! -f "$panel" ] ||
            [ "$(head -n 1 "$panel")" != "# Managed by WEBYCP." ]; then
            fail "panel Nginx configuration is not managed by WEBYCP"
        fi
        if cmp -s "$SOURCE_DIR/packaging/nginx/panel-bootstrap.conf" "$panel"; then
            ensure_bootstrap_certificate
        fi
    else
        ensure_bootstrap_certificate
        install -o root -g root -m 0644 \
            "$SOURCE_DIR/packaging/nginx/panel-bootstrap.conf" "$panel"
        panel_created=true
    fi

    if [ -L "$link" ]; then
        [ "$(readlink "$link")" = "../sites-available/panel.conf" ] ||
            fail "panel Nginx symlink has an unexpected target"
    elif [ -e "$link" ]; then
        fail "panel Nginx enabled path is not a symlink: $link"
    else
        ln -s ../sites-available/panel.conf "$link"
        link_created=true
    fi

    if ! /usr/sbin/nginx -t; then
        [ "$link_created" = false ] || rm -f -- "$link"
        [ "$panel_created" = false ] || rm -f -- "$panel"
        [ "$include_created" = false ] || rm -f -- "$include"
        fail "Nginx rejected the WEBYCP bootstrap configuration"
    fi
}

ensure_admin() {
    if [ "$START_SERVICES" = false ]; then
        log "Administrator initialization skipped; run webycp-server admin init before first startup"
        return
    fi
    log "Initializing the administrator account"
    /usr/sbin/runuser -u webycp -- \
        /usr/lib/webycp/webycp-server admin init
}

wait_for_socket() {
    attempts=0
    while [ "$attempts" -lt 20 ]; do
        [ -S /run/webycp/agent.sock ] && return
        attempts=$((attempts + 1))
        sleep 1
    done
    fail "Agent socket did not become ready"
}

wait_for_server() {
    attempts=0
    while [ "$attempts" -lt 20 ]; do
        if curl --fail --silent --show-error \
            http://127.0.0.1:8080/api/v1/health >/dev/null 2>&1; then
            return
        fi
        attempts=$((attempts + 1))
        sleep 1
    done
    fail "Server health endpoint did not become ready"
}

wait_for_web() {
    attempts=0
    while [ "$attempts" -lt 20 ]; do
        if curl --fail --silent --show-error \
            http://127.0.0.1:3000/login >/dev/null 2>&1; then
            return
        fi
        attempts=$((attempts + 1))
        sleep 1
    done
    fail "Web frontend did not become ready"
}

start_services() {
    if [ "$START_SERVICES" = false ]; then
        log "Files installed; service startup was skipped"
        return
    fi
    [ -d /run/systemd/system ] ||
        fail "systemd is not running; use --no-start only when building an image"

    log "Enabling and starting services"
    systemctl daemon-reload
    systemctl enable nginx mysql php8.3-fpm cron webycp-agent webycp-server webycp-web >/dev/null
    systemctl start nginx mysql php8.3-fpm cron
    systemctl restart webycp-agent
    wait_for_socket
    systemctl restart webycp-server
    wait_for_server
    systemctl restart webycp-web
    wait_for_web
    systemctl reload nginx
}

main() {
    require_root
    check_host
    check_source
    install_packages
    ensure_identity
    prepare_directories
    install_release
    install_config
    install_nginx
    ensure_admin
    start_services
    log "Installation complete"
    log "Panel URL: https://SERVER_IP:8443"
}

main

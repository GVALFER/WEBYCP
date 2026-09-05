#!/bin/sh

POWERDNS_CONFIG=/etc/powerdns/pdns.d/webycp.conf
POWERDNS_KEY=/etc/webycp/powerdns.key
POWERDNS_DATABASE=/var/lib/powerdns/webycp.sqlite3
POWERDNS_SCHEMA=/usr/share/doc/pdns-backend-sqlite3/schema.sqlite3.sql

webycp_check_powerdns() {
    for path in "$POWERDNS_CONFIG" "$POWERDNS_KEY" "$POWERDNS_DATABASE"; do
        [ ! -L "$path" ] || fail "PowerDNS path must not be a symlink: $path"
    done
    if [ -e "$POWERDNS_CONFIG" ]; then
        [ -f "$POWERDNS_CONFIG" ] || fail "PowerDNS configuration is not a regular file"
        [ "$(sed -n '1p' "$POWERDNS_CONFIG")" = "# Managed by WEBYCP." ] ||
            fail "PowerDNS configuration is not managed by WEBYCP"
        return
    fi
    if [ -e "$POWERDNS_KEY" ] || [ -e "$POWERDNS_DATABASE" ]; then
        fail "PowerDNS WEBYCP paths exist without managed configuration"
    fi
    if dpkg-query -W -f='${Status}' pdns-server 2>/dev/null |
        grep -q '^install ok installed$'; then
        fail "an existing PowerDNS installation is not managed by WEBYCP"
    fi
}

webycp_install_powerdns() (
    policy_created=false
    policy=/usr/sbin/policy-rc.d
    policy_stage=
    cleanup_policy() {
        if [ "$policy_created" = true ] &&
            [ -f "$policy" ] &&
            [ "$(sed -n '2p' "$policy")" = "# Managed temporarily by WEBYCP." ]; then
            rm -f -- "$policy"
        fi
        [ -z "$policy_stage" ] || rm -f -- "$policy_stage"
    }
    trap cleanup_policy EXIT
    trap 'exit 1' HUP INT TERM
    if [ ! -e "$policy" ]; then
        policy_stage=$(mktemp /usr/sbin/.webycp-policy-rc.d.XXXXXX)
        {
            printf '%s\n' '#!/bin/sh'
            printf '%s\n' '# Managed temporarily by WEBYCP.'
            printf '%s\n' 'exit 101'
        } >"$policy_stage"
        chmod 0755 "$policy_stage"
        if ln "$policy_stage" "$policy" 2>/dev/null; then
            policy_created=true
        fi
        rm -f -- "$policy_stage"
        policy_stage=
    fi
    apt-get install -y -qq --no-install-recommends \
        pdns-backend-sqlite3 \
        pdns-server \
        sqlite3
    cleanup_policy
    trap - EXIT HUP INT TERM
)

webycp_powerdns_addresses() {
    addresses=$(
        ip -o addr show scope global up |
            awk '$3 == "inet" || $3 == "inet6" { sub(/\/.*/, "", $4); values = values separator $4; separator = "," } END { print values }'
    )
    [ -n "$addresses" ] || fail "PowerDNS requires at least one global server address"
    printf '%s\n' "$addresses"
}

webycp_configure_powerdns() {
    for path in "$POWERDNS_CONFIG" "$POWERDNS_KEY" "$POWERDNS_DATABASE"; do
        [ ! -L "$path" ] || fail "PowerDNS path must not be a symlink: $path"
    done
    [ -f "$POWERDNS_SCHEMA" ] || fail "PowerDNS SQLite schema is missing"
    install -d -o root -g root -m 0755 /etc/powerdns/pdns.d
    install -d -o pdns -g pdns -m 0750 /var/lib/powerdns

    if [ ! -e "$POWERDNS_KEY" ]; then
        key_stage=$(mktemp /etc/webycp/.powerdns-key.XXXXXX)
        openssl rand -hex 32 >"$key_stage"
        chown root:root "$key_stage"
        chmod 0600 "$key_stage"
        mv -f -- "$key_stage" "$POWERDNS_KEY"
    fi
    [ -f "$POWERDNS_KEY" ] || fail "PowerDNS API key is not a regular file"
    chown root:root "$POWERDNS_KEY"
    chmod 0600 "$POWERDNS_KEY"
    powerdns_key=$(sed -n '1p' "$POWERDNS_KEY")
    printf '%s\n' "$powerdns_key" | grep -Eq '^[0-9a-f]{64}$' ||
        fail "PowerDNS API key is invalid"

    if [ ! -e "$POWERDNS_DATABASE" ]; then
        database_stage=$(mktemp /var/lib/powerdns/.webycp.XXXXXX)
        sqlite3 "$database_stage" <"$POWERDNS_SCHEMA"
        chown pdns:pdns "$database_stage"
        chmod 0640 "$database_stage"
        mv -f -- "$database_stage" "$POWERDNS_DATABASE"
    fi
    [ -f "$POWERDNS_DATABASE" ] || fail "PowerDNS database is not a regular file"
    chown pdns:pdns "$POWERDNS_DATABASE"
    chmod 0640 "$POWERDNS_DATABASE"

    addresses=$(webycp_powerdns_addresses)
    config_stage=$(mktemp /etc/powerdns/pdns.d/.webycp.XXXXXX)
    {
        printf '%s\n' '# Managed by WEBYCP.'
        printf '%s\n' 'launch=gsqlite3'
        printf '%s\n' "gsqlite3-database=$POWERDNS_DATABASE"
        printf '%s\n' 'api=yes'
        printf '%s\n' "api-key=$powerdns_key"
        printf '%s\n' 'webserver=yes'
        printf '%s\n' 'webserver-address=127.0.0.1'
        printf '%s\n' 'webserver-port=8081'
        printf '%s\n' 'webserver-allow-from=127.0.0.1,::1'
        printf '%s\n' "local-address=$addresses"
        printf '%s\n' 'local-port=53'
    } >"$config_stage"
    chown root:pdns "$config_stage"
    chmod 0640 "$config_stage"
    mv -f -- "$config_stage" "$POWERDNS_CONFIG"

    pdns_server --config=check >/dev/null
}

webycp_wait_powerdns() {
    attempts=0
    while [ "$attempts" -lt 20 ]; do
        if systemctl is-active --quiet pdns &&
            powerdns_key=$(sed -n '1p' "$POWERDNS_KEY") &&
            curl --fail --silent --show-error \
                -H "X-API-Key: $powerdns_key" \
                http://127.0.0.1:8081/api/v1/servers/localhost >/dev/null 2>&1; then
            return
        fi
        attempts=$((attempts + 1))
        sleep 1
    done
    return 1
}

webycp_start_powerdns() {
    systemctl enable pdns >/dev/null || return 1
    systemctl restart pdns || return 1
    webycp_wait_powerdns
}

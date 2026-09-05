# WEBYCP operations and recovery

This runbook applies to the native Ubuntu 24.04 installation described in the
[packaging guide](../packaging/README.md). Run host commands with `sudo`. Do not
edit the SQLite database, WEBYCP-managed Nginx files, backup manifests, or Agent
requests by hand.

## Health check

Check the control plane before and after maintenance:

```sh
sudo systemctl is-active webycp-agent webycp-server webycp-web nginx mysql php8.3-fpm cron pdns
curl --fail --silent --show-error http://127.0.0.1:8080/api/v1/health
curl --fail --silent --show-error http://127.0.0.1:3000/login >/dev/null
sudo curl --fail --silent --show-error \
    --unix-socket /run/webycp/agent.sock \
    http://localhost/agent/v1/health
sudo nginx -t
```

The two health responses must report `status` as `ok`. The Agent response must
also report `protocolVersion` as `v1`. Inspect recent service logs when a check
fails. Immediately after a start or restart, retry the health endpoints for up
to 20 seconds because systemd can report `active` before the listeners are
ready.

```sh
sudo journalctl \
    -u webycp-agent \
    -u webycp-server \
    -u webycp-web \
    -u nginx \
    --since "-30 minutes" \
    --no-pager
```

Do not expose ports 3000 or 8080, or the Agent socket, publicly. They are local
control-plane interfaces.

## Authoritative DNS

PowerDNS listens on TCP and UDP port 53 of the server's global addresses. Its
HTTP API is restricted to `127.0.0.1:8081`. Allow inbound TCP and UDP port 53
before delegating a public zone, then configure two distinct nameserver
hostnames under **DNS > Nameservers**. The hostnames and any required registrar
glue records must resolve to authoritative server addresses.

WEBYCP creates zones independently from website hostname bindings. Creating a
website does not publish DNS automatically. Each managed PowerDNS zone carries
an internal WEBYCP ownership marker; the Agent will not overwrite or delete a
foreign zone with the same name.

Check PowerDNS without printing its API key:

```sh
sudo systemctl --no-pager --full status pdns
sudo pdns_control ping
dig @SERVER_IP example.com SOA
sudo stat -c '%U:%G %a %n' \
    /etc/webycp/powerdns.key \
    /var/lib/powerdns/webycp.sqlite3
```

Expected ownership and modes are `root:root 600` for the API key and
`pdns:pdns 640` for the authoritative database. Use the panel to change zones
and records; do not edit the PowerDNS SQLite database directly.

## Administrator access

The installer creates `admin` with a generated temporary password. The first
authenticated session can access only the profile, session, and logout
endpoints until the username, name, email, and password are confirmed.

If an administrator password is lost, generate a new temporary password from a
root shell without editing SQLite:

```sh
sudo /usr/sbin/runuser -u webycp -- \
    /usr/lib/webycp/webycp-server admin reset-password USERNAME
```

The command prints the replacement once, invalidates every existing session for
that user, and requires another password change at login. Do not pass passwords
through command-line arguments, environment variables, or service logs.

## Account backups

Create or edit plans from **Backups → Plans** in the panel. A plan belongs to one active
hosting account and may contain its files, databases, or both. An empty schedule
makes the plan manual-only; scheduled plans use standard five-field cron syntax
in UTC. Retention is between 1 and 100 completed artifacts per plan.

Use **Run now**, then confirm that the run is `succeeded` and that a new entry
appears under **Backups → Archives**. An artifact is not usable until both have
happened.

Artifacts are stored as root-only files at:

```text
/var/backups/webycp/<account-id>/<run-id>.tar.gz
```

Do not rename or move individual artifacts. Their exact path and SHA-256 value
are recorded in SQLite.

**Backups → Destinations** shows the storage reported by each node and when it
was checked. Use **Check agent** to refresh this observation. The local driver
stores archives on the account's node; a node ID plus a storage driver identifies
the destination. There are no remote credentials or destination configuration
records yet. A remote provider must be selected before adding its implementation.

All drivers implement creation, verified preview, scoped restore and idempotent
deletion. Preview and restore must verify identity, entry paths and checksums
before writing. Restore must reject absent scopes and repair managed ownership;
the control plane reconciles selected metadata after the Agent succeeds.

### Backup coverage

| Scope     | Included                                                       | Not included                                                                |
| --------- | -------------------------------------------------------------- | --------------------------------------------------------------------------- |
| Files     | The managed account home directory                             | Symlinks; their presence fails the backup                                   |
| Databases | Dumps of active databases, including routines and events       | MySQL users, passwords, and grants                                          |
| Metadata  | Websites, domain bindings, database definitions, and scheduled tasks | Accounts, DNS zones, sessions, backup plans, certificates, and private keys |

Local artifacts share the managed server's failure domain. Copy
`/var/backups/webycp` to protected off-host storage or include it in a verified
infrastructure snapshot. Preserve ownership and permissions.

## Restore procedure

1. Run and verify a fresh backup of the current account state.
2. Confirm that the account is active and that all health checks pass.
3. Open **Backups → Restore** and select the required archive. Opening its modal
   verifies the actual archive on its server before offering restore options.
4. Review the archive identity and select files, database contents, metadata,
   or all available scopes. Absent content cannot be selected. Submit only after
   reviewing the overwrite warning; the Agent verifies the archive again when
   the restore job executes.
5. Monitor **Jobs** until the restore job is `succeeded`.
6. Test the restored websites, database contents, and cron definitions.
7. Recreate or rotate database-user credentials and grants when required; they
   are deliberately not retained in backups.

Before restoration, WEBYCP verifies the artifact SHA-256, manifest identity,
entry checksums, and paths. Restore overlays matching files and databases; it
does not delete unrelated content to recreate an exact point-in-time image.
Files, databases, and metadata are not restored inside one global transaction,
so a failed restore must be investigated before it is retried. The fresh backup
from step 1 is the recovery point.

When metadata is restored, enabled domains and scheduled tasks are reconciled with the
host. Existing active certificate records are reapplied, but certificate files
and certificate records are not created from the account artifact.

## Certificate operations

Before issuing a certificate:

- Point the primary hostname to a global address on this server.
- Point every alias that should be included to the same server.
- Allow inbound TCP ports 80 and 443. HTTP-01 validation requires port 80.
- Keep Nginx healthy and do not replace WEBYCP's ACME challenge location.

Issue and renew certificates from **Certificates** in the panel. Domain
certificates include only aliases that pass DNS preflight. A panel certificate
replaces the temporary port 8443 listener with the final hostname on port 443.
WEBYCP checks hourly for certificates whose renewal time has arrived, beginning
30 days before expiry.

For a failed issue or renewal:

1. Read the certificate error and its job under **Jobs**.
2. Confirm DNS with the authoritative records and verify ports 80 and 443 from
   outside the server.
3. Run the health check and inspect the Agent and Nginx logs.
4. Inspect Certbot's known certificates without reading private keys:

    ```sh
    sudo certbot certificates
    ```

5. Correct DNS, firewall, or unrelated Nginx errors, then use **Renew** in the
   panel. Use the panel operation instead of an ad-hoc Certbot command so that
   certificate state and Nginx configuration remain synchronized.

The Agent validates Nginx before every reload and restores the previous managed
configuration if validation or reload fails. A failed renewal does not replace
the last working panel configuration. If `/etc/letsencrypt` is lost or corrupt,
restore it together with the matching WEBYCP and Nginx state from a host
snapshot; account backup artifacts do not contain private keys.

## Agent recovery

Systemd restarts the Agent after ordinary process failures. If it does not
recover, first capture its status and logs:

```sh
sudo systemctl --no-pager --full status webycp-agent webycp-server webycp-web
sudo journalctl -u webycp-agent -u webycp-server -u webycp-web -b --no-pager
sudo stat -c '%U:%G %a %n' \
    /etc/webycp/agent.env \
    /run/webycp \
    /run/webycp/agent.sock
```

Expected ownership and modes are `root:root 600` for `agent.env`,
`root:webycp 750` for `/run/webycp`, and `root:webycp 660` for the socket. The
socket exists only while the Agent is running.

Recover the service boundary in this order so that the Server cannot consume
queued host jobs while the Agent is unavailable:

```sh
sudo systemctl stop webycp-web
sudo systemctl stop webycp-server
sudo systemctl restart webycp-agent
attempt=0
until sudo curl --fail --silent --show-error \
        --unix-socket /run/webycp/agent.sock \
        http://localhost/agent/v1/health; do
    attempt=$((attempt + 1))
    [ "$attempt" -lt 20 ] || exit 1
    sleep 1
done
sudo systemctl start webycp-server
attempt=0
until curl --fail --silent --show-error \
        http://127.0.0.1:8080/api/v1/health; do
    attempt=$((attempt + 1))
    [ "$attempt" -lt 20 ] || exit 1
    sleep 1
done
sudo systemctl start webycp-web
```

Then run **Probe** for the local node in the dashboard and inspect **Jobs**.
Jobs that were `running` when the Server stopped are requeued on startup. Jobs
already marked `failed` are not replayed silently; after correcting the cause,
submit the operation again through the panel.

If an installed binary, systemd unit, frontend build, or SQLite state is
damaged, recover an exact upgrade snapshot using the release's recovery
command:

```sh
sudo ./packaging/ubuntu/upgrade.sh \
    --recover /var/lib/webycp/upgrades/EXACT_BACKUP_DIRECTORY
```

The command validates the selected snapshot, creates another safety snapshot,
restores matching binaries and state, and verifies the Agent, Server, Nginx,
and version before completing. If no upgrade snapshot exists, verify and
extract the same release version and rerun its idempotent installer. Preserve
`/etc/webycp` and take an infrastructure snapshot first.

## Full-host recovery boundary

An upgrade snapshot is a software rollback point, not a full server backup. It
contains the web environment and panel Nginx file needed for release rollback,
but not the Agent/Server environment, Let's Encrypt material, customer homes,
MySQL data, hosted-site Nginx configuration, or local backup artifacts.

For full-host recovery, maintain a tested VM/provider snapshot or an equivalent
off-host backup containing at least:

- `/var/lib/webycp/server`
- `/var/backups/webycp`
- `/home/wcp_*` and `/home/.webycp-trash`
- `/etc/webycp`
- `/var/lib/powerdns/webycp.sqlite3`
- `/etc/nginx/webycp` and `/etc/nginx/conf.d/webycp.conf`
- `/etc/letsencrypt`
- the MySQL state or restorable dumps
- `/usr/lib/webycp/VERSION`

Do not copy a live SQLite directory or MySQL data directory as an ordinary file
backup. Use a coordinated snapshot or stop the affected services. Bare-metal
restore automation is not part of v1, so rehearse the infrastructure restore
procedure before relying on it in production.

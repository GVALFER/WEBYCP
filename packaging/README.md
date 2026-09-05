# Ubuntu packaging

These files define the production layout for Ubuntu 24.04. The installer
consumes an already-built Linux amd64 release, so Go and Node.js are not needed
on the managed server.

## Install

Download the release archive and its adjacent `.sha256` file, then verify and
extract it. For release `0.1.0`:

```sh
sha256sum --check webycp-0.1.0-linux-amd64.tar.gz.sha256
tar -xzf webycp-0.1.0-linux-amd64.tar.gz
cd webycp-0.1.0-linux-amd64
sudo ./packaging/ubuntu/install.sh
```

The extracted release contains Linux amd64 binaries under `bin/`, a bundled
Node.js executable under `runtime/`, and the Next.js standalone build under
`web/`. The installer also accepts `--source DIR`
when the release files live elsewhere. Run:

```sh
sudo ./packaging/ubuntu/install.sh
```

`--no-start` installs and validates the files without enabling or starting
services; it is intended only for image construction and automated tests. It
does not create the administrator, preventing temporary credentials from being
captured in image-build logs. Before the first service startup, initialize it
inside the provisioned server:

```sh
sudo /usr/sbin/runuser -u webycp -- \
    /usr/lib/webycp/webycp-server admin init
```

The installer creates the initial `admin` user and prints a random temporary
password once. Sign in at `https://SERVER_IP:8443`, complete the administrator
name and email, and replace the temporary password before the rest of the panel
becomes available. Port 8443 uses a temporary self-signed certificate and is
isolated from hosted websites. After a panel hostname resolves to the server,
issue its Let's Encrypt certificate in the UI; the Agent then replaces the
bootstrap listener with the final HTTPS listener on port 443.

The installer is idempotent and preserves existing environment files and
WEBYCP-managed Nginx and PowerDNS configuration. It stops on conflicting identities,
symlinks, or configuration owned by another application. An existing PowerDNS
installation is not adopted automatically.

## Upgrade and recovery

Verify and extract the new release beside its `.sha256` file, then run the
preflight and upgrade from the extracted directory:

```sh
sudo ./packaging/ubuntu/upgrade.sh --check
sudo ./packaging/ubuntu/upgrade.sh
```

The upgrade requires a healthy existing installation. It stops only the WEBYCP
Web, Server, and Agent services, then snapshots the current binaries, frontend,
systemd units, web environment, panel Nginx configuration, control-plane SQLite
state, and WEBYCP's PowerDNS configuration, key, and SQLite state.
Database migrations run as the unprivileged `webycp` user before the services
start. Nginx is validated, the Agent socket, Server API, and Next.js frontend
must become ready, and the running version must match the release marker.

If any step fails after shutdown, the command automatically restores the
snapshot and restarts the previous version. Successful upgrade snapshots remain
under `/var/lib/webycp/upgrades` with mode `0700`. To recover one explicitly:

```sh
sudo ./packaging/ubuntu/upgrade.sh \
    --recover /var/lib/webycp/upgrades/EXACT_BACKUP_DIRECTORY
```

Recovery accepts only an exact snapshot directory below that root and creates a
new safety snapshot before changing the installation. Agent/Server
configuration, certificates, customer home directories, and backup artifacts
are not replaced by either operation.

For health checks, account restore, certificate troubleshooting, Agent recovery,
and the full-host recovery boundary, follow the
[operations runbook](../docs/operations.md).

## Service boundary

- `webycp-server` runs as the unprivileged `webycp` user with no Linux
  capabilities. It can write only its private state directory.
- `webycp-web` runs the bundled Next.js standalone server as the same
  unprivileged user and forwards API requests to the loopback Go service.
- `webycp-agent` runs as root because it owns the narrow, typed host operations.
  Its Unix socket is available to the `webycp` group and is never exposed over
  TCP.
- The Agent keeps permission to create set-group-ID customer directories; that
  permission is required by the hosted-site ownership model.
- All three services write logs to stdout for collection by journald.
- PowerDNS serves authoritative DNS on the server's global addresses. Its HTTP
  API listens only on `127.0.0.1:8081`; only the root Agent reads its API key.

## Filesystem layout

| Path | Owner | Mode | Purpose |
| --- | --- | --- | --- |
| `/usr/lib/webycp` | `root:root` | `0755` | Server, Agent, and bundled Node.js binaries |
| `/usr/share/webycp/web` | `root:root` | `0755` | Next.js standalone production build |
| `/etc/webycp` | `root:webycp` | `0750` | Service environment files |
| `/etc/webycp/bootstrap` | `root:root` | `0700` | Temporary bootstrap TLS material |
| `/etc/webycp/powerdns.key` | `root:root` | `0600` | Local PowerDNS API key read by the Agent |
| `/var/lib/powerdns/webycp.sqlite3` | `pdns:pdns` | `0640` | Authoritative zones and records |
| `/var/lib/webycp/server` | `webycp:webycp` | `0700` | SQLite database and WAL files |
| `/var/lib/webycp/upgrades` | `root:root` | `0700` | Upgrade and recovery snapshots |
| `/home/.webycp-trash` | `root:root` | `0700` | Recoverable deleted accounts on the hosting filesystem |
| `/var/lib/webycp/acme` | `root:webycp` | `0755` | HTTP-01 challenge webroot |
| `/var/backups/webycp` | `root:root` | `0700` | Local backup artifacts |
| `/run/webycp` | `root:webycp` | `0750` | Agent socket directory |
| `/run/webycp/agent.sock` | `root:webycp` | `0660` | Private Server-Agent transport |

The installer must create the `webycp` system user and group before enabling
the units. `server.env` and `web.env` must be owned by `root:webycp` with mode
`0640`; `agent.env` must be owned by `root:root` with mode `0600`.

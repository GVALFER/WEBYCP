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

The extracted release contains Linux amd64 binaries under `bin/` and the Vite
production build under `web/dist/`. The installer also accepts `--source DIR`
when the release files live elsewhere. Run:

```sh
sudo ./packaging/ubuntu/install.sh
```

`--no-start` installs and validates the files without enabling or starting
services; it is intended only for image construction and automated tests.

The first login is available at `https://SERVER_IP:8443` using a temporary
self-signed certificate. Port 8443 is isolated from hosted websites. After a
panel hostname resolves to the server, issue its Let's Encrypt certificate in
the UI; the Agent then replaces the bootstrap listener with the final HTTPS
listener on port 443.

The installer is idempotent and preserves existing environment files and
WEBYCP-managed Nginx panel configuration. It stops on conflicting identities,
symlinks, or configuration owned by another application.

## Service boundary

- `webycp-server` runs as the unprivileged `webycp` user with no Linux
  capabilities. It can write only its private state directory.
- `webycp-agent` runs as root because it owns the narrow, typed host operations.
  Its Unix socket is available to the `webycp` group and is never exposed over
  TCP.
- The Agent keeps permission to create set-group-ID customer directories; that
  permission is required by the hosted-site ownership model.
- Both services write structured logs to stdout for collection by journald.

## Filesystem layout

| Path | Owner | Mode | Purpose |
| --- | --- | --- | --- |
| `/usr/lib/webycp` | `root:root` | `0755` | Server and Agent binaries |
| `/usr/share/webycp/web` | `root:root` | `0755` | React production build |
| `/etc/webycp` | `root:webycp` | `0750` | Service environment files |
| `/etc/webycp/bootstrap` | `root:root` | `0700` | Temporary bootstrap TLS material |
| `/var/lib/webycp/server` | `webycp:webycp` | `0700` | SQLite database and WAL files |
| `/home/.webycp-trash` | `root:root` | `0700` | Recoverable deleted accounts on the hosting filesystem |
| `/var/lib/webycp/acme` | `root:webycp` | `0755` | HTTP-01 challenge webroot |
| `/var/backups/webycp` | `root:root` | `0700` | Local backup artifacts |
| `/run/webycp` | `root:webycp` | `0750` | Agent socket directory |
| `/run/webycp/agent.sock` | `root:webycp` | `0660` | Private Server-Agent transport |

The installer must create the `webycp` system user and group before enabling
the units. `server.env` must be owned by `root:webycp` with mode `0640`;
`agent.env` must be owned by `root:root` with mode `0600`.

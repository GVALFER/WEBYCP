# WEBYCP

WEBYCP is a self-hosted, open-source hosting control panel for Ubuntu 24.04.

The v1 feature and native packaging baseline is now present:

- Next.js App Router frontend with SSR, HeroUI, Tailwind, Lucide, SWR,
  `reqly-js`, `urlstate-js`, and Valibot
- Unprivileged Go REST API and a privileged Go Agent connected only through a
  protected Unix socket
- OpenAPI-generated Go and TypeScript contracts
- SQLite metadata, forward-only migrations, and typed `sqlc` queries
- Installer-generated temporary administrator credentials, mandatory first-login
  password change, server-side sessions, CSRF protection, and audit events
- Linux hosting account create, enable, disable, and recoverable delete
- Websites and hostname bindings with isolated Nginx and PHP-FPM 8.3 configuration
- Account Packages with enforced limits and explicit service defaults
- Let's Encrypt HTTP-01 certificates for hosted domains and the panel, DNS
  preflight, HTTPS redirects, expiry state, and renewal jobs
- MySQL database, user, and same-account grant lifecycle with one-time generated
  credentials
- Authoritative DNS zones and records through a local PowerDNS Agent driver
- Scheduled Tasks through validated per-account crontab files
- Scheduled and on-demand local backups with retention, SHA-256 verification,
  restore preview, and selective file/database/metadata restore

This is a pre-release project. Steps 1–7 have passed acceptance on the test VPS;
Step 8 is stabilizing the first public schema and release lifecycle. Clean
installation, uninstall, and recovery acceptance remain release gates. See
[plan.md](plan.md) for the current status; do not treat development candidates
as production releases.

## Requirements

- Go 1.27 or newer
- Node.js 24 or newer
- npm 10 or newer
- Docker with Buildx for Linux release creation

Production releases contain compiled Go binaries, the Next.js standalone build,
and its Linux Node.js runtime, so these build tools are not required on the
managed host.

## Development

Install dependencies and regenerate the API clients:

```sh
make setup
make generate
```

Run the Agent, API, and frontend in separate terminals:

```sh
make dev-init
make dev-agent
make dev-server
make dev-web
```

`make dev-init` creates the local `admin` user once and prints its temporary
password. Complete the profile and replace that password on the first login.

The first public schema uses `0001_initial.sql`. Databases created by the old
development migration chain are rejected, not converted or deleted. Preserve
any existing development database and use a separate fresh database path for
this baseline; never rename migration records to bypass the check.

The Next.js development server proxies `/api` to `127.0.0.1:8080`. Host
mutations require the expected Ubuntu services and root Agent permissions;
ordinary UI development and automated tests do not.

Run the complete repository check:

```sh
make check
```

## Production installation

Production installation consumes a release containing Linux amd64 binaries,
the standalone Next.js frontend, and its Node.js runtime. On Ubuntu 24.04, run:

```sh
sudo ./packaging/ubuntu/install.sh
```

Maintainers build a versioned release and its SHA-256 checksum from a clean
worktree with:

```sh
make release VERSION=0.1.0
```

The result is written to `dist/webycp-0.1.0-linux-amd64.tar.gz`. The build uses
the locked frontend dependencies, static Go binaries, a Linux Next.js runtime,
normalized archive
metadata, the current commit timestamp, and the current commit identifier.

The installer prints the initial `admin` username and a generated temporary
password. The panel is then available on `https://SERVER_IP:8443` with a
temporary self-signed certificate and requires a new password on first login.
See the [Ubuntu packaging guide](packaging/README.md) for installation, atomic
upgrades, recovery, installed paths, permissions, and the bootstrap TLS flow.
Use the [operations runbook](docs/operations.md) for credential, backup,
restore, certificate, Agent, and full-host recovery procedures.

## REST resources

- Authentication and administrator profile: `/api/v1/auth/*`
- Nodes, jobs, and audit: `/api/v1/nodes`, `/api/v1/jobs`, `/api/v1/audit-events`
- Hosting accounts and Packages: `/api/v1/accounts`, `/api/v1/packages`
- Websites and hostnames: `/api/v1/websites`, `/api/v1/website-domains`
- Certificates: `/api/v1/certificates`,
  `/api/v1/websites/{websiteId}/certificate`
- MySQL: `/api/v1/databases`, `/api/v1/database-users`,
  `/api/v1/database-grants`
- Scheduled Tasks: `/api/v1/scheduled-tasks`
- Authoritative DNS: `/api/v1/dns/*`
- Service defaults: `/api/v1/service-settings`
- Backup plans, runs, artifacts, previews, and restores: `/api/v1/backup-*`
- Health: `/api/v1/health`

The Agent exposes typed operations under `/agent/v1` over its Unix socket. It
does not accept raw shell commands or listen on a public TCP port.

## Backup credential boundary

Database contents and WEBYCP database definitions are backed up. MySQL user
passwords are intentionally never retained after provisioning, so they cannot
be recovered from an artifact; database users must receive new credentials
after a disaster restore.

## Security

Never commit server credentials, private keys, generated certificates, backup
artifacts, or production environment files. The public API must not run as
root. Rotate any credential that has been shared through chat before external
host testing.

Run the security baseline locally with:

```sh
make security
```

It checks reachable Go vulnerabilities, production frontend dependencies, the
complete Git history, and the current working tree. Release archives receive a
separate secret scan before their checksum is published. Scanner versions are
pinned in `scripts/security.sh`. If the npm audit endpoint is unavailable, a
pinned OSV Scanner image checks the lockfile instead; if neither scan can run,
the check fails closed.

## License

Copyright 2026 Gualter Fernandes.

Licensed under the [Apache License 2.0](LICENSE).

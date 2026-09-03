# WEBYCP

WEBYCP is a self-hosted, open-source hosting control panel for Ubuntu 24.04.

The v1 feature implementation through local backup and restore is now present:

- React/Vite SPA with HeroUI, Tailwind, Lucide, SWR, `reqly-js`, and
  `urlstate-js`
- Unprivileged Go REST API and a privileged Go Agent connected only through a
  protected Unix socket
- OpenAPI-generated Go and TypeScript contracts
- SQLite metadata, forward-only migrations, and typed `sqlc` queries
- First-admin bootstrap, server-side sessions, CSRF protection, persistent
  jobs, and audit events
- Linux hosting account create, enable, disable, and recoverable delete
- Domain and alias lifecycle with isolated Nginx and PHP-FPM 8.3 configuration
- Let's Encrypt HTTP-01 certificates for hosted domains and the panel, DNS
  preflight, HTTPS redirects, expiry state, and renewal jobs
- MySQL database, user, and same-account grant lifecycle with one-time generated
  credentials
- Validated per-account cron files
- Scheduled and on-demand local backups with retention, SHA-256 verification,
  restore preview, and selective file/database/metadata restore

Native Ubuntu packaging, hardening, and real-host acceptance are deliberately
left for M8 and M9. See [plan.md](plan.md) for the complete scope and milestone
status.

## Requirements

- Go 1.27 or newer
- Node.js 22 or newer
- npm 10 or newer

Production releases will contain compiled Go binaries and static frontend
assets, so these build tools will not be required on the managed host.

## Development

Install dependencies and regenerate the API clients:

```sh
make setup
make generate
```

Run the Agent, API, and frontend in separate terminals:

```sh
make dev-agent
make dev-server
make dev-web
```

The frontend development server proxies `/api` to `127.0.0.1:8080`. Host
mutations require the expected Ubuntu services and root Agent permissions;
ordinary UI development and automated tests do not.

Run the complete repository check:

```sh
make check
```

## REST resources

- Bootstrap and authentication: `/api/v1/bootstrap`, `/api/v1/auth/*`
- Nodes and jobs: `/api/v1/nodes`, `/api/v1/jobs`
- Hosting accounts: `/api/v1/accounts`
- Domains and aliases: `/api/v1/domains`, `/api/v1/domains/{domainId}/aliases`
- Certificates: `/api/v1/certificates`,
  `/api/v1/domains/{domainId}/certificate`
- MySQL: `/api/v1/databases`, `/api/v1/database-users`,
  `/api/v1/database-grants`
- Cron: `/api/v1/cron-jobs`
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

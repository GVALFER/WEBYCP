# WEBYCP Implementation Plan

## 1. Purpose

WEBYCP is an open-source hosting control panel. Version 1 will manage one
Ubuntu 24.04 LTS server while preserving clear upgrade paths for additional
nodes, operating systems, web servers, database engines, PHP versions, and
backup destinations.

The first release must remain small enough to build and operate safely. Its
architecture must support growth through stable contracts and isolated
implementations, not through premature microservices or speculative features.

## 2. Product Principles

1. **Safe by default**
    - The public API never runs as root.
    - The privileged agent exposes a small, typed operation surface.
    - User input never becomes an arbitrary shell command.
    - Configurations are validated before activation and written atomically.

2. **Simple first deployment**
    - One server, two Go processes, one React build, and one SQLite database.
    - Native Ubuntu packages are preferred over external repositories.
    - systemd owns process lifecycle and journald owns process logs.

3. **Scalable contracts**
    - Every managed resource belongs to a node, even when v1 has one local node.
    - Driver interfaces describe capabilities rather than specific products.
    - The local agent protocol is versioned and can later run remotely over mTLS.
    - Core services do not depend directly on SQLite or a concrete driver.

4. **No unnecessary duplication**
    - OpenAPI is the source of truth for HTTP contracts and generated types.
    - Shared behavior is extracted when it is genuinely reusable or
      security-critical.
    - Feature-specific helpers remain with their feature.
    - Generic dumping-ground packages are avoided.

5. **Observable operations**
    - System changes run as persistent jobs.
    - Jobs expose steps, timestamps, outcomes, and safe logs.
    - Administrative actions are recorded in an audit log.

## 3. Confirmed Technology Baseline

### Frontend

- React with TypeScript
- Vite for development and production builds
- React Router for application routing
- HeroUI for UI components
- Tailwind CSS for styling
- Lucide for icons
- SWR for server-state cache, revalidation, mutations, and job polling
- `reqly-js` as the HTTP transport
- `urlstate-js` for filters, search, pagination, tabs, and other URL state

Vite is not a production server. The production build is served by the Go API
service so the UI and REST API share the same origin.

### Backend

- Go for both the REST API and privileged agent
- Standard `net/http` where practical
- REST/JSON APIs described with OpenAPI
- SQLite for panel metadata
- `sqlc` for typed database access
- Versioned SQL migrations
- Structured logging through Go `slog`

### Managed Host Baseline

- Ubuntu 24.04 LTS
- amd64 initially
- Nginx from the Ubuntu repositories
- PHP-FPM 8.3 from the Ubuntu repositories
- MySQL 8.0 from the Ubuntu repositories
- system cron/crontab integration
- Local backup storage
- ACME certificates, initially through Let's Encrypt

Exact patch versions are owned by Ubuntu security updates and must not be
hard-coded into domain logic.

## 4. High-Level Architecture

```text
Browser
   |
   | HTTPS
   v
webycp-server                         unprivileged
   |-- React static assets
   |-- REST API /api/v1
   |-- authentication and authorization
   |-- SQLite metadata
   |-- job queue and workers
   |-- audit log
   |
   | HTTP/JSON over a protected Unix socket
   v
webycp-agent                          root
   |-- account operations
   |-- Nginx driver
   |-- PHP-FPM driver
   |-- MySQL driver
   |-- cron driver
   |-- certificate driver
   `-- backup engine and storage driver
```

### Production Request Path

```text
HTTPS client
   -> dedicated panel listener/reverse proxy
   -> webycp-server on loopback
      -> /api/v1/* for JSON
      -> /* for the React application
```

The panel Nginx configuration must be isolated from generated customer site
configuration. A failed customer configuration must never replace the last
known valid Nginx configuration.

## 5. Process Responsibilities

### `webycp-server`

Runs under an unprivileged `webycp` system account and owns:

- Static frontend delivery
- Authentication, sessions, and authorization
- REST input validation and response serialization
- Panel metadata and database migrations
- Desired resource state
- Persistent jobs and retry policy
- Audit records
- Calls to the local Agent
- Health and readiness endpoints

It must not modify system users, `/etc`, Nginx, PHP-FPM, MySQL, crontabs, or
backup directories directly.

### `webycp-agent`

Runs as root and owns only host-level execution:

- System user and directory management
- Atomic configuration writes
- Ownership and permission changes
- Service validation and reloads
- MySQL administrative operations
- Cron installation and removal
- Certificate issue, install, renew, inspect, and revoke operations
- Backup creation and restore operations

The Agent listens only on `/run/webycp/agent.sock` in v1. Socket filesystem
permissions authenticate the API service. The protocol is explicitly versioned
so a future controller can use the same semantics over TCP with mTLS.

The Agent must never accept a raw command string from the REST API.

## 6. Repository Structure

```text
webycp/
|-- api/
|   |-- public.openapi.yaml
|   `-- agent.openapi.yaml
|
|-- web/
|   |-- src/
|   |   |-- app/
|   |   |-- components/
|   |   |-- features/
|   |   |   |-- accounts/
|   |   |   |-- domains/
|   |   |   |-- databases/
|   |   |   |-- cron/
|   |   |   |-- certificates/
|   |   |   |-- backups/
|   |   |   `-- jobs/
|   |   |-- hooks/
|   |   |-- lib/
|   |   |   |-- api.ts
|   |   |   `-- swr.ts
|   |   |-- routes/
|   |   `-- utils/
|   |       |-- classnames.ts
|   |       |-- format.ts
|   |       `-- validation.ts
|   |-- package.json
|   `-- vite.config.ts
|
|-- cmd/
|   |-- webycp-server/
|   |   `-- main.go
|   `-- webycp-agent/
|       `-- main.go
|
|-- internal/
|   |-- accounts/
|   |-- audit/
|   |-- auth/
|   |-- backups/
|   |-- certificates/
|   |-- cron/
|   |-- databases/
|   |-- domains/
|   |-- jobs/
|   |-- nodes/
|   |-- agent/
|   |   |-- client/
|   |   |-- protocol/
|   |   `-- server/
|   |-- drivers/
|   |   |-- backup/local/
|   |   |-- certificate/acme/
|   |   |-- cron/crontab/
|   |   |-- database/mysql/
|   |   |-- runtime/phpfpm/
|   |   `-- web/nginx/
|   |-- execx/
|   |-- fsx/
|   |-- httpapi/
|   |-- store/sqlite/
|   `-- validate/
|
|-- migrations/
|-- packaging/
|   |-- systemd/
|   `-- ubuntu/
|-- scripts/
|-- tests/
|   |-- contract/
|   `-- integration/
|-- go.mod
|-- Makefile
|-- plan.md
`-- README.md
```

### Reuse Rules

- `web/src/utils` contains pure, framework-independent frontend helpers.
- `web/src/lib` contains configured integrations such as the API client and SWR
  configuration.
- Feature-specific formatters, schemas, and hooks stay inside the feature.
- Go does not use a generic `utils` package.
- Reusable Go behavior uses focused packages such as:
    - `fsx` for atomic files, permissions, and ownership
    - `execx` for safe command invocation and bounded output capture
    - `validate` for shared names, domains, paths, and resource limits
- A helper is extracted when it has multiple real consumers or when a single
  implementation is necessary for correctness or security.

## 7. Domain Model

The initial schema should contain explicit tables rather than a generic
entity/value model.

### Identity and tenancy

- `users`: people who authenticate to the panel
- `sessions`: revocable server-side sessions
- `accounts`: hosting isolation and ownership boundary
- `account_members`: future-compatible user/account membership and role mapping

Panel users and Linux hosting accounts are separate concepts. A hosting account
maps to one managed Linux user in v1.

### Infrastructure

- `nodes`: managed hosts; contains one local node in v1
- `domains`: primary hosted domains
- `domain_aliases`: aliases attached to a primary domain
- `databases`: managed customer databases
- `database_users`: managed MySQL users
- `database_grants`: explicit access relationships
- `cron_jobs`: desired scheduled commands and status
- `certificates`: ACME certificate lifecycle and expiry state
- `certificate_names`: certificate SAN names
- `backup_plans`: schedule, retention, and scope
- `backup_runs`: execution state
- `backup_artifacts`: stored backup metadata and checksums
- `jobs`: durable asynchronous operations
- `job_steps`: safe step-level output and timing
- `audit_events`: actor, action, resource, result, and timestamp

Every managed resource includes a globally unique ID, `node_id`, timestamps,
desired status, observed status, and a safe last-error field where applicable.
IDs and timestamps must not depend on one SQLite instance so they remain valid
after a future PostgreSQL or multi-node migration.

## 8. Driver Model

Interfaces are defined close to the core service that consumes them. Concrete
host implementations live under focused `internal/agent` capability packages
and are registered at build time. Go runtime plugins are out of scope.

### Initial capabilities

| Capability     | v1 implementation  | Future implementations              |
| -------------- | ------------------ | ----------------------------------- |
| Web server     | Nginx              | Apache, OpenLiteSpeed               |
| Database       | MySQL              | MariaDB, PostgreSQL                 |
| Runtime        | PHP-FPM 8.3        | Multiple PHP versions               |
| Scheduler      | crontab            | systemd timers                      |
| Certificate    | ACME/Let's Encrypt | Other ACME CAs, manual certificates |
| Backup storage | Local filesystem   | S3-compatible, Cloudflare R2        |

### Driver requirements

Every mutating driver operation must be:

- Idempotent
- Context-aware and cancellable where safe
- Validated before execution
- Explicit about capabilities
- Safe to retry or explicitly marked non-retryable
- Observable through structured results
- Covered by unit or golden-file tests

The code must not introduce an abstraction for a second implementation before a
real boundary exists. The v1 interface should cover only operations required by
the v1 use cases.

## 9. REST and Agent Contracts

### Public API

- Base path: `/api/v1`
- JSON request and response bodies
- Consistent error envelope with stable machine-readable codes
- Pagination and filtering encoded in query parameters
- `202 Accepted` plus a `jobId` for host-changing operations
- Request IDs on every response
- Idempotency keys for retry-sensitive creation operations

Example:

```text
POST /api/v1/domains
  -> validate and authorize
  -> persist desired domain and job in one SQLite transaction
  -> return 202 with jobId
  -> worker dispatches typed operation to the Agent
  -> observed state and job outcome are persisted
```

### Agent API

- Separate OpenAPI contract
- HTTP/JSON over Unix socket in v1
- No public listener
- No browser authentication concepts
- No raw shell or unrestricted filesystem endpoints
- Version/capability handshake
- Idempotency key on every mutating operation
- Bounded and redacted output

Generated Go and TypeScript types should prevent manual DTO duplication.

## 10. Persistent Jobs and Reconciliation

Host mutations cannot be part of the same atomic transaction as SQLite. The
system therefore uses desired and observed state:

1. Validate and authorize the requested change.
2. Persist desired state and a queued job in one database transaction.
3. Lock the affected logical resource.
4. Execute a typed Agent operation.
5. Validate generated service configuration.
6. Activate changes atomically.
7. Persist observed state, job steps, and audit outcome.
8. On failure, keep the previous known-good host configuration and expose the
   error without claiming the resource is healthy.

On restart, interrupted jobs are recovered and reconciled according to their
idempotency and retry policy. Only independent resources may run concurrently.

SWR polls active jobs in v1. Server-Sent Events may be added later without
changing the job model.

## 11. SSL/TLS Scope for v1

SSL/TLS is a complete v1 requirement for both hosted domains and the panel.

### Hosted domains

- ACME through Let's Encrypt
- HTTP-01 challenge
- DNS preflight before requesting a certificate
- Primary domain and eligible aliases included as SANs
- HTTP to HTTPS redirect enabled by default
- Certificate status and expiry visible in the UI
- Automatic renewal jobs
- Atomic certificate and key installation
- `nginx -t` before every Nginx reload
- Safe fallback to the last valid configuration

Aliases that do not resolve to the managed node must not block certificates for
otherwise valid names without a clear user-visible explanation.

### Panel certificate

- Installation starts with an explicitly identified bootstrap state.
- Once a panel hostname resolves correctly, an ACME certificate is issued.
- Panel and customer certificate configuration remain isolated.
- Certificate renewal must not require a full panel reinstall.

### Deferred certificate features

- Wildcard certificates and DNS-01
- User-provided custom certificates
- Additional ACME providers
- External secret managers

ACME protocol and cryptography must use a maintained implementation. WEBYCP
must not implement the protocol or certificate primitives from scratch.

## 12. V1 Functional Scope

### Included

- First-admin bootstrap
- Session authentication and basic admin/user authorization
- Hosting accounts backed by isolated Linux users
- Domains, aliases, document roots, and enable/disable state
- Nginx site configuration
- Per-account PHP-FPM pools using PHP 8.3
- ACME certificates and renewal
- MySQL databases, users, passwords, and grants
- Cron jobs executed as the hosting account
- Local scheduled and on-demand backups
- Restore of files, databases, and WEBYCP resource metadata
- Retention and artifact checksums
- Persistent jobs and visible job history
- Audit log
- Service and node health
- Native systemd units and Ubuntu installer
- Safe upgrade and database migration path

### Explicitly excluded

- Email hosting
- Authoritative DNS hosting
- File manager and browser terminal
- FTP server
- Firewall management
- Reseller plans, billing, and subscriptions
- Containers or per-site virtual machines
- Multiple managed nodes in production
- High-availability control plane
- Apache and LiteSpeed implementations
- Multiple installed PHP versions
- Remote backup storage
- Wildcard certificate automation
- AI-controlled infrastructure operations

## 13. Filesystem and Service Layout

Initial target paths:

```text
/etc/webycp/                         application configuration
/var/lib/webycp/server/              SQLite and unprivileged durable state
/var/lib/webycp/account-trash/       root-controlled deleted account data
/var/lib/webycp/acme/                root-controlled HTTP-01 challenge data
/etc/letsencrypt/live/webycp-*/      root-controlled certificate material
/var/backups/webycp/                 local backup artifacts
/run/webycp/agent.sock               privileged Agent socket
/home/wcp_<account-id-prefix>/       hosting account home and site data
/home/wcp_<account-id-prefix>/.webycp-trash/ recoverable deleted site data
/etc/nginx/webycp/                   WEBYCP-owned Nginx includes
/etc/php/8.3/fpm/pool.d/webycp-*.conf
```

Exact paths must be verified against a clean Ubuntu 24.04 installation before
they become a compatibility contract.

Systemd units:

- `webycp-server.service`
- `webycp-agent.service`

The installer must never overwrite unrelated user-managed service
configuration. WEBYCP edits only files and include directories it owns.

## 14. Security Baseline

- API service runs without root privileges.
- Agent socket is accessible only to the API service group.
- Session cookies are Secure, HttpOnly, and SameSite protected.
- State-changing browser requests include CSRF protection.
- Passwords use a modern password-hashing function with calibrated parameters.
- Login attempts are rate limited and audited.
- Sensitive values are redacted from logs, jobs, API errors, and audit metadata.
- Private keys and host credentials use strict filesystem permissions.
- MySQL administration prefers local socket authentication.
- Commands use executable plus argument arrays, never interpolated shell text.
- Domain names, usernames, paths, cron commands, and resource limits receive
  dedicated validation.
- Generated file paths must be contained within approved roots.
- Destructive operations require exact resource identity and ownership checks.
- Backups and restores verify checksums and reject path traversal.
- Dependency and security scanning run in CI.

MFA is desirable for v1 hardening, but must not delay the safe delivery of the
core resource lifecycle unless promoted to a release requirement.

## 15. Scalability Strategy

### V1: single node

- One API instance
- One local Agent
- One SQLite database in WAL mode
- Local persistent job queue
- Local backup destination

### Future: multiple managed nodes

- Preserve `node_id` on all resources from the first migration.
- Replace Unix-socket transport with the same versioned Agent operations over
  mTLS.
- Add Agent enrollment, certificate rotation, heartbeats, and capability
  discovery.
- Route jobs to node-specific workers.
- Keep agents deterministic and able to report observed state.

### Future: highly available control plane

- Add a PostgreSQL store implementation behind the existing storage boundary.
- Move job claiming to PostgreSQL-safe leases.
- Store artifacts in S3-compatible storage.
- Run multiple stateless API replicas.
- Keep sessions and idempotency records in shared storage.

These future paths must influence identifiers, resource ownership, and protocol
versioning now. They must not cause PostgreSQL, message brokers, Kubernetes, or
remote Agents to be introduced in v1.

## 16. Testing Strategy

### Local and CI tests

- Go unit tests for core rules and validation
- Driver tests with fake process and filesystem boundaries
- Golden-file tests for Nginx and PHP-FPM configuration
- SQLite migration tests from an empty database and previous schema versions
- REST and Agent OpenAPI contract tests
- Frontend unit tests for important utilities and feature behavior
- Frontend production build verification
- Race detection for relevant Go packages
- Static analysis, formatting, and dependency checks

### Integration tests

- Disposable Ubuntu 24.04 environment
- Real systemd, Nginx, PHP-FPM, MySQL, and cron
- ACME staging environment before production certificate tests
- Account, domain, database, cron, certificate, backup, restore, and deletion
  lifecycle tests
- Invalid Nginx configuration and rollback tests
- Agent/API restart and interrupted-job recovery tests
- Repeated idempotent operation tests

Containers may test isolated components, but they do not replace a VM for final
host-management tests because systemd and host filesystem behavior are part of
the product.

### External VPS testing policy

A clean Ubuntu 24.04 VPS supplied by the project owner may be used during the
integration milestone only after:

1. Access is explicitly confirmed for that test run.
2. Any password shared through chat is rotated.
3. A dedicated temporary SSH key is installed.
4. The OS version and machine identity are verified read-only.
5. A provider snapshot or other recovery method is confirmed where available.
6. The exact destructive test scope is stated before execution.

No VPS IP address, password, private key, token, or provider credential may be
stored in this repository, generated artifacts, logs, or the napkin.

## 17. Delivery Milestones

### M0 — Architecture and project decisions

- [x] Agree on React/Vite, Go API, Go Agent, SQLite, and Ubuntu 24.04.
- [x] Agree on SWR, `reqly-js`, and `urlstate-js`.
- [x] Include full SSL/TLS lifecycle in v1.
- [x] Define the initial repository and process boundaries.
- [x] Choose the public Go module/repository path (`github.com/GVALFER/WEBYCP`).
- [x] Choose the open-source license (Apache-2.0).
- [x] Choose npm as the JavaScript package manager.

Verification: the architecture, repository path, package manager, and license
decisions are recorded before public packaging.

### M1 — Scaffold and developer workflow

- [x] Create the Go module and both commands.
- [x] Create the React/Vite application.
- [x] Configure HeroUI, Tailwind, Lucide, SWR, `reqly-js`, and `urlstate-js`.
- [x] Add OpenAPI contracts and code-generation commands.
- [x] Add lint, Go format, type-check, test, and build commands.
- [x] Add CI for frontend and Go.

Verification: one command validates the repository, both Go binaries build,
the frontend production build succeeds, and generated contracts are current.

### M2 — Core control-plane foundation

- [x] Add configuration loading and structured logging.
- [x] Add SQLite migrations and `sqlc` queries.
- [x] Add users, sessions, accounts, nodes, jobs, job steps, and audit events.
- [x] Implement first-admin bootstrap and login/logout.
- [x] Implement the Agent Unix socket, protocol version, and health handshake.
- [x] Add the first persistent worker and interrupted-job recovery.

Verification: an authenticated admin can queue a node probe, observe its job
states, and communicate with an Agent without granting root to the API. Account
provisioning starts in M3 together with its Linux lifecycle.

### M3 — Accounts, domains, Nginx, and PHP

- [x] Create hosting accounts through an atomic durable job.
- [x] Reconcile Linux hosting users idempotently through the Agent.
- [x] Create a symlink-safe owned account directory layout.
- [x] Add account disable/delete lifecycle.
- [x] Create and list primary domains through atomic durable jobs.
- [x] Create symlink-safe document roots owned by the hosting account.
- [x] Create and list aliases through atomic durable jobs.
- [x] Add alias enable, disable, and delete lifecycle.
- [x] Add alias hostname update lifecycle.
- [x] Generate isolated Nginx HTTP sites.
- [x] Generate per-account PHP-FPM 8.3 pools.
- [x] Connect Nginx sites to isolated PHP-FPM Unix sockets.
- [x] Validate Nginx configuration and roll back before recovery reload.
- [x] Add domain enable, disable, and recoverable delete reconciliation.
- [x] Add primary domain hostname update reconciliation.

Verification: a domain serves a PHP test application, repeated reconciliation
is idempotent, and an invalid update leaves the previous site online.

### M4 — SSL/TLS

- [x] Implement ACME account management.
- [x] Add DNS preflight and HTTP-01 challenge routing.
- [x] Issue certificates for domains and eligible aliases.
- [x] Add panel certificate bootstrap and issuance.
- [x] Add redirect policy and renewal jobs.
- [x] Expose expiry and errors in the API and UI.

Verification: staging certificates issue and renew automatically; production
issuance is then tested once; failed issuance does not break HTTP service or the
panel.

### M5 — MySQL

- [x] Add database, database user, and grant lifecycle.
- [x] Generate strong credentials and reveal them safely.
- [x] Prevent cross-account grants.
- [x] Add idempotent delete and reconciliation behavior.

Verification: an account can connect only to its granted databases and no
credential is exposed in logs or job output.

### M6 — Cron

- [x] Add cron validation and CRUD.
- [x] Install crontabs under the hosting Linux user.
- [x] Add enable/disable behavior and safe output policy.

Verification: scheduled commands run with the intended UID, environment, and
working directory and cannot write to another account's data.

### M7 — Local backup and restore

- [x] Define versioned backup manifest format.
- [x] Back up selected files, databases, and WEBYCP metadata.
- [x] Add local storage, retention, checksums, and status.
- [x] Implement restore preview and execution.
- [x] Test partial and complete restores.

Verification: a deleted test site and database can be restored from a verified
artifact and served successfully afterward.

### M8 — Hardening and native packaging

- [ ] Add systemd sandboxing appropriate to each process.
- [x] Add production configuration and filesystem permissions.
- [x] Add reproducible Linux amd64 release archives and SHA-256 checksums.
- [ ] Add install, upgrade, migration, and recovery commands.
- [ ] Add dependency, security, and secret scanning.
- [ ] Document backup, restore, certificate, and Agent recovery procedures.

Verification: installation on a clean Ubuntu 24.04 VM is repeatable, upgrades
preserve state, and removal does not delete customer data without an explicit
destructive option.

### M9 — External VPS validation and v1 release candidate

- [ ] Rotate chat-shared credentials and install a temporary SSH key.
- [ ] Perform read-only host preflight.
- [ ] Confirm snapshot/recovery options.
- [ ] Install the release candidate.
- [ ] Run the complete lifecycle test suite.
- [ ] Review logs, permissions, recovery behavior, and resource cleanup.
- [ ] Remove temporary access after testing.

Verification: all v1 acceptance criteria pass on a fresh Ubuntu 24.04 host and
the host can be recovered from every intentionally tested failure.

## 18. V1 Definition of Done

Version 1 is complete only when:

- A clean Ubuntu 24.04 server can install WEBYCP reproducibly.
- The REST API and Agent run under distinct privilege boundaries.
- An administrator can manage hosting accounts, domains, aliases, PHP sites,
  MySQL databases, cron jobs, certificates, and local backups.
- A complete restore has been demonstrated.
- Certificate issuance and renewal have been demonstrated.
- All mutations are represented by durable jobs and audit events.
- Invalid generated configuration cannot replace known-good configuration.
- Repeated operations are idempotent.
- No secrets appear in the repository, logs, job output, or API errors.
- Automated tests and the external Ubuntu 24.04 lifecycle suite pass.
- Installation, upgrade, backup, restore, and recovery are documented.

The M3–M7 checkboxes record completed implementation and local automated
coverage. Their real Ubuntu, MySQL, cron, and ACME acceptance checks remain part
of M9 and must not be treated as complete yet.

## 19. Immediate Next Step

M8 is in progress. Add the upgrade, migration, and recovery workflow, then
validate the service sandbox on a disposable Ubuntu 24.04 environment. External
VPS validation follows only in M9 and only under the access policy above.

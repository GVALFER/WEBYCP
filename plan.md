# WEBYCP Product and Implementation Plan

## 1. Product Direction

WEBYCP is an open-source, self-hosted hosting control panel for Ubuntu 24.04
LTS. It starts as a single-server panel, but its contracts must allow more
servers, service implementations, runtime versions, database engines, DNS
providers, and backup destinations later.

The product must remain understandable while it grows. Extensibility comes
from explicit domain boundaries and small driver interfaces, not speculative
abstractions, generic entity stores, or premature microservices.

### Working rules

1. Implement one reviewed step at a time.
2. Keep the smallest complete implementation for the current requirement.
3. Prefer explicit state and typed contracts over implicit defaults.
4. Reuse code only when behavior is genuinely shared.
5. Keep feature-specific code beside its feature.
6. Do not add compatibility fallbacks, dual reads, dual writes, deprecated
   routes, or temporary adapters before the first public release.
7. Replace obsolete development structures cleanly after taking a snapshot;
   do not preserve test data at the cost of permanent complexity.
8. Stop after every roadmap step for review before expanding the scope.

## 2. Confirmed Stack

### Frontend

- Next.js App Router, React, and TypeScript
- Server Components for authentication and initial route data
- Client Components only where interaction is required
- HeroUI for components, modals, confirmation dialogs, toasts, and spinners
- Tailwind CSS for styling and design tokens
- Lucide for icons
- SWR for client-side server-state synchronization
- `reqly-js` for HTTP requests
- `urlstate-js` for filters, search, tabs, sorting, and pagination in the URL
- React Hook Form with `valibotResolver` for forms
- Day.js for display formatting in the administrator's IANA timezone
- No Prettier; frontend formatting remains owned by the editor

### Control plane

- Go REST API running without root privileges
- REST/JSON contracts described by OpenAPI
- Generated Go and TypeScript contract types
- SQLite for control-plane metadata
- `sqlc` for typed database queries
- Forward-only SQL migrations during development
- Structured logging with `slog`

### Host execution

- Privileged Go Agent
- Versioned typed protocol over a protected Unix socket
- Ubuntu 24.04 LTS, amd64 initially
- systemd for process lifecycle
- journald for service logs

### Initial service implementations

| Capability | Initial implementation | Future implementations |
| --- | --- | --- |
| Web server | Nginx | Apache, OpenLiteSpeed |
| Runtime | PHP-FPM 8.3 | Multiple PHP versions, Node.js, Python, Go |
| Database | MySQL 8.0 | MariaDB, PostgreSQL |
| Scheduler | crontab | systemd timers, distributed scheduler |
| Backup storage | Local filesystem | S3-compatible, R2, remote SSH |
| Certificates | Let's Encrypt HTTP-01 | Other ACME CAs, DNS-01 |
| DNS | PowerDNS Authoritative Server | Cloudflare, other providers |
| File transfer | Pure-FTPd with explicit FTPS | Separate SFTP driver |
| Firewall | firewalld with nftables | Other host firewall drivers |
| Service lifecycle | systemd | Other supported host managers |
| WAF | ModSecurity v3 with OWASP CRS and Nginx connector | Validated connectors for other web servers |

Drivers are Go interfaces compiled with the application in v1. Dynamic binary
plugins are not required. A second driver is only introduced when its feature
is implemented.

## 3. Runtime Architecture

```text
Browser
   |
   v
Nginx
   |-- Next.js standalone web service
   |      |-- SSR authentication
   |      |-- SSR route data
   |      `-- client application
   |
   `-- /api/v1 -> webycp-server
                        |
                        |-- authentication and authorization
                        |-- SQLite metadata
                        |-- jobs and audit events
                        `-- Unix socket -> webycp-agent (root)
                                                |
                                                |-- system accounts
                                                |-- service configuration
                                                |-- databases
                                                |-- certificates
                                                |-- scheduled tasks
                                                |-- backups
                                                `-- authoritative DNS
```

### Responsibility boundaries

`webycp-web` owns rendering and browser interaction. It never performs host
operations directly.

`webycp-server` owns authentication, authorization, desired state, validation,
metadata, jobs, and audit events. It runs unprivileged and never writes service
configuration or invokes arbitrary shell commands.

`webycp-agent` owns the smallest possible set of privileged host operations. It
accepts typed operations, validates every identifier and path, writes files
atomically, tests service configuration before activation, and rolls back
failed changes.

V1 uses one local server. Every managed resource still carries a `serverId` so
the model can later support remote Agents without rewriting ownership.

## 4. Frontend Architecture Rules

### Route data flow

Every protected route follows one sequence:

```text
page.tsx (Server Component)
   -> fetch initial typed data
   -> pass data as props
route client component
   -> display SSR data with fallbackData
   -> render shared components
```

`fallbackData` displays the SSR response without populating the SWR cache. Do
not seed the cache through a mount-time `mutate`. Initial mount skips duplicate
revalidation; changed keys revalidate normally, including a return to the SSR
page. Never fabricate missing business data.

Additional rules:

- Do not use `refreshInterval` or background polling by default.
- Revalidate on focus where appropriate.
- Mutations call `mutate` only for affected keys after success.
- A `401` from an expired protected request redirects to login through the
  shared API status handler; authentication endpoints are excluded.
- The protected layout resolves the session on the server before HTML is sent.
- Backend timestamps remain UTC; the browser formats them with
  `useTimezone().dt`.

### Components and actions

- Route client components own presentation and explicit SWR hooks.
- POST, PUT, PATCH, and DELETE logic belongs in focused components under the
  adjacent `actions` directory.
- Action components own validation, pending state, API mutation, toast, and
  cache mutation.
- Forms use React Hook Form, `valibotResolver`, the shared `Form`, and shared
  field components.
- Create and edit forms open in HeroUI modals instead of permanent sidebars.
- Destructive actions use the shared HeroUI `Confirm` component.
- Pending buttons retain their label and show a HeroUI `Spinner`.
- Native `alert`, `confirm`, and `prompt` are forbidden.

### Tables and URL state

- Lists use the shared `Table` and `Paginate` components.
- `Table` receives the typed response and renders rows, empty state, loading
  state, and pagination internally.
- `useTable` owns URL state only; it does not fetch data.
- The route's SWR key changes when `urlstate-js` changes the URL query.
- Multiple tables use dotted namespaces such as `plans.page` and
  `archives.size`.
- APIs continue receiving ordinary `page` and `size` parameters.

### Reuse and project structure

- Pure reusable frontend functions live in `web/src/utils`.
- Configured integrations live in `web/src/lib`.
- OpenAPI-generated contracts live in `web/src/contracts`.
- Shared UI belongs in `web/src/components`.
- Route-only schemas, formatters, components, and actions stay in the route.
- Do not create a `features` directory or a generic frontend dumping ground.
- Go reuse remains in small purpose-named packages such as `fsx`, `execx`, and
  `validate`; do not create a generic `utils` package.

## 5. Design System and Navigation

The next implementation step is visual only. It must establish one coherent
design system before any domain refactor.

### Visual direction

- Modern, quiet control-panel interface with strong information hierarchy
- Light theme based on the referenced soft mint/off-white atmosphere
- Dark theme based on the referenced deep navy/teal atmosphere
- Neutral content surfaces with restrained accent color
- Consistent typography, spacing, radii, borders, shadows, and focus states
- Dense enough for infrastructure management without looking dated
- Accessible contrast and complete keyboard focus behavior
- Responsive sidebar and content without duplicating desktop/mobile markup

Exact color tokens are selected and reviewed in Step 1. Components consume
semantic tokens such as background, surface, elevated surface, border,
foreground, muted foreground, accent, success, warning, and danger. Pages must
not introduce isolated hard-coded palettes.

### Target navigation

```text
Overview

Accounts
|-- Accounts
|-- Packages
`-- Templates                         later

Websites
|-- Websites
|-- Domains
|-- Aliases
`-- Certificates

DNS
|-- Zones
|-- Providers
`-- Nameservers

Databases
|-- Databases
|-- Users
`-- Permissions

File Access
|-- FTP Accounts
|-- File Manager                      later
`-- SFTP Accounts                     later

Backups
|-- Plans
|-- Archives
|-- Restore
`-- Destinations

Scheduled Tasks
|-- Tasks
`-- Script Library                    later

Monitoring
|-- Metrics
|-- Logs
`-- Alerts                            later

Security
|-- Firewall
|-- WAF
`-- SSH                               later

Applications                          later
|-- Catalog
|-- Containers
`-- Compose

Mail                                  later
|-- Domains
|-- Mailboxes
|-- Aliases and Forwarders
`-- Queue

System
|-- Servers
|-- Services
|-- Jobs
|-- Updates                           later
|-- Audit Log
`-- Settings
```

The sidebar uses expandable groups and clear active states. Unimplemented
features remain hidden rather than disabled or backed by placeholder pages.
The panel's own certificate belongs under `System > Settings > Panel SSL`, not
under hosted website certificates.

## 6. Target Domain Model

### Identity and hosting ownership

- `PanelUser`: a person who authenticates to the control panel
- `Session`: a revocable server-side login session
- `Account`: a hosting isolation boundary mapped to a managed Linux identity
- `Package`: resource limits assigned to an Account
- `AccountMember`: future user-to-account role mapping

Panel users and hosting Accounts are separate concepts. Package limits always
belong to Accounts, never directly to panel users.

### Servers and capabilities

- `Server`: a host managed by an Agent; currently one local server
- `ServerCapability`: installed and available drivers, services, and versions
- `ServiceState`: desired and observed service state

The UI uses `Server`, not `Node`, to avoid confusion with Node.js. The internal
transport may continue using node terminology until its dedicated refactor.

### Websites and hostnames

`Website` is the hosted application/vhost aggregate:

```text
Website
|-- accountId
|-- serverId
|-- name
|-- kind: php initially
|-- documentRoot
|-- webDriver: nginx initially
|-- runtimeDriver: phpfpm initially
|-- runtimeVersion: 8.3 initially
`-- enabled
```

`WebsiteDomain` represents a hostname binding:

```text
WebsiteDomain
|-- websiteId
|-- hostname
|-- kind: primary | alias
`-- enabled
```

Primary domains and aliases share one normalized global hostname namespace.
The database and service layer enforce collision rules once rather than
duplicating domain and alias logic. Redirect bindings are added only when that
feature is implemented.

PHP version and webserver are explicit per Website. Server-level defaults may
preselect a form value, but persisted Websites never rely on an empty value to
mean “use default”.

When a second Website kind is implemented, its typed configuration is added at
that time. V1 does not create speculative nullable configuration tables.

### Hosted certificates

- `Certificate`: lifecycle, issuer, expiry, renewal state, and Website scope
- `CertificateName`: normalized SAN names covered by a certificate

Hosted certificates remain separate from the panel listener certificate.
Certificate jobs derive allowed names from the Website's bindings.

### DNS

- `DNSProvider`: non-secret driver registration and health
- `DNSZone`: an authoritative DNS zone owned by an Account
- `DNSRecord`: a typed record inside a zone

The local PowerDNS HTTP API is loopback-only. Its API key remains in a root-only
Agent file and never enters control-plane SQLite, public API responses, jobs,
audit metadata, or ordinary resource payloads.

Website bindings and DNS zones are independent resources. Creating a Website
does not silently create or take ownership of DNS. An explicit assisted flow
may link them later.

### Databases

- `Database`
- `DatabaseUser`
- `DatabaseGrant`

They remain separate resources even if the common create flow can provision a
database, user, and grant in one modal.

### Scheduled tasks

The product resource is `ScheduledTask`; `crontab` is its first driver.

```text
ScheduledTask
|-- accountId
|-- serverId
|-- kind: command | http | backup | maintenance
|-- schedule
|-- target
|-- runAs
|-- enabled
`-- driver: crontab initially
```

V1 exposes only task kinds that can be validated and safely scoped. The public
API never accepts unrestricted root commands. Backup plans may reuse the
scheduler internally while remaining under the Backups section in the UI.

### Backups and operations

- `BackupPlan`: scope, schedule, retention, and destination
- `BackupRun`: one execution and its result
- `BackupArtifact`: stored archive metadata and checksum
- `Job`: durable asynchronous control-plane operation
- `JobStep`: observable operation phase
- `AuditEvent`: requested action and final system outcome

## 7. API, Jobs, and Security Rules

- Public endpoints are resource-oriented REST, not `action=` RPC endpoints.
- Read operations use GET and never expose generic table access.
- OpenAPI is the source of truth for public and Agent contracts.
- Host mutations create durable jobs with explicit states.
- Retried Agent operations are idempotent or detect completed work safely.
- Authentication uses server-side sessions in `HttpOnly` cookies.
- Protected mutations require CSRF validation and authorization.
- Passwords and generated database credentials are never returned twice.
- Secrets are not written to logs, jobs, audit payloads, or repository files.
- Host paths are derived from validated resource IDs and trusted roots.
- Service files are written atomically, validated, activated, and rolled back
  on failure.
- All backend timestamps and schedules are stored and executed in UTC.
- The administrator's timezone affects display only.

## 8. Pre-release Schema Strategy

Nothing is in public production yet, so schema cleanup must favor the final
model instead of compatibility code.

- Snapshot the development VPS before destructive schema work.
- Use one canonical model after each refactor.
- Do not keep legacy endpoints or old/new database representations together.
- Do not add aliases solely to preserve unfinished frontend code.
- Rebuild disposable test data when that is cleaner than migration scaffolding.
- Preserve real test data only when explicitly requested.
- Squash development migrations into a clean baseline before the first public
  release.
- After the first public release, migrations become permanent and backwards
  compatibility follows a documented policy.

## 9. Version 1 Scope

### Included

- Installer-generated administrator credentials and forced password change
- Administrator profile and display timezone
- One local Ubuntu 24.04 Server and Agent
- Hosting Accounts and Packages
- PHP Websites with domains and aliases
- Nginx and PHP-FPM 8.3
- Hosted and panel Let's Encrypt certificates
- MySQL databases, users, and grants
- Scheduled Tasks using the crontab driver
- Local backup plans, archives, verification, and restore
- DNS zones, records, and one selected initial DNS driver
- Pure-FTPd virtual credentials with mandatory explicit FTPS
- Host firewall rules and service profiles through firewalld
- Allowlisted service status and lifecycle operations through systemd
- A separately validated WAF step using ModSecurity v3 and OWASP CRS
- Durable Jobs and Audit Log
- Native installation, upgrade, rollback, and recovery procedures

### Deferred

- Remote multi-server control
- Apache and OpenLiteSpeed
- Multiple PHP versions and non-PHP Website kinds
- MariaDB, PostgreSQL, MongoDB, and Redis management
- Mail hosting
- File Manager and SFTP server management
- Docker, application catalog, and extension marketplace
- Intrusion detection and malware scanning
- High-availability control plane

Deferred features influence boundaries and naming only. They do not justify
empty pages, placeholder tables, unused interfaces, or speculative code.

## 10. Implementation Roadmap

Each step ends with automated checks, targeted browser validation, and a review
pause. The next step does not begin without approval.

### Step 0 — Replace the plan

Deliverables:

- Replace the obsolete implementation plan with this product and roadmap plan.
- Record the agreed architecture, terminology, navigation, and clean-code rules.

Verification:

- Only documentation changes.
- No frontend, backend, schema, or VPS changes.

### Step 1 — Design system and application shell

Scope is presentation only; existing business behavior remains unchanged.

Deliverables:

- Define reviewed semantic color tokens for light and dark themes.
- Apply the referenced mint/off-white light atmosphere.
- Apply the referenced deep navy/teal dark atmosphere.
- Establish typography, spacing, borders, radii, shadows, and focus rings.
- Redesign login without loading flashes or form resets.
- Build the scalable expandable sidebar and responsive shell.
- Apply the final navigation grouping only to currently implemented pages.
- Align page headers, cards, tables, pagination, modals, toasts, buttons,
  loading states, and empty states.
- Fix existing table header/footer spacing and separator inconsistencies.

Verification:

- Light and dark themes checked on every existing route.
- Desktop and narrow responsive layouts checked with global Playwright MCP.
- Keyboard focus and modal behavior checked.
- No continuous network polling.
- Frontend lint, typecheck, tests, and production build pass.
- Stop for visual review.

### Step 2 — Website domain refactor

Deliverables:

- Replace the current hosted `Domain` aggregate with `Website`.
- Replace separate primary/alias persistence with normalized
  `WebsiteDomain` bindings.
- Introduce explicit Website kind, web driver, runtime driver, and version.
- Rename Go packages, SQL queries, OpenAPI resources, jobs, Agent operations,
  Next.js routes, and UI copy consistently.
- Move panel certificate controls to System Settings.
- Update Nginx, PHP-FPM, certificate, backup, and restore references.
- Remove obsolete domain endpoints and code in the same step.

Verification:

- Hostname collisions remain transactionally enforced.
- Website create, enable, disable, delete, domain, alias, SSL, backup, and
  restore lifecycle pass locally and on the VPS.
- No dual model or compatibility adapter remains.
- Stop for architecture and UI review.

### Step 3 — Accounts and Packages

Deliverables:

- Add Package CRUD and Account assignment.
- Define explicit v1 count limits for Websites, domains, aliases, databases,
  database users, Scheduled Tasks, and backup retention/storage where it can be
  truthfully enforced.
- Enforce limits in the control-plane transaction before a job is queued.
- Display current usage against each Package limit.
- Add the Accounts > Packages page and Package assignment actions.
- Keep Templates hidden until their behavior is defined.

Verification:

- Boundary and over-limit tests for every implemented limit.
- Concurrent creates cannot bypass a limit.
- Lowering a Package never deletes resources silently.
- Existing resources remain usable while new over-limit creates are blocked.
- Stop for review.

### Step 4 — System capabilities and service settings

Deliverables:

- Expose the Server's installed webserver, runtime, database, scheduler, and
  backup capabilities.
- Add System > Servers and System > Services views.
- Add explicit defaults used only to preselect creation forms.
- Keep persisted resource driver selections explicit.
- Keep only Nginx, PHP-FPM 8.3, MySQL, crontab, and local storage selectable
  until additional drivers really exist.

Verification:

- Observed capabilities match the VPS.
- Unsupported selections cannot enter API requests or persisted state.
- Agent/service health failures are visible without aggressive polling.
- Stop for review.

### Step 5 — DNS foundation

Selected implementation: PowerDNS Authoritative Server with its SQLite backend
and loopback HTTP API.

Status: implementation, release, real provider lifecycle, VPS upgrade, rollback,
and recovery verification are complete. During Step 6, global Playwright checked
Zones, Providers, Nameservers and the nameserver modal. The full record-editor
visual pass remains pending. The empty-zone warning needs a dark-theme contrast
adjustment; it was recorded without expanding the backup changes.

Deliverables:

- Add typed DNS provider interface and one implementation only.
- Add DNS zones and common record types.
- Validate zone ownership, names, record data, TTL, and duplicate rules.
- Store provider credentials outside ordinary resource payloads and logs.
- Add DNS > Zones, Providers, and Nameservers where supported.
- Keep DNS resources separate from Website bindings.

Verification:

- Zone and record lifecycle tested against the selected real provider.
- Provider failures do not corrupt desired state.
- No credential appears in logs, jobs, audit events, or API responses.
- Stop for review.

### Step 6 — Backups and destinations

Status: implemented and deployed to the Ubuntu 24.04 test VPS as `0.1.1-rc.18`.
Reviewed; Step 7 has also passed VPS acceptance. The local driver contract and observed
node capabilities are reused; no remote provider has been selected.

Acceptance evidence:

- Repository checks, frontend production build, Linux release build and security
  checks passed. The release artifact passed the secret scan.
- Four SSR routes load Plans, Archives, Restore and Destinations. Plans and runs
  keep independent dotted URL pagination; the shared archive table only renders
  supplied data and contains no fetches.
- Global Playwright verified plan creation (including edited numeric retention),
  archive preview, empty-scope rejection, full and files-only restore submission,
  destination display and explicit Agent checks. Light and dark pages were
  inspected, with no browser console errors in the inspected flows.
- The resumed global Playwright pass verified independent `plans.size=25` and
  `runs.size=50` changes: each requested only its own API list, without document
  reloads. Opening that URL directly preserved both sizes with SSR data and no
  duplicate initial client API requests.
- A temporary, unscheduled plan passed create, edit, reopen and delete through
  the UI. Name, numeric retention and files-only selection persisted; retention
  zero was blocked before an API request. The existing plan remained unchanged.
- At 390 x 844, Plans, Archives, Restore, Destinations, the edit/restore modals
  and sidebar navigation were inspected in light/dark views. The restore preview
  was cancelled without a restore submission. Console errors and warnings: zero.
- An isolated VPS account passed manual and scheduled backups, files-only,
  databases-only, metadata-only and full restores. File contents, MySQL rows,
  Unix ownership/modes and Website reactivation were checked independently.
- Retention removed the older archive from the scheduled plan without touching
  another plan. Requesting an absent database scope returned HTTP 422.
- Unit tests reject corrupt archives, missing metadata, truncated gzip trailers,
  duplicate manifest entries and empty/absent restore scopes before host writes.
- The temporary account, website, database, plans and archives were removed.
  The account home remains recoverable under the panel's quarantine directory.
- Upgrade recovery snapshot:
  `/var/lib/webycp/upgrades/before-0.1.1-rc.18-20260905T145337Z.vk8Lnu`.

Deliverables:

- Normalize Backups into Plans, Archives, Restore, and Destinations views.
- Preserve the existing verified local driver.
- Define the storage driver contract from real local behavior.
- Add the first remote destination only after it is selected.
- Keep restore validation, checksums, ownership repair, and metadata
  reconciliation mandatory for every destination.

Verification:

- Scheduled and on-demand backups pass.
- Selective and complete restores pass.
- Corrupt or incomplete artifacts are rejected.
- Remote failure and retry behavior is tested if a remote driver is added.
- Stop for review.

### Step 7 — Scheduled Tasks and operational visibility

Status: implemented, deployed as `0.1.1-rc.19` and validated on the Ubuntu 24.04
test VPS. Reviewed; Step 8 has been authorized. The archive-format change was
explicitly approved before deployment.

Local acceptance evidence:

- `make check` passed, including regenerated contracts, migration tests, Go vet,
  Go tests, frontend lint/types/tests and both application builds.
- Targeted `-race` tests passed for tasks, worker jobs, SQLite, public API,
  Agent protocol and the crontab driver.
- Migration tests preserve existing schedules, kind/driver, timestamps, queued
  jobs and Package usage. Old audit events gain no fabricated Job correlation.
- Lifecycle tests cover create, update, disable, enable, retry and delete,
  including non-member rejection and Agent failures. Driver tests reject root,
  mismatched ownership and invalid commands while preserving installed files.
- Global Playwright verified local task create/edit/reopen/enable/delete and
  required-field rejection. The temporary task was removed. This isolated
  fixture has no privileged Agent; real Linux command execution is not claimed.
- Jobs detail and filtered Audit Log correctly show accepted requests and final
  failures with the same Job ID. Desktop and 390 x 844 mobile views were inspected
  in light/dark themes; no console errors or warnings occurred.
- Pagination checks pass for 1 → 2 → 1, direct SSR on page 2 and independent
  backup tables. Initial SSR requests are not duplicated; URL changes fetch the
  affected list without document reloads or mount-time cache mutations.
- Security checks passed: no vulnerable Go call paths, no production npm
  advisories, and successful history/tree secret scans plus the negative fixture.
  Govulncheck noted one advisory in a required module outside imported call paths.

VPS acceptance evidence (2026-09-05):

- Release checks, Linux amd64 build, artifact secret scan and transferred
  SHA-256 passed. The upgrade migrated SQLite and restored all eight services;
  Nginx, PHP-FPM and SQLite integrity checks passed.
- Existing task `Smoke marker` migrated to kind `command`; its installed
  crontab remained byte-for-byte unchanged.
- Temporary Account `Step7 VPS QA` exercised create, edit, disable, enable and
  delete through the global Playwright MCP. Cron executed as UID 1003, not root,
  with the Account home as its working directory at 17:30 UTC. Disabling stopped
  the 17:31 invocation; the edited two-minute schedule executed at 17:32.
- A fresh v4 archive retained typed task metadata. Metadata-only restore
  restored the original definition and crontab without replacing account files;
  the restored task executed at 17:33. Deletion removed the crontab and prevented
  the 17:34 invocation. All lifecycle Jobs succeeded, with request and final
  execution audit events linked by Job ID.
- Jobs detail and filtered Audit Log rendered correctly. Unauthenticated
  requests returned 401. Audit pagination 1 → 2 → 1 made one list request per
  changed key and no duplicate initial SSR fetch.
- The temporary task, plan, archive and Account were removed. The Account home
  remains recoverable at `/home/.webycp-trash/94fd374b7df00761674a83baf85ce033`;
  the temporary archive was permanently deleted. Existing Accounts were retained.
- Both pre-upgrade archives (v1 and v3) were copied and checksum-verified under
  `/var/lib/webycp/preserved/before-rc19-Zhl8yTVg`, mode 0700, outside plan retention.
  A new v4 `Smoke backup` archive, `822632185696d847b5fd6fdbdb9a836c`, passed live
  preview verification with files, database contents and metadata. Normal retention
  removed the active v1 archive; its protected copy remains intact.
- Upgrade recovery point:
  `/var/lib/webycp/upgrades/before-0.1.1-rc.19-20260905T172752Z.NNFCR5`.
  This is a control-plane rollback point, not a full-host backup.
- Known stabilization issue: previewing an old-format archive is rejected before
  restore, but the API currently returns a generic 500 instead of a descriptive
  validation error. The fresh v4 preview and restore passed.
- The browser was left open. A user edit to `web/src/index.css` made after the
  release build is preserved locally and is not part of rc.19.

Implementation boundaries:

- `command` is the only supported task kind; HTTP, backup and maintenance tasks
  are not exposed yet. The execution identity is derived from the Account;
  clients cannot submit `runAs` or privileged arbitrary commands.
- Jobs show configuration synchronization, attempts and final outcomes, not
  every command execution performed later by the cron daemon.
- Overview shows timestamped last-observed capabilities. Historical Monitoring
  stays hidden until collection and retention are implemented.
- Backup format 4 replaces `cronJobs` metadata with typed `scheduledTasks`.
  Existing version 3 archives are preserved but rejected by version 4 readers.
  The pre-release compatibility impact was approved before deployment; keep the
  matching old release and recovery snapshot for any required old restore.

Deliverables:

- Rename Cron Jobs to Scheduled Tasks across API, backend, Agent, routes, and
  UI without retaining the old model.
- Keep `crontab` as the only initial scheduler driver.
- Add typed task kinds without exposing unrestricted privileged commands.
- Add Jobs and Audit Log under System.
- Separate current status on Overview from historical Monitoring data.

Verification:

- Schedule validation and account isolation tests pass.
- Creation, update, enable, disable, execution, and deletion pass on the VPS.
- Audit records connect the administrator request to the final Agent outcome.
- No default polling is introduced.
- Stop for review.

### Step 8 — FTP / FTPS

Status: authorized and in progress. Implement this step before Firewall,
Services Management, WAF and final clean-host acceptance. Do not deploy to or
reinstall the existing rc.19 VPS as part of local development.

Reviewable increments:

1. **8.1 — Agent and driver:** typed synchronization, private PureDB state,
   Account chroot, revocation, Linux Account lifecycle hooks and tested TLS
   installation. Include an opt-in Ubuntu integration test and proposed service
   unit. This increment does not install/start FTP on existing machines or add
   menu placeholders. The initial jail is the Account home; an arbitrary path
   or per-Website jail is not part of this contract.
2. **8.2 — Control plane and UI:** metadata, Jobs/audit, permissions, Package
   limits, paginated SSR page and modal actions. Credential changes disconnect
   all FTP sessions belonging to that Account; other Accounts are unaffected.
3. **8.3 — Deployment and acceptance:** installer/upgrade integration, capability
   reporting, panel certificate lifecycle wiring, backup/restore and clean-host
   service acceptance. Review the complete FTP step before starting Firewall.

8.1 acceptance (2026-09-05):

- Agent contract/client/handler and the Pure-FTPd driver are implemented. Shared
  Argon2id hash validation is reused; no dependency or frontend change was needed.
- `make generate`, `make check`, targeted race tests and `make security` pass.
  The Agent also builds for Linux amd64. No release artifact was published.
- The real Ubuntu 24.04 Pure-FTPd 1.0.50 package passed wire-level FTPS tests in
  a native ARM64 container: encrypted control/data, login/upload/download, wrong
  passwords, plaintext rejection, traversal/symlink jail checks, password rotation,
  revocation, suspension/re-enable, deletion preserving files, Account UID/GID
  and mode 0640. A second Account's open session survived another's revocation.
- Certificate replacement was verified by an FTPS client trusting the new
  certificate. Unit tests cover invalid/mismatched/expired PEMs and rollback.
- The amd64 Docker attempt could not complete: packaged `pure-ftpwho` terminates
  with SIGTRAP under the current emulated environment. No workaround was added
  to production code. Native amd64/systemd and clean-VPS acceptance remain open.
- The proposed systemd unit is not wired into installation or upgrades yet;
  certificate lifecycle wiring, control-plane/UI, Packages and backups remain
  8.2/8.3 work. The existing VPS and browser were not touched.

8.2 acceptance (2026-09-05):

- Added FTP metadata, public API, Account authorization, CSRF, durable `ftp.sync`
  Jobs and request/execution audit correlation. Passwords are hashed before
  storage and excluded from public responses, Job payloads and audit metadata.
- Added atomic Package limits/usage (`ftpAccounts`, default 10, maximum 100),
  node-scoped username uniqueness, per-Account queued-write exclusion, partial
  updates merged transactionally and recoverable deletion after Agent revocation.
  Account deletion rechecks resource ownership inside the queue transaction.
- Added File Access > FTP Accounts with SSR, SWR, URL pagination, shared Table,
  modal actions, confirmations, stable pending labels and spinners. No polling
  or mount-time cache seeding. The shared HeroUI modal/confirmation triggers
  were corrected after browser testing exposed accessibility warnings.
- `make generate`, `make check`, targeted race tests and `make security` pass.
  Service tests use the real Agent socket with a recording driver; HTTP tests
  cover auth/CSRF, validation, forbidden overrides and secret-free responses.
  The forward migration preserves existing consolidated-baseline Package data.
- Global Playwright passed local light/dark/mobile checks, page 1 → 2 → 1
  without reload, form validation/conflicts, queued writes and unavailable-Agent
  errors. The disposable UI fixture does not provision real host services.
- A pre-existing initial-admin-setup issue was observed: protected child SSR
  requests may receive 403 while the layout renders the mandatory setup form.
  The setup completes, but this server-side request/error belongs to the final
  stabilization review; authentication was not refactored in this FTP increment.
- No installer or VPS change. Native amd64/systemd, capability reporting,
  certificate lifecycle, backups and clean-host acceptance remain in 8.3.

Deliverables:

- A File Access > FTP Accounts page with SSR data, URL pagination, modal forms,
  focused action components, confirmations and pending spinners.
- Pure-FTPd virtual logins mapped to the existing Account Unix identity, with
  a chroot limited to an authorized existing Account directory. No new Linux
  identity per FTP login and no implicit SSH/SFTP access.
- Create, edit, change password, enable, disable and delete credentials through
  typed API/Agent contracts, durable Jobs and audit events. Never persist or log
  plaintext passwords; deleting credentials leaves customer files untouched.
- Atomic Package limits and usage for FTP credentials; account suspension and
  deletion must not leave working FTP access or active transfer sessions.
- Mandatory TLS for control and data, no anonymous/PAM/Unix login, port 21 and
  an explicit passive range. Service certificate installation and renewal must
  remain protected and must not weaken the existing panel TLS lifecycle.
- Installer, Agent sandbox, capability checks and backup/restore integration.
  Firewall changes belong to the next step, not implicit installer mutations.

Verification:

- Contracts, validation, permissions, Package limits, lifecycle/retry and secret
  handling tests pass across the real Agent socket.
- Ubuntu 24.04 integration checks cover encrypted login/upload/download, wrong
  passwords, plaintext rejection, chroot escape, suspension, password rotation,
  deletion preserving files, and TLS certificate handling.
- Browser checks cover modal forms, errors, pagination and light/dark layouts.
- Record any clean-host acceptance still pending; stop for review.

### Step 9 — Firewall

Status: planned; starts only after Step 8 review.

- Security > Firewall uses firewalld/nftables through a narrow Agent driver.
- Show explicit rules, service profiles and default policy, not an invented
  list of every closed port. A permitted port does not mean a daemon is listening.
- Manage ports/ranges, TCP/UDP and source IP/CIDR for IPv4 and IPv6.
- Protect SSH and panel access, with a local timed rollback independent of the
  browser/API. Detect conflicting firewall owners; do not silently adopt them.
- Test runtime/permanent consistency, reboot persistence and recovery from an
  interrupted rule change. Stop for review.

### Step 10 — Services Management

Status: planned; starts only after Step 9 review.

- System > Services reports real systemd running/stopped/failed state separately
  from configuration validity and timestamped capability observations.
- Start, stop, restart and supported reload operations use an explicit service
  allowlist, typed Agent actions, Jobs, audit events and UI confirmations.
- Never accept arbitrary unit names or shell commands. Protect WEBYCP's own
  API/Agent/Web services and keep firewall access changes in the firewall flow.
- Validate configuration before activating changes and test failure recovery.
  Stop for review.

### Step 11 — WAF

Status: planned; starts only after Step 10 review.

- Security > WAF uses ModSecurity v3, the Nginx connector and OWASP CRS; do not
  implement a custom request-filtering engine.
- Global and per-Website off/detection-only/blocking modes, beginning with
  detection-only; actionable events and narrowly scoped rule/URL exclusions.
- Pin and validate engine/module/rule compatibility, controlled updates and
  rollback. Protect log privacy and define retention.
- Test false positives, representative attacks, CPU/memory/latency and Nginx
  package/module updates. Other web server integrations need separate validation.
  Stop for review.

### Step 12 — Release stabilization

Status: local foundations implemented; clean-host acceptance resumes after
Steps 8–11. No stabilization candidate has been deployed or published. The current
test VPS remains
on `0.1.1-rc.19` with its database and protected old archives unchanged.

Implemented:

- Backup preview returns HTTP 422 with `backup_version` or `backup_invalid` for
  unsupported or damaged archives. Known error codes cross the Agent socket
  without exposing private paths; unrelated internal failures remain 500.
- Backup reads and restore writes use opened account roots. Restored regular
  files are written separately and renamed into place after ownership and mode
  checks, avoiding symlink traversal and writes through existing hard links.
- The twelve development migrations were replaced with `0001_initial.sql`,
  preserving the final schema and typed queries without obsolete conversions.
  The new baseline has no administrator or customer fixture data. Removed SQL
  and migration-only tests remain recoverable through Git history.
- Server `check-schema` checks migration history read-only. Installer and upgrade
  preflight reject old or unknown histories before host changes. There is no
  in-place conversion from the rc19 development database to this baseline.
- README, packaging and operations documentation describe the new boundary,
  current resources, archive errors and per-file restore behavior.

Local acceptance evidence:

- `make generate`, `make check` and targeted race tests pass. Generated `sqlc`
  output is unchanged after consolidation; fresh-schema tests cover empty
  operational tables, default Package state and valid foreign-key targets.
- `make security` passes: no reachable Go vulnerabilities, no production npm
  vulnerabilities, and history/tree secret scans plus the negative fixture
  pass. One advisory exists in a required Go module outside imported packages.
- Linux amd64 Server and Agent builds pass. The backup suite passes as UID/GID
  1000 in an isolated Ubuntu 24.04 container; filesystem isolation regressions
  also pass as root. Fresh migration, repeated migration, read-only schema
  validation and rejection of a missing database pass in that container.
- Tests reject old/future migration histories without replacing existing data,
  reject old/future/corrupt archives before restore writes, and preserve the
  previous file when a replacement cannot be copied completely.

Remaining release gates:

- Validate a clean installation on a separate fresh Ubuntu 24.04 VPS; do not
  reinstall or clear the current test host. A second host has been requested.
- Implement and validate the uninstall lifecycle with an explicit data-retention
  boundary; no uninstall or purge has been run.
- Exercise upgrade and recovery between candidates sharing this baseline, run
  the full v1 lifecycle suite and scan the final release archive before publishing.
- Container tests do not replace systemd, hosting-service or full-host acceptance.

Deliverables:

- Return a descriptive validation error for unsupported or invalid backup
  archives instead of a generic 500, without adding old-format compatibility.
- Remove development-only data and obsolete migrations.
- Squash the schema into the first public baseline.
- Complete installation, upgrade, rollback, backup, and disaster-recovery
  documentation.
- Run security, contract, integration, release artifact, and VPS acceptance
  checks.
- Publish the first release candidate only after the full clean install and
  uninstall lifecycle passes.

Verification:

- A fresh Ubuntu 24.04 VPS can install and reach a healthy panel.
- Every v1 lifecycle works without manual database or filesystem repair.
- Upgrade rollback preserves the last working panel.
- Release archives pass checksum and secret scans.

## 11. Definition of Done for Every Step

A step is complete only when:

1. Its accepted behavior is implemented end to end.
2. No compatibility hack, placeholder, duplicated path, or abandoned code from
   that step remains.
3. Generated contracts and migrations match the implementation.
4. Relevant automated tests pass.
5. Browser behavior is checked where UI changed.
6. VPS behavior is checked where host state changed.
7. Documentation is updated only for behavior that exists.
8. The diff is reviewed before the next step begins.

## 12. Immediate Next Step

Review **8.2 — FTP control plane and UI**, then implement **8.3 — FTP deployment
and acceptance**: installer/upgrade, capabilities, panel certificate lifecycle,
backup/restore and native amd64/systemd tests. Finish FTP before starting Firewall,
Services Management and WAF, in that order.

Clean-install, uninstall and recovery acceptance remain Step 12 and require a
separate fresh Ubuntu 24.04 VPS. Preserve the existing `0.1.1-rc.19` test host;
the consolidated schema is not an in-place upgrade from its development history.
Do not publish until the remaining release gates pass.

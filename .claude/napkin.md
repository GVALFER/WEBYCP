# Napkin Runbook

## Curation Rules

- Re-prioritize on every read.
- Keep recurring, high-value notes only.
- Max 10 items per category.
- Each item includes date + "Do instead".

## Shell & Tool Reliability

1. **[2026-09-05] sqlc SQLite queries must not mix named and anonymous parameters**
   Do instead: use `sqlc.arg(...)` for every parameter in a query once one is named; mixed `?` and generated numbered placeholders can compile but fail at runtime with missing arguments.

1. **[2026-09-05] Playwright MCP code runs in a restricted JavaScript context**
   Do instead: use built-in snapshots and network inspection instead of persistent event listeners; unavailable globals such as `URL` can crash callbacks and terminate the transport. Keep the user's browser and tabs open.
1. **[2026-09-05] Removed Next.js routes can leave stale dev type validators**
   Do instead: move only the generated `.next/dev` cache aside when it still imports deleted pages; regenerate build types instead of restoring obsolete routes.
1. **[2026-09-04] Release builds require Node.js 24 and Docker Buildx**
   Do instead: verify `node --version` before release, select the installed NVM Node 24 when needed, and build the pinned Linux amd64 Next.js standalone runtime through Docker.
2. **[2026-09-04] Gitleaks must recognize Next.js dev build keys**
   Do instead: allowlist only the exact generated `.next/dev` key and manifest paths alongside their production equivalents; never bypass the working-tree secret scan.
3. **[2026-09-04] Browser QA may clean tracked screenshot artifacts**
   Do instead: compare `git status` before and after Playwright, then restore only exact artifacts removed by the browser session.
4. **[2026-09-04] Docker Desktop build cache can exhaust its internal disk**
   Do instead: inspect `docker system df` first and prune only disposable builder cache when a release fails with `ENOSPC`; preserve containers, images in use, and volumes.
5. **[2026-09-04] Next.js may generate nested agent instruction files during development**
   Do instead: keep `agentRules: false` in `next.config.ts` so `next dev` does not create `web/AGENTS.md` or `web/CLAUDE.md`.

## Execution & Validation (Highest Priority)

1. **[2026-09-04] Security checks must cover source, history, and release artifacts**
   Do instead: run pinned `govulncheck`, production `npm audit`, and redacted Gitleaks scans; prove leak detection with a temporary negative fixture and scan every archive before checksumming it.
2. **[2026-09-03] Keep test-host credentials out of the repository and logs**
   Do instead: enter authorized test credentials through a non-echoing prompt, verify the target read-only, confirm recovery, and state destructive scope before remote integration tests. Recommend rotation, but obtain authorization before changing passwords or installing SSH keys.
3. **[2026-09-03] Scope Go package commands away from frontend dependencies**
   Do instead: run Go checks against `./cmd/... ./internal/...`; `./...` may traverse Go packages embedded in `web/node_modules`.
4. **[2026-09-03] Treat SSL lifecycle as a complete v1 requirement**
   Do instead: cover panel and hosted-domain certificates, ACME issuance, renewal, expiry state, validation, atomic installation, and safe Nginx reloads.
5. **[2026-09-03] Keep the control panel architecture modular without overbuilding v1**
   Do instead: define narrow service interfaces and ship only the initial implementation for each v1 capability.
6. **[2026-09-03] Test privileged flows across the real Unix-socket boundary**
   Do instead: verify API cookie/CSRF behavior, durable job transitions, Agent protocol validation, and the Agent's effective systemd mount permissions together before extending Agent operations.
7. **[2026-09-03] Activate generated service configs only through validation and rollback**
   Do instead: use `agent/configfile` snapshots and atomic writes, run the service's config test, and restore the previous file before a recovery reload on failure; keep Nginx sites behind WEBYCP's exact include anchor.
8. **[2026-09-03] Keep resource deletion recoverable and retry-safe**
   Do instead: complete the typed Agent operation before deleting metadata, quarantine customer files under the account by immutable resource ID, and make job replay treat already-finished deletion as success.
9. **[2026-09-03] Preserve live panel TLS while renewing certificates**
   Do instead: keep the last valid panel listener active while preparing HTTP-01, and replace its Nginx configuration only after Certbot has produced a valid certificate.
10. **[2026-09-03] Separate durable state by privilege boundary**
   Do instead: keep Server-owned SQLite files under `/var/lib/webycp/server`; keep account quarantine on the `/home` filesystem and other Agent-owned recovery, ACME, certificate, and backup data outside directories writable by the unprivileged Server.

## Project Conventions

1. **[2026-09-03] Keep v1 fully self-hosted without a required external control plane**
   Do instead: install the UI build, API, state, Agent, and hosting services on the managed Ubuntu server; permit only explicit infrastructure dependencies such as package mirrors and ACME.
2. **[2026-09-04] Use the agreed self-hosted application architecture**
   Do instead: build a Next.js App Router frontend with SSR, an unprivileged Go REST API, and a privileged Go agent targeting Ubuntu 24.04 LTS; run the Next standalone server as an unprivileged service.
3. **[2026-09-03] Route protected mutations through the established security path**
   Do instead: require the `HttpOnly` session, CSRF header, appropriate role, a durable job for host changes, and audit both the request and final execution outcome.
4. **[2026-09-03] Never adopt pre-existing host identities by username alone**
   Do instead: derive `wcp_*` usernames from resource IDs and verify the passwd-safe `WEBYCP-<resource-id>` ownership marker before treating a Linux user as managed.
5. **[2026-09-03] Make privileged filesystem changes through no-follow descriptors**
   Do instead: open trusted roots and child directories with `O_NOFOLLOW`, then apply ownership and modes through file descriptors instead of mutable absolute paths.
6. **[2026-09-03] Use SWR for frontend server state**
   Do instead: use `reqly-js` as the HTTP transport, SWR for cache/revalidation, and `urlstate-js` for URL query state.
7. **[2026-09-03] Reuse stable helpers without creating a generic backend dumping ground**
   Do instead: place pure frontend helpers in `web/src/utils` and shared Go functions in small purpose-named packages such as `fsx`, `execx`, or `validate`.
8. **[2026-09-03] Keep code and UI copy in English**
   Do instead: use English for identifiers, comments, errors, labels, and other UI text unless explicitly requested otherwise.
9. **[2026-09-04] Enforce Account Package limits atomically**
   Do instead: check the assigned Package and current usage inside the same serialized SQLite transaction that inserts the resource and job; let lowered limits preserve existing resources while blocking only new capacity, including restore upserts.
10. **[2026-09-04] Keep Websites separate from hostname bindings**
    Do instead: persist the vhost, document root, and explicit stack on `Website`; keep primary and alias hostnames in one global `WebsiteDomain` namespace; key host paths and service configuration by immutable Website ID so hostname changes never move customer files.

## Systemd & Runtime

1. **[2026-09-04] Gate upgrades on offline state snapshots and health checks**
   Do instead: stop Web, Server, then Agent; snapshot binaries, frontend, units, panel Nginx config, and SQLite; automatically restore unless Agent, Server, Web, Nginx, and version checks all pass.
2. **[2026-09-04] Go interface discovery needs Netlink inside the Agent sandbox**
   Do instead: retain `AF_NETLINK` in the Agent's `RestrictAddressFamilies` whenever certificate DNS preflight uses `net.InterfaceAddrs`.
3. **[2026-09-04] Agent operations can legitimately exceed five seconds**
   Do instead: keep a multi-minute internal client timeout for Certbot and backup operations while relying on job state for user-facing progress.
4. **[2026-09-04] Metadata restore must preserve active TLS**
   Do instead: reconcile each active domain first, then reapply its active certificate before completing the restore job.
5. **[2026-09-04] Upgrade snapshots are rollback points, not host backups**
   Do instead: protect service configuration, certificates, customer homes, MySQL state, local artifacts, and Server state together in a tested off-host or infrastructure snapshot.
6. **[2026-09-04] An active systemd service may not be ready yet**
   Do instead: after starting the Agent or Server, retry the Unix-socket or HTTP health endpoint for up to 20 seconds before declaring recovery successful.
7. **[2026-09-04] Restored files must return to the account privilege boundary**
   Do instead: validate the managed Unix identity, restore account ownership, map the web tree to `www-data`, and preserve safe modes including directory `setgid`.
8. **[2026-09-04] Remove account runtimes before deleting Unix identities**
   Do instead: remove and validate/reload the account PHP-FPM pool with rollback first; only then delete the managed user so future PHP-FPM reloads cannot fail on an orphaned pool.
9. **[2026-09-04] Initialize panel credentials under the Server identity**
   Do instead: generate one-time admin credentials in the Go command, write only their hash as `webycp`, force profile completion, and invalidate sessions after root-assisted resets.
10. **[2026-09-04] Separate observed capabilities from configured resource drivers**
    Do instead: let the Agent report honest per-service health, snapshot it on the Server, use global defaults only to preselect create forms, and persist the selected driver on every resource.

## DNS & Provider State

1. **[2026-09-05] DNS record retries must retain the last applied RRset identity**
   Do instead: persist the last successfully synchronized record name and type, then replace both the old and desired RRsets atomically so repeated failed renames or type changes cannot leave orphaned provider records.
2. **[2026-09-05] Keep local DNS credentials inside the Agent boundary**
   Do instead: bind the PowerDNS API to loopback, keep its key in a root-only file, pass only typed DNS operations across the Agent socket, and reject deletion or mutation of zones without WEBYCP's immutable ownership marker.
3. **[2026-09-05] PowerDNS API zone defaults contain an invalid SOA primary**
   Do instead: create each zone atomically with explicit SOA and NS RRsets plus `SOA-EDIT-API=DEFAULT`; verify that later API record changes advance the serial.

## User Directives

1. **[2026-09-03] Converse in European Portuguese**
   Do instead: write all user-facing conversation in Portuguese from Portugal.
2. **[2026-09-03] Do not use Prettier for frontend code**
   Do instead: leave TypeScript, React, CSS, and JSON formatting to the user's editor; retain `gofmt` for Go source files.
3. **[2026-09-03] Prefer the global Playwright MCP for browser QA**
   Do instead: use the globally available Playwright MCP for local UI validation when it is callable.
4. **[2026-09-04] Never poll frontend resources by default**
   Do instead: let SWR fetch on mount and revalidate on focus, then call `mutate` after successful writes.
5. **[2026-09-04] Avoid native browser dialogs and repeated request feedback**
   Do instead: use the shared HeroUI `Confirm` component for destructive choices, a HeroUI modal for editable prompts, and HeroUI toasts for action feedback; keep async button labels stable and show a HeroUI `Spinner` through `isPending` while the action runs.
6. **[2026-09-04] Resolve the initial session on the server**
   Do instead: fetch `auth/me` in the protected Next.js server layout, redirect before sending HTML, and seed one client session context; use the reqly `401` handler only for expired protected requests and exclude `auth/login` and `auth/me`.
7. **[2026-09-04] Format panel dates in the administrator timezone**
   Do instead: persist the IANA timezone on the administrator, expose it on the session, and format client dates through `useTimezone().dt` with UTC as the migration and runtime fallback.
8. **[2026-09-04] Keep pre-release changes free of compatibility hacks**
   Do instead: replace obsolete development models cleanly, avoid dual reads/writes, deprecated routes, fabricated fallbacks, and temporary adapters, then stop after each reviewed step.
9. **[2026-09-04] Load each route's initial data in its Server Component**
   Do instead: make every route `page.tsx` fetch its initial data on the server and pass typed props to an adjacent Client Component with explicit SWR `fallbackData`. Do not seed the cache with mount-time `mutate`; suppress initial revalidation and let changed keys fetch normally, including a return to the SSR page. Keep `useTable` for URL state only and rendering/loading/`Paginate` inside `Table`; namespace multiple tables with dotted parameters such as `plans.page`, keeping API keys as `?page=&size=`. Keep the shell in the four `components/layout` files, without a `features` directory.
10. **[2026-09-04] Keep all mutation logic in action components**
   Do instead: keep page clients focused on SWR fallback and presentation; move every POST, PATCH, PUT, DELETE, pending state, validation, toast, mutate, and mutation button/form into focused components under the adjacent `actions` directory; build submitted forms with React Hook Form, `valibotResolver`, the shared `Form`/field components, and `useTransition`, without manual `safeParse` calls in submit handlers; open resource creation forms in the shared HeroUI modal instead of permanent page sidebars.

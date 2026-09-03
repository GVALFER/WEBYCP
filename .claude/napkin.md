# Napkin Runbook

## Curation Rules

- Re-prioritize on every read.
- Keep recurring, high-value notes only.
- Max 10 items per category.
- Each item includes date + "Do instead".

## Execution & Validation (Highest Priority)

1. **[2026-09-03] Scope Go package commands away from frontend dependencies**
   Do instead: run Go checks against `./cmd/... ./internal/...`; `./...` may traverse Go packages embedded in `web/node_modules`.
2. **[2026-09-03] Keep test-host credentials out of the repository and logs**
   Do instead: rotate any chat-shared password, install a temporary SSH key, verify the target read-only, confirm recovery, and state destructive scope before remote integration tests.
3. **[2026-09-03] Treat SSL lifecycle as a complete v1 requirement**
   Do instead: cover panel and hosted-domain certificates, ACME issuance, renewal, expiry state, validation, atomic installation, and safe Nginx reloads.
4. **[2026-09-03] Keep the control panel architecture modular without overbuilding v1**
   Do instead: define narrow service interfaces and ship only the initial implementation for each v1 capability.
5. **[2026-09-03] Test privileged flows across the real Unix-socket boundary**
   Do instead: verify API cookie/CSRF behavior, durable job transitions, Agent protocol validation, and observed node state together before extending Agent operations.
6. **[2026-09-03] Activate generated service configs only through validation and rollback**
   Do instead: use `agent/configfile` snapshots and atomic writes, run the service's config test, and restore the previous file before a recovery reload on failure; keep Nginx sites behind WEBYCP's exact include anchor.
7. **[2026-09-03] Keep resource deletion recoverable and retry-safe**
   Do instead: complete the typed Agent operation before deleting metadata, quarantine customer files under the account by immutable resource ID, and make job replay treat already-finished deletion as success.
8. **[2026-09-03] Preserve live panel TLS while renewing certificates**
   Do instead: keep the last valid panel listener active while preparing HTTP-01, and replace its Nginx configuration only after Certbot has produced a valid certificate.

## Project Conventions

1. **[2026-09-03] Keep v1 fully self-hosted without a required external control plane**
   Do instead: install the UI build, API, state, Agent, and hosting services on the managed Ubuntu server; permit only explicit infrastructure dependencies such as package mirrors and ACME.
2. **[2026-09-03] Use the agreed three-part application architecture**
   Do instead: build a React/Vite SPA, an unprivileged Go REST API, and a privileged Go agent targeting Ubuntu 24.04 LTS.
3. **[2026-09-03] Route protected mutations through the established security path**
   Do instead: require the `HttpOnly` session, CSRF header, appropriate role, a durable job for host changes, and audit both the request and final execution outcome.
4. **[2026-09-03] Never adopt pre-existing host identities by username alone**
   Do instead: derive `wcp_*` usernames from resource IDs and verify the `WEBYCP:<resource-id>` ownership marker before treating a Linux user as managed.
5. **[2026-09-03] Make privileged filesystem changes through no-follow descriptors**
   Do instead: open trusted roots and child directories with `O_NOFOLLOW`, then apply ownership and modes through file descriptors instead of mutable absolute paths.
6. **[2026-09-03] Use SWR for frontend server state**
   Do instead: use `reqly-js` as the HTTP transport, SWR for cache/revalidation, and `urlstate-js` for URL query state.
7. **[2026-09-03] Reuse stable helpers without creating a generic backend dumping ground**
   Do instead: place pure frontend helpers in `web/src/utils` and shared Go functions in small purpose-named packages such as `fsx`, `execx`, or `validate`.
8. **[2026-09-03] Keep code and UI copy in English**
   Do instead: use English for identifiers, comments, errors, labels, and other UI text unless explicitly requested otherwise.
9. **[2026-09-03] Prefer concise arrow functions and practical `div` elements**
   Do instead: follow the repository's established style, defaulting new JavaScript/TypeScript functions to arrow functions and concise names.
10. **[2026-09-03] Treat domains and aliases as one global hostname namespace**
    Do instead: normalize names before persistence and reject collisions against primary, alias, and reserved previous names in the same transaction until rename reconciliation finishes.

## User Directives

1. **[2026-09-03] Converse in European Portuguese**
   Do instead: write all user-facing conversation in Portuguese from Portugal.
2. **[2026-09-03] Do not use Prettier for frontend code**
   Do instead: leave TypeScript, React, CSS, and JSON formatting to the user's editor; retain `gofmt` for Go source files.
3. **[2026-09-03] Prefer the global Playwright MCP for browser QA**
   Do instead: use the globally available Playwright MCP for local UI validation when it is callable.

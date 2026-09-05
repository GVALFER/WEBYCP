# FTP / FTPS implementation

## Current boundary

Steps 8.1 and 8.2 implement the Agent/driver, public API and File Access > FTP
Accounts UI. FTP is not installed by the release scripts yet. No FTP ports have
been opened and the development VPS has not been updated for these increments.

Step 8.3 adds installer/upgrade, observed capabilities, certificate lifecycle
and backup/restore integration. Do not deploy this as a completed FTP feature.

## Control plane

- `GET /api/v1/ftp-accounts?page=1&size=10` lists only accessible Account logins.
- `POST /api/v1/ftp-accounts` accepts `accountId`, `username`, `password`, `enabled`.
- `PATCH /api/v1/ftp-accounts/{id}` accepts username, password and/or enabled state;
  omitting password preserves it. The owning Account and home cannot be changed.
- `DELETE /api/v1/ftp-accounts/{id}` queues revocation without deleting files.

Writes require session authentication, CSRF and Account membership (or admin).
Passwords use the shared 12–128-character validation and Argon2id hashing before
storage. Public responses contain no password/hash; Job payloads contain only
the Account ID. Request and execution audit events share the Job ID. Driver
response details are not propagated into public Job errors.

Migration `0002_ftp.sql` adds private credential metadata and the `ftpAccounts`
Package limit/usage, preserving the consolidated baseline's existing data.
The default limit is 10 and the maximum is 100, matching the Agent contract.
Disabled or deleting credentials still consume capacity; only a successful
revocation frees it. Lowered limits preserve existing logins and allow edits,
password rotation, disable and deletion. Names remain unique per node until
revocation succeeds, including disabled logins.

Credential changes and Job insertion share a serialized SQLite transaction.
One queued/running `ftp.sync` Job blocks other FTP mutations in the same Account.
PATCH fields merge inside that transaction; the worker synchronizes the complete
Account snapshot. A failed Job leaves recoverable metadata and a safe error.
Repeat the edit or deletion after failure to queue a new attempt. A deleting
login cannot be edited back into service. Account deletion rechecks resource
ownership in its queue transaction and is also protected by a restrictive FK.

The page uses SSR props, explicit SWR fallback data, URL-state pagination and
the shared Table. Forms and mutations live in `actions/`; no polling, initial
cache seeding or table fetcher was added. A non-active hosting Account is shown
separately from the credential's desired enabled state.

## Ownership and storage

`PUT /agent/v1/ftp-accounts` synchronizes an Account's complete list of virtual
logins. It is available only over the existing protected Agent Unix socket.
The request carries Account identity, immutable login IDs, usernames, enabled
flags and password hashes. It accepts no UID, GID, shell or filesystem path.

The Agent verifies the existing `WEBYCP-<accountId>` Unix ownership marker and
derives UID/GID and `/home/wcp_<accountId[:12]>`. This directory is the chroot for
every FTP login in the Account. A Website-level jail requires a separate design;
changing the starting directory is not equivalent to restricting access.

Authentication state lives outside customer homes:

```text
/etc/webycp/ftp/                  0700, Agent-owned
|-- accounts/                    0700
|   `-- <accountId>.json          0600, hashes and desired login state
|-- pureftpd.pdb                 0600, compiled authentication database
`-- tls.pem                      0600, certificate chain and private key
```

The Argon2id implementation is shared with panel authentication. Plaintext
passwords never enter the Agent protocol, process arguments, authentication
files, errors or logs. PureDB is built in a private temporary directory using
Ubuntu's `pure-pw`, then installed by atomic rename. A failed compile preserves
the previous source and database. Synchronizations are serialized in the Agent.

Disabled logins remain in the private source but are omitted from PureDB.
Account suspension is independent of each login's enabled flag. Re-enabling
an Account does not re-enable individually disabled credentials.

Credential changes disconnect existing FTP sessions for that Account, including
sessions authenticated using an old password. Other Accounts and non-FTP
processes are not signalled. Disconnection uses the real Pure-FTPd session
report, verifies the process executable and effective UID, and signals a Linux
process handle. A failed disconnect leaves revoked credentials absent and is
reported as a failure so the durable Job can retry.

Removing FTP access never removes customer files. Account deletion removes FTP
access before deleting the Unix user; the existing Account quarantine remains
responsible for customer files.

## Transport and certificates

The proposed `packaging/systemd/webycp-ftp.service` runs the standard Ubuntu
`pure-ftpd` binary, not its `virtualchroot` variant. It uses only PureDB login,
with anonymous, PAM and Unix authentication unavailable through that service.

- Explicit FTPS, TCP 21; TLS is required for both control and data (`--tls=3`).
- Passive TCP range: 40000–40100. No firewall changes are made by this increment.
- New uploads use mode 0640 and directories 0750 (`--umask=137:027`).
- The driver validates certificate/key matching and validity, installs a private
  combined PEM and restarts only `webycp-ftp.service`. It restores the previous
  PEM and attempts recovery if activation fails. Unchanged PEMs do not restart.
- Wiring this operation to panel certificate issuance/renewal, hostname guidance
  and initial certificate provisioning belongs to Step 8.3. It is not live yet.

The service must not be enabled alongside another daemon already owning the FTP
listener. The deployment step must check this explicitly and must not silently
adopt an existing Pure-FTPd configuration.

## Verification

Step 8.2 passed `make generate`, `make check`, `make security` and targeted Go
race tests. Service tests cross the real Agent Unix socket with a recording
driver; they cover hashing, rotation, permissions, Account status, per-node
names, queued-write exclusion, Package capacity, failed revocation and replay.
HTTP tests cover missing auth/CSRF, forbidden identity/path/hash overrides,
request validation, secret-free responses and request/audit correlation.
Migration tests verify existing Package data is preserved.

Global Playwright checks used an isolated local database and API without a
privileged Agent: SSR rows, page 1 → 2 → 1 (one API request per changed key),
validation, duplicate usernames, queued create/edit/disable/delete, failed
synchronization state, cleared passwords, pending spinners, confirmations,
light/dark and mobile. Host-side successful provisioning is covered by the
socket tests and the 8.1 FTPS integration, not by that browser fixture.

The shared modal/confirmation triggers now follow the documented
[Modal](https://heroui.com/en/docs/react/components/modal) and
[AlertDialog](https://heroui.com/en/docs/react/components/alert-dialog) composition:
the Button is a direct child of the root, avoiding duplicate accessible triggers
and the missing-pressable-child warning.

Normal tests run with `go test ./internal/agent/ftp/...` and the Agent account,
server/client and authentication package suites. They cover identity/path and
hash validation, duplicate usernames, private state, idempotence, suspension,
revocation retries, compile rollback, TLS rollback and the real Unix socket.

The opt-in `TestUbuntuFTPS` test uses the real Ubuntu Pure-FTPd and Python's
standard-library FTPS client, random test passwords and a locally trusted test
certificate. It reads the actual proposed service `ExecStart` and changes only
the temporary state path and loopback listening port. It creates and removes a
dedicated Unix Account and refuses to run outside an opted-in root container.

Build with `GOOS=linux GOARCH=<container architecture> CGO_ENABLED=0 go test
-tags integration -c ./internal/agent/ftp/pureftpd`. In a disposable Ubuntu 24.04
container, install `pure-ftpd`, `python3` and `openssl`; add Docker capabilities
`DAC_READ_SEARCH`, `SYS_NICE` and `SYS_PTRACE` for the daemon and Agent checks.
Run the test binary with `WEBYCP_FTP_INTEGRATION=1` and `WEBYCP_FTP_UNIT` pointing
to the proposed service unit. Do not run these fixtures on a customer host.

On 2026-09-05 this passed with Ubuntu 24.04 / Pure-FTPd 1.0.50 in a native ARM64
container. The amd64 emulated attempt failed because the packaged `pure-ftpwho`
terminated with SIGTRAP; production code does not bypass session verification.
The Agent cross-build for Linux amd64 passes, but this is not native amd64
daemon acceptance.

Container protocol tests do not validate systemd sandboxing, installation,
upgrade/recovery or real network/firewall reachability. Those remain mandatory
clean-VPS acceptance gates before FTP is considered complete.
That acceptance must also cover authentication already in flight during an
access change, alongside the established-session revocation tested here.

Upstream references: [virtual users and PureDB](https://github.com/jedisct1/pure-ftpd/blob/master/README.Virtual-Users),
[mandatory TLS modes](https://github.com/jedisct1/pure-ftpd/blob/master/README.TLS).

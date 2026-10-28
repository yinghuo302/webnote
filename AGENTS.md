# Instructions for AI agents working in this backend

These instructions apply to this repository (`webnote/`) and all its subdirectories.

## Repository boundaries

- This is the Go 1.26 backend/API repository. The independently versioned browser frontend lives alongside it in `../webnote-web/`; do not copy or generate frontend files under this repository's `public/` directory. Nginx serves the frontend build and forwards `/api/` to Go.
- The local target is a single-host native installation: Go in `/opt/webnote/bin`, application data in `/opt/webnote/data`, systemd runs as the existing low-privilege `web` account. Docker/Compose files are compatibility options only; do not start Docker or replace the local deployment with a container.
- Read `README.md` and relevant `docs/*.md` before changing authentication, storage, or deployment. Inspect `git status --short` before edits; preserve existing user changes.
- Backend and frontend are separate Git repositories/remotes. Verify `git remote -v` and branch before committing/pushing; never push to a different host or repository merely because it is called `origin`.

## Security invariants

- PostgreSQL is the only application database. Keep the application's database role separate from the OS `web` account: Nginx also runs as `web` on the current host. Never configure PostgreSQL peer authentication for OS user `web`.
- Production secrets are root-owned, mode `0600`, at `/opt/webnote/config/webnote.env` and injected with systemd `LoadCredential`. The app reads the credential file via `CREDENTIALS_DIRECTORY`; do not replace this with `EnvironmentFile`, expose secrets as command-line arguments, print credentials, commit `.env`, or put credentials under `/opt/webnote/www`.
- Do not weaken the opaque server-side session design: browser cookie contains only a random 256-bit token; PostgreSQL stores its SHA-256 digest. Lifetime is 24 hours. Preserve `Secure`, `HttpOnly`, `SameSite=Strict`, `Path=/` and the `__Host-` name when secure cookies are enabled. Logout, password changes and `/api/private/logout-all` must revoke server-side records.
- Preserve strict unsafe-request Origin checks, loopback-only backend binding, trusted-proxy restrictions, upload/body limits, login throttles and email-code controls. A stolen, unrevoked session can be replayed; document rather than overclaim protection.
- Never render user Markdown as trusted HTML. If frontend rendering changes are necessary, work in `../webnote-web/` and validate against XSS. Do not change the separate frontend repo unless the task asks for it.

## Development and verification

From this repository root:

```sh
gofmt -w <changed Go files>
WEBNOTE_TEST_DATABASE_URL='postgres://.../dedicated_test_db?sslmode=disable' go test ./... -count=1
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/webnote .
```

- Use only a disposable PostgreSQL test database. Tests create/drop unique temporary schemas; do not point `WEBNOTE_TEST_DATABASE_URL` at a production database. Database tests are skipped if it is unset.
- Run `go mod tidy` after dependency changes and review the complete `go.mod`/`go.sum` diff. Do not reintroduce SQLite/MySQL runtime drivers or deprecated SMTP dependencies into the API.
- The SQLite reader exists only in `cmd/import-sqlite`; it uses Python's read-only SQLite support. Run its `--check` before any import. Never import automatically: duplicate normalized emails, missing Markdown files, or a non-empty PostgreSQL target must stop the operation pending an explicit data-owner decision.
- If the frontend is intentionally changed, build/check it from `../webnote-web/` (for example `pnpm exec tsc --noEmit` and `pnpm exec vite build`). Do not let package managers rewrite its lockfile as an incidental backend change.

## Deployment and Git

- Deployment templates are under `deploy/`. `install-local.sh` is privileged and deliberately does not start the service or alter the active Nginx site. Review it before asking the user to execute it; do not install/reload services or alter `/etc/nginx` without explicit authorization.
- Replace Nginx `YOUR_DOMAIN` and certificate paths only after the user supplies them and a valid certificate exists. Run `nginx -t` with appropriate privileges before reloading; do not enable a placeholder TLS server.
- Keep operational architecture and threat-model guidance in `docs/` and update the relevant document when behavior changes.
- Before committing, review `git diff --check`, `git status`, staged paths and secrets. Do not include `.env`, SQLite databases/backups, compiled binaries, local data, or unrelated frontend work. Before pushing, verify the user-requested GitHub URL explicitly; a Coding.net remote is not GitHub.

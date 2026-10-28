# Architecture and repository boundaries

## Components

- `webnote/` (this repository) is the Go 1.26 JSON API and PostgreSQL application. It does not embed, build, or serve HTML/CSS/JavaScript.
- `../webnote-web/` is the separate browser frontend repository. Build its production output there; Nginx owns static serving and routes `/api/` to `127.0.0.1:8000`.
- PostgreSQL is the runtime system of record. Native deployments use a dedicated database role `webnote`, distinct from the Linux `web` service/Nginx account. Docker/Compose remains an optional compatibility path, not the preferred local deployment.
- The intended single-host filesystem is `/opt/webnote/bin` (root-owned API binary), `/opt/webnote/www` (read-only frontend assets), `/opt/webnote/data/img` (uploads) and `/opt/webnote/data/legacy` (temporary fallback for pre-migration Markdown files). PostgreSQL owns its data directory separately.

## Request flow

Nginx serves the SPA and forwards API requests. For API proxy requests it sets a fixed configured `Host`, replaces forwarding headers with the actual client address and scheme, and applies an additional authentication-route rate limit. The Go server listens on loopback by default, trusts only loopback proxies, applies request timeouts/body limits, and performs same-origin validation for unsafe methods. Private routes then validate a database-backed session before calling handlers.

Do not expose the Go listener directly to the internet. A public deployment requires HTTPS at Nginx, the production `Secure` cookie configuration and a real domain/certificate; the sample Nginx configuration intentionally has placeholder hostnames.

## Storage model

GORM connects only to PostgreSQL and migrates `users`, `logins`, `codes`, `files`, and `sessions` at startup. `files.content` stores Markdown bodies; new saves update body and note metadata together. The legacy `data/<uuid>.md` file read is a compatibility fallback for unmigrated records, not the primary storage design. Images live in the configured image directory; the installer places them under `/opt/webnote/data/img`.

`cmd/import-sqlite` is a one-time offline importer, not an application runtime driver. It exports SQLite read-only through Python's standard library and imports users, logins and notes in one PostgreSQL transaction. It requires a new/empty PostgreSQL target and existing Markdown bodies. It refuses duplicate normalized email groups and missing body files; there is intentionally no automatic account merge or data deletion.

## Important code areas

- `main.go`: validated configuration, PostgreSQL lifecycle, HTTP timeouts, trusted proxy boundary and graceful shutdown.
- `config/`: environment configuration plus systemd `LoadCredential` file reading.
- `db/`: GORM models and PostgreSQL operations; session token hashing/revocation and password/code verification.
- `server/route.go`: route table, per-process rate limiting and unsafe-request origin validation.
- `server/handlers/`: input validation, ownership checks, note/image APIs.
- `server/middleware/auth.go`: cookie naming/attributes and server-side session lookup.
- `deploy/`: reviewed-but-not-automatically-enabled systemd, Nginx and `/opt/webnote` staging templates.

## Build and test

```sh
gofmt -w <changed Go files>
WEBNOTE_TEST_DATABASE_URL='postgres://.../dedicated_test_db?sslmode=disable' go test ./... -count=1
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/webnote .
```

Database tests create a unique temporary schema and drop it afterward. Supply only a disposable test database. With no test URL, PostgreSQL integration tests are skipped; do not treat that as database verification. Check `go list -deps .` if a change might accidentally add a runtime frontend or SQLite dependency.

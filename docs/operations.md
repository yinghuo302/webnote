# Operations: native single-host deployment

## Current deployment posture

The intended deployment is native PostgreSQL + Go systemd service + Nginx static frontend; Docker is optional compatibility only. The checked-in installer deliberately **does not** start Webnote or change/reload the active Nginx configuration. It stages files, reloads systemd units, and enables the five-minute Nginx log-rotation timer. The site is `webnote.zanilia.eu.org`, listens only on 443, and intentionally has no HTTP port-80 listener. The certificate is `/etc/letsencrypt/live/zanilia.eu.org/{fullchain,privkey}.pem`; enable only after `nginx -t` succeeds.

## Build artifacts

From this backend repository:

```sh
CGO_ENABLED=0 go build -trimpath -o bin/webnote .
```

Build the separate frontend from its own repository:

```sh
cd ../webnote-web
pnpm run build
```

Output is `../webnote-web/dist`; it is not copied into this Go repository. Frontend/API paths are same-origin `/api/...`. The checked-out frontend currently has uncommitted editor changes; do not publish a build from that tree until the user approves the frontend version. If not built, `/opt/webnote/dist` remains empty and Nginx will not have a SPA to serve yet.

## PostgreSQL and credential setup

1. Install/start PostgreSQL locally; create a dedicated non-superuser login role and `webnote` database. Do not reuse the operating-system `web` user as a peer-authenticated PostgreSQL role: this host's Nginx workers run as `web` too.
2. Put `WEBNOTE_DATABASE_URL=postgres://webnote:<URL-escaped-password>@127.0.0.1:5432/webnote?sslmode=disable` in the backend source `.env` (mode `0600`). `sslmode=disable` is suitable only for a loopback connection. Add SMTP variables only if email registration is enabled: `WEBNOTE_SMTP_HOST`, `WEBNOTE_SMTP_PORT=587`, `WEBNOTE_SMTP_USER`, `WEBNOTE_SMTP_PASSWORD`. Keep all secrets out of Git.
3. The installer copies only the database URL and SMTP allowlist into `/opt/webnote/config/webnote.env` (root-owned, mode `0600`). The systemd service uses `LoadCredential`; the application reads it via `CREDENTIALS_DIRECTORY`. Do not replace this with `EnvironmentFile` or make the config group-readable.

## Staging under `/opt/webnote`

Review `deploy/install-local.sh`, then, from the parent workspace containing `webnote/` and `webnote-web/`, run the script explicitly with root privileges:

```sh
sudo sh webnote/deploy/install-local.sh
```

It checks for the low-privilege `web` account, PostgreSQL URL and build artifacts; refuses symlinks in protected installation paths; stages the binary, frontend, legacy Markdown fallback and images under `/opt/webnote`; sets the application data owner to `web`; installs `deploy/webnote.service`; and reloads the systemd unit definitions. It preserves an existing credential file. Because copy tools can retain source directory modes, the script explicitly resets `/opt/webnote/data`, `img` and `legacy` to owner-only access after copying. It does not import SQLite records, start/enable Webnote, edit Nginx, or reload Nginx. Inspect file ownership/modes after staging.

The service binds only `127.0.0.1:10167`, runs as `web`, uses `ProtectSystem=strict`, and may write only to `/opt/webnote/data`. Commands after staging, once the domain/TLS/site configuration is ready:

```sh
sudo systemctl enable --now webnote
systemctl status webnote
journalctl -u webnote -f
curl --fail http://127.0.0.1:10167/api/ping
```

## Logs and rotation

The Go process writes standard-library, GORM error, Gin access and Gin recovery logs to `/var/log/webnote/webnote.log`; it also mirrors them to stdout/journald. The file rolls at 30 MiB, gzip-compresses archives and retains at most 7 backups. SMTP verification failures are logged with the server/error but without the verification code, SMTP authorization code or recipient address.

The Webnote Nginx virtual host writes `/var/log/webnote/nginx-access.log` and `nginx-error.log`. `deploy/webnote.logrotate` rotates both at 30 MiB, compresses them, and retains 7 backups; `webnote-logrotate.timer` checks every five minutes and signals Nginx to reopen rotated logs. Inspect logs with:

```sh
sudo tail -F /var/log/webnote/webnote.log
sudo tail -F /var/log/webnote/nginx-error.log
sudo journalctl -u webnote -f
systemctl list-timers webnote-logrotate.timer
```

## Nginx

`deploy/webnote.conf` is the site configuration for `webnote.zanilia.eu.org`. It listens on 443 only (no listener/redirect on port 80), uses the existing `zanilia.eu.org` certificate confirmed by the user to cover this hostname, serves `/opt/webnote/dist`, proxies `/api/` to `127.0.0.1:10167`, rate-limits authentication routes, sets security headers and bounds uploads. Install it as `/etc/nginx/conf.d/webnote.conf`; run `sudo nginx -t` before `sudo systemctl reload nginx`. Do not change unrelated virtual hosts.

Nginx must set a fixed `Host` matching the deployed domain and overwrite `X-Forwarded-For`; the Go server trusts only local proxy addresses. CSP must include `frame-src 'self'` and `frame-ancestors 'self'` for the Muya editor's same-origin `/markdown-editor/index.html` iframe; do not use `frame-ancestors 'none'`. HTTPS is mandatory for production `Secure` cookies. An unset domain/certificate is a staging state, not a production-ready site.

## Backups and upgrades

Use `pg_dump -Fc -h 127.0.0.1 -U webnote webnote > webnote-YYYYMMDD.dump`; back up `/opt/webnote/data/img` separately. Retain legacy Markdown files until a successful migration and restore test. Restrict and encrypt off-host backups; periodically test restore into an isolated database and filesystem. The PostgreSQL role should have only the rights the application/schema migration needs, not superuser privileges.

For dependency upgrades, review release compatibility and security advisories, run `go mod tidy`, full PostgreSQL tests, `go vet`, a release build and frontend build/typecheck if that project changed. Back up before database schema or importer changes.

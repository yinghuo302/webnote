# Security model and incident response

## Login and registration

- Passwords use bcrypt. New registrations require 12–72 bytes; successful login upgrades a legacy plaintext password to bcrypt. Login errors are generic. The IP-based in-process throttles are defense in depth; the Nginx sample also rate-limits sign-in, sign-up and email-code endpoints. These limits are per process at the Go layer, so they are not a distributed rate limiter.
- Email codes are generated with `crypto/rand`, valid for ten minutes, persisted as bcrypt hashes, limited to five verification attempts, and throttled to one send per email per minute. The registration code is deleted on success. SMTP delivery is permitted only after STARTTLS; an unconfigured mail service cannot register new accounts.
- Requests with unsafe HTTP methods must carry an exact same-origin `Origin`; cross-site fetch metadata is also rejected. The code endpoint uses POST so a cross-site image/link GET cannot trigger a send. Keep frontend API calls same-origin and do not add permissive CORS.

## Session cookie and server-side state

At successful login/registration the backend generates 32 cryptographically random bytes and sends the 64-character hexadecimal token. The browser holds only this opaque bearer token: no user ID, email, password, or claims are encoded in it. PostgreSQL stores SHA-256(token), user ID and absolute expiry; authenticated requests look up that hash on the server.

Lifetime is 24 hours, fixed rather than sliding (`Max-Age=86400` plus `Expires`, checked against database `expires_at`). Production uses `__Host-webnote_session`, `Secure`, `HttpOnly`, `SameSite=Strict`, `Path=/`, and no `Domain`. When secure cookies are enabled the `__Host-` prefix is required by browsers. `WEBNOTE_SECURE_COOKIES` defaults to enabled; only a local HTTP development session may explicitly set it to `false`, which uses the unprefixed cookie name.

The old signed client-side `LoginSession` cookie is not accepted after the server-side session change. Existing users must sign in again. Session rows are revoked on logout; password changes delete all sessions for that user; `POST /api/private/logout-all` is the emergency all-device revocation endpoint. Expired rows are opportunistically pruned on session creation.

## If a cookie is stolen

A stolen, unexpired token can be replayed until it is revoked or expires. `HttpOnly` stops ordinary JavaScript from reading the cookie, but it does not stop an XSS payload from issuing authenticated requests in the victim's browser. `Secure` protects transport only when the site is HTTPS. `SameSite=Strict` and exact-Origin checks mitigate CSRF; neither is an XSS defense.

If the account owner still has access, use **Log out all devices** (`POST /api/private/logout-all`) or change the password; both delete server-side session records. An administrator can delete rows with that user's `user_id` from `sessions` to revoke them. Also rotate exposed database/SMTP credentials if host access is suspected, inspect access/system logs, and investigate the original theft path. The `LoginSession` value itself is never logged by application or sample Nginx access logging. Do not claim that a valid stolen token is intrinsically protected from replay.

## Deployment boundaries and sensitive values

The Go API is loopback-only; Nginx terminates TLS and forwards a fixed configured host and actual client IP. Go trusts only loopback proxy addresses. Never let arbitrary clients reach port 8000 or supply trusted proxy headers directly. The Linux `web` account is shared with this host's Nginx worker, so the PostgreSQL login is a separate password-authenticated role `webnote`, not peer-authenticated as OS user `web`.

Production database/SMTP values are kept in a root-owned mode-0600 credential source under `/opt/webnote/config` and passed through systemd `LoadCredential`, not `EnvironmentFile`; the app reads the private credential file from `CREDENTIALS_DIRECTORY`. Do not put secrets in the frontend root, container build context, command-line arguments, committed files, access logs, or agent output. Do not expose `.env`, SQLite database/WAL files or local backups to GitHub.

Uploads are size-limited and detected by file signature; only PNG/JPEG/GIF/WebP are accepted and filenames are server-generated. Keep request-size limits aligned between Nginx and Go. Public Markdown is untrusted; use safe DOM/text rendering and sanitize any future HTML-capable renderer.

These safeguards reduce risk but do not provide absolute security against compromised browsers, hosts, mailboxes, database operators, distributed attacks or software vulnerabilities. Patch Go/frontend dependencies and test recovery regularly.

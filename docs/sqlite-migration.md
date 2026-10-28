# Legacy SQLite to PostgreSQL migration

This is an offline, one-time import. Never delete the SQLite database, WAL/SHM, old `data/`, or `img/` until PostgreSQL records, Markdown content, images and restore procedures are verified.

## Preparation

1. Stop the old writer process. Make a consistent SQLite backup (SQLite online backup API or a clean stopped copy), preserve any relevant WAL/SHM, and separately back up `data/` and `img/`.
2. Provision PostgreSQL and a dedicated non-superuser `webnote` login/database. Use an empty target database; the importer refuses to add records to non-empty `users`, `logins`, or `files` tables.
3. Confirm Python 3 with the standard-library `sqlite3` module is installed. The Go application does not load SQLite; Python reads source data in read-only URI mode.
4. Verify each legacy note whose body is still on disk has `data/<uuid>.md`, and plan to restore old images to `/opt/webnote/data/img`.

## Read-only preflight

From the backend repository root:

```sh
go run ./cmd/import-sqlite --check --sqlite /path/to/legacy.db --data-dir /path/to/legacy-data
```

Preflight reports only user/login/note counts, normalized duplicate-email group count, and missing Markdown-file count. It does not connect to PostgreSQL or print user data. Resolve any duplicate accounts with the data owner before attempting import; do not guess, discard, or auto-rename accounts.

## Import and verification

Once preflight is clean, load `WEBNOTE_DATABASE_URL` securely into the importer process and import into the empty database:

```sh
set -a
. ./.env
set +a
go run ./cmd/import-sqlite --sqlite /path/to/legacy.db --data-dir /path/to/legacy-data
```

The importer checks again, rejects duplicate normalized emails or missing bodies, streams records, imports all users/logins/notes in one PostgreSQL transaction, marks note content as stored, and advances the users identity sequence. Any importer/exporter/constraint failure rolls back the transaction. Run this only against a dedicated local target; do not paste or log a real password.

Afterward, compare source and target row counts, verify note IDs/public flags and sample Markdown bodies, restore images, then exercise login, private access, sharing, PostgreSQL backup and restore. Existing legacy plaintext passwords are upgraded to bcrypt on their first successful login. Expired registration codes are not migrated. Cookie sessions are not migrated; users sign in again.

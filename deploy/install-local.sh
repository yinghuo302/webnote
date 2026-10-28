#!/bin/sh
# Run manually as root after reviewing the files and obtaining a valid TLS certificate.
set -eu
umask 077
if [ "$(id -u)" -ne 0 ]; then echo 'Run with sudo after reviewing this script.' >&2; exit 1; fi
base=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
frontend="$base/../webnote-web/dist"
[ -f "$base/bin/webnote" ] || { echo 'Build the Go binary first.' >&2; exit 1; }
[ -f "$frontend/index.html" ] || { echo 'Build the separate frontend first.' >&2; exit 1; }
[ -f "$base/.env" ] || { echo 'Prepare backend .env first.' >&2; exit 1; }
id web >/dev/null 2>&1 || { echo 'Low-privilege user web does not exist.' >&2; exit 1; }
grep -q '^WEBNOTE_DATABASE_URL=' "$base/.env" || { echo 'Add WEBNOTE_DATABASE_URL to the private source .env.' >&2; exit 1; }
[ ! -L /opt/webnote ] || { echo 'Refusing symlink at /opt/webnote.' >&2; exit 1; }
for path in /opt/webnote/bin /opt/webnote/config /opt/webnote/www /opt/webnote/data; do [ ! -L "$path" ] || { echo "Refusing symlink at $path." >&2; exit 1; }; done
[ ! -L /opt/webnote/config/webnote.env ] || { echo 'Refusing symlinked service credentials.' >&2; exit 1; }
install -d -o root -g web -m 0750 /opt/webnote /opt/webnote/bin /opt/webnote/www
install -d -o root -g root -m 0700 /opt/webnote/config
install -d -o web -g web -m 0700 /opt/webnote/data /opt/webnote/data/img /opt/webnote/data/legacy
if [ -d "$base/data" ]; then cp -an "$base/data/." /opt/webnote/data/legacy/; chown -R web:web /opt/webnote/data/legacy; fi
if [ -d "$base/img" ]; then cp -an "$base/img/." /opt/webnote/data/img/; chown -R web:web /opt/webnote/data/img; fi
install -o root -g web -m 0750 "$base/bin/webnote" /opt/webnote/bin/webnote
if [ ! -e /opt/webnote/config/webnote.env ]; then
  install -o root -g root -m 0600 /dev/null /opt/webnote/config/webnote.env
  awk '$0 ~ /^(WEBNOTE_DATABASE_URL|WEBNOTE_SMTP_HOST|WEBNOTE_SMTP_PORT|WEBNOTE_SMTP_USER|WEBNOTE_SMTP_PASSWORD)=/' "$base/.env" > /opt/webnote/config/webnote.env
  chown root:root /opt/webnote/config/webnote.env
  chmod 0600 /opt/webnote/config/webnote.env
else
  [ -f /opt/webnote/config/webnote.env ] || { echo 'Existing credential path is not a regular file.' >&2; exit 1; }
  echo 'Existing private configuration preserved: /opt/webnote/config/webnote.env' >&2
fi
chown root:root /opt/webnote/config/webnote.env
chmod 0600 /opt/webnote/config/webnote.env
cp -R "$frontend/." /opt/webnote/www/
chown -R root:web /opt/webnote/www
chmod -R u=rwX,g=rX,o= /opt/webnote/www
install -o root -g root -m 0644 "$base/deploy/webnote.service" /etc/systemd/system/webnote.service
systemctl daemon-reload
printf 'Staged /opt/webnote and systemd unit. No service or Nginx site was enabled.\n'

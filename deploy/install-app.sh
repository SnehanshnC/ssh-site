#!/usr/bin/env bash
#
# install-app.sh - the one thing a deploy credential (the personal admin key,
# or the narrowly-scoped ci-deploy key) is allowed to do as root: install a
# new app binary with a health-checked rollback path, or restore the last one
# that passed. Lives at /opt/ssh-site/bin/install-app.sh, root:root 0700.
# ci-deploy's entire sudo grant (deploy/ci-deploy.sudoers) is the two
# invocations below and nothing else - it can never read or edit this file.
#
#   install-app.sh install STAGING_DIR
#     STAGING_DIR holds ssh-site-linux-amd64 (required), and optionally
#     ssh-site.service and PACK_SHA. Whatever is currently live is snapshotted
#     as the "prev" generation before being replaced.
#
#   install-app.sh rollback
#     Restores the "prev" generation snapshotted by the last install and
#     restarts. Fails if there is nothing to roll back to.
#
# Neither action polls for reachability - that is scripts/deploy.sh's job,
# from off the box, using the same app_reachable check a visitor's client
# would trigger. This script only proves the service is active locally, as a
# fast fail-fast before that real check runs.

set -euo pipefail

[ "$(id -u)" -eq 0 ] || { echo "install-app.sh must run as root" >&2; exit 1; }

BIN=/opt/ssh-site/ssh-site
BIN_PREV=/opt/ssh-site/ssh-site.prev
UNIT=/etc/systemd/system/ssh-site.service
UNIT_PREV=/opt/ssh-site/ssh-site.service.prev
SHA=/opt/ssh-site/PACK_SHA
SHA_PREV=/opt/ssh-site/PACK_SHA.prev

restart_and_check() {
  systemctl daemon-reload
  systemctl restart ssh-site.service
  sleep 1
  if systemctl is-active --quiet ssh-site.service; then
    echo "$1"
  else
    systemctl status ssh-site.service --no-pager || true
    journalctl -u ssh-site.service --no-pager -n 40 || true
    exit 1
  fi
}

action="${1:-}"

case "$action" in
  install)
    dir="${2:-}"
    [ -n "$dir" ] || { echo "usage: install-app.sh install STAGING_DIR" >&2; exit 1; }
    [ -f "$dir/ssh-site-linux-amd64" ] || { echo "no ssh-site-linux-amd64 in $dir" >&2; exit 1; }

    # Snapshot whatever is live before it is overwritten. On a fresh box
    # there is nothing to snapshot yet, and that is fine - rollback simply
    # has nothing to restore until one successful install has happened.
    [ -f "$BIN" ] && cp -p "$BIN" "$BIN_PREV"
    [ -f "$UNIT" ] && cp -p "$UNIT" "$UNIT_PREV"
    [ -f "$SHA" ] && cp -p "$SHA" "$SHA_PREV"

    install -o root -g root -m 0755 -d /opt/ssh-site
    install -o root -g root -m 0755 "$dir/ssh-site-linux-amd64" "$BIN"
    [ -f "$dir/ssh-site.service" ] && install -o root -g root -m 0644 "$dir/ssh-site.service" "$UNIT"
    [ -f "$dir/PACK_SHA" ] && install -o root -g root -m 0644 "$dir/PACK_SHA" "$SHA"

    restart_and_check "ssh-site.service is active"
    ;;

  rollback)
    [ -f "$BIN_PREV" ] || { echo "no previous binary to roll back to" >&2; exit 1; }
    install -o root -g root -m 0755 "$BIN_PREV" "$BIN"
    [ -f "$UNIT_PREV" ] && install -o root -g root -m 0644 "$UNIT_PREV" "$UNIT"
    [ -f "$SHA_PREV" ] && install -o root -g root -m 0644 "$SHA_PREV" "$SHA"

    restart_and_check "rolled back and ssh-site.service is active"
    ;;

  *)
    echo "usage: install-app.sh install STAGING_DIR | install-app.sh rollback" >&2
    exit 1
    ;;
esac

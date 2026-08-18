#!/usr/bin/env bash
#
# deploy.sh - harden the box slice 11 built and put the app server on port
# 22. Non-interactive: everything here is a machine step, so unlike
# provision-box.sh there is nothing to pause for.
#
#   bash scripts/deploy.sh          # everything: harden + ship the app
#   bash scripts/deploy.sh app      # just rebuild, upload and restart the
#                                    app - what a content-pack push re-runs
#
# Facts come from .scratch/ssh-site/box.env (KEY=value lines - read with
# grep/cut below, never sourced, matching provision-box.sh's own convention).
# Override its location with ENV_FILE=... if it lives elsewhere.
#
# The risky part is nftables: a chain with `policy drop` can lock the box up
# exactly like slice 11's port move could, so it is sequenced the same way -
# arm a dead man's switch on the box, apply, prove a brand new session still
# opens, only then cancel the switch. sshd's own hardening reload gets the
# same treatment, cheap insurance even though PubkeyAuthentication never
# changes. See scripts/provision-box.sh for the pattern this borrows.

set -euo pipefail
cd "$(dirname "$0")/.."

MODE="${1:-full}"
case "$MODE" in
  full|app) ;;
  *) echo "usage: $0 [full|app]" >&2; exit 1 ;;
esac

ENV_FILE="${ENV_FILE:-.scratch/ssh-site/box.env}"
[ -f "$ENV_FILE" ] || { echo "✗ no $ENV_FILE - run scripts/provision-box.sh first" >&2; exit 1; }

log()  { printf '\n▸ %s\n' "$1"; }
ok()   { printf '  ✓ %s\n' "$1"; }
warn() { printf '  ⚠ %s\n' "$1" >&2; }
die()  { printf '  ✗ %s\n' "$1" >&2; exit 1; }

# env_val KEY - a value out of $ENV_FILE, read the way box.env documents
# itself as wanting to be read: grep the line, cut past the first '='. Never
# sourced - values are free-form and this file is not trusted as shell.
env_val() {
  local line
  line=$(grep -E "^${1}=" "$ENV_FILE" | tail -n1) || true
  printf '%s' "${line#*=}"
}

ADMIN_KEY="$(env_val ADMIN_KEY)"
BOX_IP="$(env_val BOX_IP)"
ADMIN_PORT="$(env_val ADMIN_PORT)"
REMOTE_USER="$(env_val REMOTE_USER)"
ADDRESS="$(env_val ADDRESS)"

for v in ADMIN_KEY BOX_IP ADMIN_PORT REMOTE_USER; do
  [ -n "${!v}" ] || die "$ENV_FILE has no $v - re-run scripts/provision-box.sh"
done
[ -n "$ADDRESS" ] || ADDRESS="$BOX_IP"

SSH_BASE=(-o PasswordAuthentication=no -o KbdInteractiveAuthentication=no
          -o NumberOfPasswordPrompts=0 -o StrictHostKeyChecking=accept-new
          -o ConnectTimeout=15)

# box CMD... - run a command on the box as the admin user, no sudo.
box() { ssh -n "${SSH_BASE[@]}" -i "$ADMIN_KEY" -p "$ADMIN_PORT" "$REMOTE_USER@$BOX_IP" "$@"; }

# remote <<'EOS' - run the script on stdin as root on the box.
remote() { ssh "${SSH_BASE[@]}" -i "$ADMIN_KEY" -p "$ADMIN_PORT" "$REMOTE_USER@$BOX_IP" "sudo bash -s"; }

box_ok() { box true >/dev/null 2>&1; }

# wait_for_admin TRIES - poll until a fresh session opens on the admin port.
wait_for_admin() {
  local tries="$1" i=1
  while [ "$i" -le "$tries" ]; do
    if box_ok; then return 0; fi
    printf '    · waiting for %s:%s (%s/%s)\n' "$BOX_IP" "$ADMIN_PORT" "$i" "$tries"
    sleep 5
    i=$((i + 1))
  done
  return 1
}

# app_reachable TRIES - poll until the app server on port 22 answers a real,
# anonymous, no-PTY session with the plain-text document (D2) and exits 0.
# No admin key involved: this is exactly what a visitor's client does, and
# the app takes no credentials from anyone.
app_reachable() {
  local tries="$1" i=1
  while [ "$i" -le "$tries" ]; do
    if ssh -n -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
         -o ConnectTimeout=8 -p 22 "$BOX_IP" true >/dev/null 2>&1; then
      return 0
    fi
    printf '    · waiting for %s:22 (%s/%s)\n' "$BOX_IP" "$i" "$tries"
    sleep 5
    i=$((i + 1))
  done
  return 1
}

BUILD_OUT="bin/ssh-site-linux-amd64"
STAGE_DIR="$(mktemp -d)"
trap 'rm -rf "$STAGE_DIR"' EXIT

# ── build ────────────────────────────────────────────────────────────────
log "Fetching the content pack and cross-compiling for linux/amd64"
sh scripts/fetch-pack.sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
  -o "$BUILD_OUT" ./cmd/ssh-site
ok "built $BUILD_OUT ($(du -h "$BUILD_OUT" | cut -f1))"

# ── stage templated configs ─────────────────────────────────────────────
# The admin port and login user are private facts (box.env, gitignored), so
# the checked-in templates carry placeholders rather than the real values.
log "Templating deploy configs with this box's own facts"
sed "s/@ADMIN_PORT@/$ADMIN_PORT/g" deploy/nftables.conf > "$STAGE_DIR/nftables.conf"
sed "s/@ADMIN_PORT@/$ADMIN_PORT/g" deploy/fail2ban-jail.conf > "$STAGE_DIR/fail2ban-jail.conf"
sed "s/@ADMIN_USER@/$REMOTE_USER/g" deploy/sshd-hardening.conf > "$STAGE_DIR/sshd-hardening.conf"
cp deploy/ssh-site.service "$STAGE_DIR/ssh-site.service"
ok "staged configs in $STAGE_DIR"

# ── upload ───────────────────────────────────────────────────────────────
log "Uploading the binary and configs"
box 'mkdir -p /tmp/ssh-site-deploy'
scp -o PasswordAuthentication=no -o StrictHostKeyChecking=accept-new -o ConnectTimeout=15 \
  -i "$ADMIN_KEY" -P "$ADMIN_PORT" \
  "$BUILD_OUT" "$STAGE_DIR/nftables.conf" "$STAGE_DIR/fail2ban-jail.conf" \
  "$STAGE_DIR/sshd-hardening.conf" "$STAGE_DIR/ssh-site.service" \
  "$REMOTE_USER@$BOX_IP:/tmp/ssh-site-deploy/" >/dev/null
ok "uploaded to /tmp/ssh-site-deploy on the box"

# ── prerequisite packages ───────────────────────────────────────────────
log "Installing prerequisite packages (idempotent)"
remote <<'EOS'
set -eu
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq nftables fail2ban unattended-upgrades >/dev/null
echo "packages present: nftables, fail2ban, unattended-upgrades"
EOS

if [ "$MODE" = "full" ]; then
  # ── sshd hardening, with its own dead man's switch ────────────────────
  # PubkeyAuthentication never changes and the admin key already works
  # without a password, so this is lower risk than the port move it borrows
  # the pattern from - but MaxStartups/AllowUsers/PerSourceMaxStartups are
  # new, and a restart of the daemon is still a restart of the daemon.
  log "Hardening the admin sshd"
  remote <<'EOS'
set -eu
cfg=/etc/ssh/sshd_config.d/20-ssh-site-hardening.conf
staged=/tmp/ssh-site-deploy/sshd-hardening.conf
backup=/root/ssh-site-sshd-hardening-backup.conf

if [ -f "$cfg" ] && cmp -s "$cfg" "$staged"; then
    echo "sshd hardening already applied - skipping the restart"
    exit 0
fi

if [ -f "$cfg" ]; then cp "$cfg" "$backup"; else rm -f "$backup"; fi
cp "$staged" "$cfg"
chmod 644 "$cfg"
if ! sshd -t; then
    echo "sshd -t rejected the new config; reverting" >&2
    if [ -f "$backup" ]; then cp "$backup" "$cfg"; else rm -f "$cfg"; fi
    exit 1
fi

systemctl reset-failed ssh-site-sshd-rollback.timer ssh-site-sshd-rollback.service 2>/dev/null || true
systemd-run --collect --unit=ssh-site-sshd-rollback --on-active=10min /bin/sh -c \
    "if [ -f $backup ]; then cp $backup $cfg; else rm -f $cfg; fi; systemctl restart ssh.service" >/dev/null
echo "rollback armed"

systemctl restart ssh.service
echo "sshd restarted with the hardened config"
EOS
  wait_for_admin 12 || die "admin port did not come back after the sshd hardening restart; the box rolls itself back within 10 minutes regardless"
  ok "a brand new session opened on the admin port"
  remote <<'EOS' || true
systemctl stop ssh-site-sshd-rollback.timer 2>/dev/null || true
systemctl reset-failed ssh-site-sshd-rollback.service ssh-site-sshd-rollback.timer 2>/dev/null || true
echo "sshd rollback cancelled"
EOS
fi

# ── dedicated app user ──────────────────────────────────────────────────
log "Creating the dedicated app user (idempotent)"
remote <<'EOS'
set -eu
if ! id ssh-site >/dev/null 2>&1; then
    useradd --system --no-create-home --home-dir /var/lib/ssh-site --shell /usr/sbin/nologin ssh-site
    echo "created the ssh-site system user"
else
    echo "ssh-site user already exists"
fi
mkdir -p /var/lib/ssh-site
chown ssh-site:ssh-site /var/lib/ssh-site
chmod 750 /var/lib/ssh-site
EOS

# ── host key: generated once, backed up (D5) ────────────────────────────
log "Generating the host key if it doesn't exist yet"
HOSTKEY_OUT="$(remote <<'EOS'
set -eu
key=/var/lib/ssh-site/host_ed25519
backupdir=/var/backups/ssh-site
if [ -f "$key" ]; then
    echo "host key already exists - leaving it alone, a redeploy never regenerates it"
else
    ssh-keygen -t ed25519 -f "$key" -N "" -C "ssh-site host key" -q
    chown ssh-site:ssh-site "$key" "$key.pub"
    chmod 600 "$key"
    chmod 644 "$key.pub"
    echo "generated the host key"
fi
mkdir -p "$backupdir"
cp "$key" "$key.pub" "$backupdir/"
chmod 700 "$backupdir"
chmod 600 "$backupdir/host_ed25519"

# /var/lib/ssh-site is 750 owned by ssh-site, and the key inside it is 600 -
# the unprivileged deploy user (REMOTE_USER, over the admin sshd) can't read
# either, only root can. Stage a copy it can reach so the plain scp below
# doesn't need root, then delete the stage - never leave the private half
# sitting somewhere the admin user could always get back to.
stage=/tmp/ssh-site-deploy
cp "$key" "$key.pub" "$stage/"
chmod 644 "$stage/host_ed25519" "$stage/host_ed25519.pub"

ssh-keygen -lf "$key.pub"
EOS
)"
echo "$HOSTKEY_OUT" | sed 's/^/  /'
HOSTKEY_FINGERPRINT="$(echo "$HOSTKEY_OUT" | tail -n1)"

# An on-box copy survives an accidental delete; it does not survive the box
# itself being destroyed, which box.md documents as the actual recovery plan
# ("delete the instance and run the wizard again"). Pull a copy to the
# private tracker too, so the key - and the fingerprint published in the
# README - can survive that.
BACKUP_DIR=".scratch/ssh-site/host-key-backup"
mkdir -p "$BACKUP_DIR"
scp -o PasswordAuthentication=no -o StrictHostKeyChecking=accept-new -o ConnectTimeout=15 \
  -i "$ADMIN_KEY" -P "$ADMIN_PORT" \
  "$REMOTE_USER@$BOX_IP:/tmp/ssh-site-deploy/host_ed25519" \
  "$REMOTE_USER@$BOX_IP:/tmp/ssh-site-deploy/host_ed25519.pub" \
  "$BACKUP_DIR/" >/dev/null
chmod 600 "$BACKUP_DIR/host_ed25519"
remote <<'EOS' >/dev/null || true
shred -u /tmp/ssh-site-deploy/host_ed25519 /tmp/ssh-site-deploy/host_ed25519.pub 2>/dev/null || rm -f /tmp/ssh-site-deploy/host_ed25519 /tmp/ssh-site-deploy/host_ed25519.pub
EOS
ok "host key backed up off-box to $BACKUP_DIR"

# ── install the binary and the systemd unit ─────────────────────────────
log "Installing the binary and the systemd unit"
remote <<'EOS'
set -eu
install -o root -g root -m 0755 -d /opt/ssh-site
install -o root -g root -m 0755 /tmp/ssh-site-deploy/ssh-site-linux-amd64 /opt/ssh-site/ssh-site
install -o root -g root -m 0644 /tmp/ssh-site-deploy/ssh-site.service /etc/systemd/system/ssh-site.service
systemctl daemon-reload
systemctl enable ssh-site.service
systemctl restart ssh-site.service
sleep 1
if ! systemctl is-active --quiet ssh-site.service; then
    systemctl status ssh-site.service --no-pager || true
    journalctl -u ssh-site.service --no-pager -n 40 || true
    exit 1
fi
echo "ssh-site.service is active"
ss -ltnp "sport = :22" || true
EOS
app_reachable 12 || die "port 22 did not answer after the app was installed"
ok "the app server answers on port 22"

if [ "$MODE" = "full" ]; then
  # ── nftables, with its own dead man's switch ──────────────────────────
  # The riskiest step, saved for last so its "prove" step is the real thing:
  # both the admin port and the now-deployed app on 22, through the new
  # policy-drop chain, not a proxy for it.
  log "Applying nftables (replaces the image's iptables ruleset)"
  remote <<'EOS'
set -eu
staged=/tmp/ssh-site-deploy/nftables.conf
live=/etc/nftables.conf

if [ -f "$live" ] && cmp -s "$live" "$staged" && systemctl is-active --quiet nftables.service; then
    echo "nftables ruleset already applied - skipping the reapply"
    exit 0
fi

nft -c -f "$staged"

iptbackup=/root/ssh-site-iptables-backup.v4
ip6backup=/root/ssh-site-ip6tables-backup.v6
iptables-save > "$iptbackup"
ip6tables-save > "$ip6backup" 2>/dev/null || true

systemctl reset-failed ssh-site-nft-rollback.timer ssh-site-nft-rollback.service 2>/dev/null || true
systemd-run --collect --unit=ssh-site-nft-rollback --on-active=10min /bin/sh -c \
    "nft flush ruleset; iptables-restore < $iptbackup; ip6tables-restore < $ip6backup 2>/dev/null || true; systemctl disable --now nftables.service 2>/dev/null || true; systemctl enable --now netfilter-persistent" >/dev/null
echo "nftables rollback armed"

# Both iptables and nftables sit on the same netfilter core on this image
# (iptables here is the nf_tables backend) - flush the old ruleset
# deliberately rather than let the new one merely sit alongside it.
iptables -F
iptables -X
iptables -P INPUT ACCEPT
iptables -P FORWARD ACCEPT
iptables -P OUTPUT ACCEPT
ip6tables -F 2>/dev/null || true
ip6tables -X 2>/dev/null || true
ip6tables -P INPUT ACCEPT 2>/dev/null || true
ip6tables -P FORWARD ACCEPT 2>/dev/null || true
ip6tables -P OUTPUT ACCEPT 2>/dev/null || true

# So a reboot can't bring the old REJECT-ending ruleset back behind our
# backs (that saved ruleset predates this ruleset and would fight it).
systemctl disable --now netfilter-persistent 2>/dev/null || true

cp "$staged" "$live"
nft -f "$live"
systemctl enable --now nftables.service
echo "nftables applied"
EOS
  wait_for_admin 12 || die "admin port did not come back after applying nftables; the box rolls itself back within 10 minutes regardless"
  ok "a brand new session opened on the admin port"
  app_reachable 12 || die "port 22 did not answer after applying nftables; the box rolls itself back within 10 minutes regardless"
  ok "the app server still answers on port 22"
  remote <<'EOS' || true
systemctl stop ssh-site-nft-rollback.timer 2>/dev/null || true
systemctl reset-failed ssh-site-nft-rollback.service ssh-site-nft-rollback.timer 2>/dev/null || true
echo "nftables rollback cancelled"
EOS

  # ── fail2ban: the admin sshd only ─────────────────────────────────────
  log "Configuring fail2ban for the admin sshd only"
  remote <<'EOS'
set -eu
staged=/tmp/ssh-site-deploy/fail2ban-jail.conf
live=/etc/fail2ban/jail.d/ssh-site-admin.conf
if [ -f "$live" ] && cmp -s "$live" "$staged"; then
    echo "fail2ban jail already configured"
else
    cp "$staged" "$live"
    chmod 644 "$live"
    echo "fail2ban jail written"
fi
systemctl enable --now fail2ban.service
systemctl restart fail2ban.service
sleep 1
fail2ban-client status sshd 2>&1 || true
EOS

  # ── unattended upgrades ────────────────────────────────────────────────
  log "Verifying unattended-upgrades"
  remote <<'EOS'
set -eu
f=/etc/apt/apt.conf.d/20auto-upgrades
if grep -q 'APT::Periodic::Update-Package-Lists "1"' "$f" 2>/dev/null \
   && grep -q 'APT::Periodic::Unattended-Upgrade "1"' "$f" 2>/dev/null; then
    echo "unattended-upgrades already enabled: $f"
else
    printf 'APT::Periodic::Update-Package-Lists "1";\nAPT::Periodic::Unattended-Upgrade "1";\n' > "$f"
    echo "wrote $f"
fi
systemctl is-enabled apt-daily.timer apt-daily-upgrade.timer 2>&1 || true
EOS
fi

# ── record ───────────────────────────────────────────────────────────────
RECORD=".scratch/ssh-site/deploy.md"
{
  echo "# Deploy record"
  echo
  echo "Written by scripts/deploy.sh ($MODE) on $(date -u '+%Y-%m-%d %H:%M UTC' 2>/dev/null || true)."
  echo "Private, like the rest of .scratch/."
  echo
  echo "| Fact | Value |"
  echo "| --- | --- |"
  echo "| Address | ssh $ADDRESS |"
  echo "| App server | port 22, user ssh-site, /opt/ssh-site/ssh-site |"
  echo "| Host key | /var/lib/ssh-site/host_ed25519, backed up to /var/backups/ssh-site and $BACKUP_DIR |"
  echo "| Host key fingerprint | $HOSTKEY_FINGERPRINT |"
} > "$RECORD"
ok "wrote $RECORD"

log "Done ($MODE)"
printf '  %s\n' "$HOSTKEY_FINGERPRINT"

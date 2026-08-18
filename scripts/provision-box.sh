#!/usr/bin/env bash
#
# provision-box.sh - builds the machine `ssh snehanshn.duckdns.org` answers on,
# from an empty cloud account to an admin login on a high port with port 22 left
# free for the app server. Run it to build the box the first time, and run it
# again to rebuild the box from scratch after that.
#
# It walks you through the console steps only you can take - creating the
# account, entering payment details, accepting terms, signing in to DuckDNS -
# and does every machine step itself. It never asks for a credential it could
# read, and the DuckDNS token is read from the environment or typed hidden,
# never written down by this script.
#
# What it leaves behind, for slice 12 to deploy onto: an Ubuntu box reachable
# by key on a high port, nothing listening on 22, and a record of the facts at
# .scratch/ssh-site/box.env plus a readable box.md beside it. That directory is
# gitignored - none of it is ever committed to this public repo. Override the
# location with ENV_FILE=... if the tracker lives elsewhere.
#
#   bash scripts/provision-box.sh
#
# Everything above the "STAGES" marker is the wizard library from the /wizard
# skill, verbatim except that its em dashes are plain dashes to match the rest
# of this repo: do not hand-edit it. The stages below the marker are this
# script's own.

set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────
# Wizard library - delightful, consistent UX. Identical across every wizard.
# ──────────────────────────────────────────────────────────────────────────

if [[ -t 1 ]] && command -v tput >/dev/null 2>&1 && [[ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]]; then
  BOLD=$(tput bold); DIM=$(tput dim); RESET=$(tput sgr0)
  BLUE=$(tput setaf 4); GREEN=$(tput setaf 2); YELLOW=$(tput setaf 3); RED=$(tput setaf 1)
else
  BOLD=""; DIM=""; RESET=""; BLUE=""; GREEN=""; YELLOW=""; RED=""
fi

# Author sets this at the top of the stages section.
TOTAL_STAGES=0

_STAGE_INDEX=0
ENV_FILE="${ENV_FILE:-.env}"
WRITTEN_ENV=()    # KEYs written to ENV_FILE this run
WRITTEN_SECRET=() # secret NAMEs set this run
SKIPPED=()        # things we couldn't do (e.g. gh missing)

# _clear - wipe the terminal so only the current step is on screen. No-op when
# output isn't a terminal, so piped logs stay readable.
_clear() {
  [[ -t 1 ]] || return 0
  if command -v tput >/dev/null 2>&1; then tput clear; else printf '\033[2J\033[3J\033[H'; fi
}

# banner "Title" - opening frame: what this wizard does.
banner() {
  _clear
  printf '\n%s%s  %s%s\n' "$BOLD" "$BLUE" "$1" "$RESET"
  printf '%s  %s stages%s\n\n' "$DIM" "$TOTAL_STAGES" "$RESET"
  printf '%s  You drive the browser; this wizard tells you exactly what to do and\n' "$DIM"
  printf '  captures the values you copy back. Stop any time with Ctrl-C and re-run\n'
  printf '  later - it remembers values already saved.%s\n' "$RESET"
  pause "Ready to start?"
}

# stage "Name" - clear the screen, then announce a stage and show progress.
# Clearing keeps only the current step on screen.
stage() {
  _clear
  _STAGE_INDEX=$((_STAGE_INDEX + 1))
  printf '\n%s%s▸ Stage %s/%s · %s%s\n' \
    "$BOLD" "$BLUE" "$_STAGE_INDEX" "$TOTAL_STAGES" "$1" "$RESET"
}

# say "..." - a plain instruction line.
say()  { printf '  %s\n' "$1"; }
# step "..." - a numbered-feeling action the human takes in the browser.
step() { printf '  %s•%s %s\n' "$BLUE" "$RESET" "$1"; }
note() { printf '  %s%s%s\n' "$DIM" "$1" "$RESET"; }
warn() { printf '  %s⚠ %s%s\n' "$YELLOW" "$1" "$RESET"; }

# open_url URL - open in the human's browser, cross-platform incl. WSL.
open_url() {
  local url="$1"
  printf '  %s↗ opening%s %s\n' "$GREEN" "$RESET" "$url"
  { if   command -v wslview     >/dev/null 2>&1; then wslview "$url"
    elif command -v explorer.exe >/dev/null 2>&1; then explorer.exe "$url"
    elif command -v xdg-open    >/dev/null 2>&1; then xdg-open "$url"
    elif command -v open        >/dev/null 2>&1; then open "$url"
    else warn "couldn't open a browser - visit it manually: $url"; fi
  } >/dev/null 2>&1 || warn "couldn't open a browser - visit it manually: $url"
}

# pause "msg" - wait for the human to confirm they've done the manual part.
pause() {
  printf '  %s%s%s ' "$DIM" "${1:-Press Enter to continue}" "$RESET"
  read -r _ || true
}

# confirm "question" - y/N gate; returns success on yes.
confirm() {
  local reply=""
  printf '  %s? %s [y/N] ' "$YELLOW" "$1"
  read -r reply || true
  [[ "$reply" =~ ^[Yy] ]]
}

# _existing KEY - current value of KEY in ENV_FILE, if any.
_existing() {
  [[ -f "$ENV_FILE" ]] || return 1
  local line; line=$(grep -E "^${1}=" "$ENV_FILE" | tail -n1) || return 1
  printf '%s' "${line#*=}"
}

# ask KEY "Prompt" - read a value into $KEY. Offers the existing .env value as
# a default on re-runs (Enter keeps it). Visible input (non-secret).
ask() {
  local key="$1" prompt="$2" current input
  current=$(_existing "$key" || true)
  if [[ -n "$current" ]]; then
    printf '  %s%s%s %s[Enter keeps current]%s ' "$BOLD" "$prompt" "$RESET" "$DIM" "$RESET"
  else
    printf '  %s%s%s ' "$BOLD" "$prompt" "$RESET"
  fi
  read -r input || true
  [[ -z "$input" && -n "$current" ]] && input="$current"
  printf -v "$key" '%s' "$input"
}

# ask_secret KEY "Prompt" - like ask, but input is hidden.
ask_secret() {
  local key="$1" prompt="$2" current input
  current=$(_existing "$key" || true)
  if [[ -n "$current" ]]; then
    printf '  %s%s%s %s[Enter keeps current]%s ' "$BOLD" "$prompt" "$RESET" "$DIM" "$RESET"
  else
    printf '  %s%s%s ' "$BOLD" "$prompt" "$RESET"
  fi
  read -rs input || true
  printf '\n'
  [[ -z "$input" && -n "$current" ]] && input="$current"
  printf -v "$key" '%s' "$input"
}

# write_env KEY VALUE - upsert KEY=VALUE into ENV_FILE (creates it; replaces
# any existing line). Idempotent.
write_env() {
  local key="$1" value="$2" tmp
  touch "$ENV_FILE"
  tmp=$(mktemp)
  grep -vE "^${key}=" "$ENV_FILE" > "$tmp" || true
  printf '%s=%s\n' "$key" "$value" >> "$tmp"
  mv "$tmp" "$ENV_FILE"
  WRITTEN_ENV+=("$key")
  printf '  %s✓ wrote%s %s → %s\n' "$GREEN" "$RESET" "$key" "$ENV_FILE"
}

# set_secret NAME VALUE - set a GitHub Actions repo secret via gh. Falls back
# to a warning (and records it) if gh is unavailable or unauthenticated.
set_secret() {
  local name="$1" value="$2"
  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    if printf '%s' "$value" | gh secret set "$name" >/dev/null 2>&1; then
      WRITTEN_SECRET+=("$name")
      printf '  %s✓ set%s GitHub secret %s\n' "$GREEN" "$RESET" "$name"
      return
    fi
  fi
  SKIPPED+=("GitHub secret $name (set it manually: gh secret set $name)")
  warn "skipped GitHub secret $name - gh not ready; set it later"
}

# set_var NAME VALUE - set a GitHub Actions repo variable (non-secret).
set_var() {
  local name="$1" value="$2"
  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    if gh variable set "$name" --body "$value" >/dev/null 2>&1; then
      printf '  %s✓ set%s GitHub variable %s\n' "$GREEN" "$RESET" "$name"
      return
    fi
  fi
  SKIPPED+=("GitHub variable $name")
  warn "skipped GitHub variable $name - gh not ready; set it later"
}

# finish - clear, then a closing summary of everything configured.
finish() {
  _clear
  printf '\n%s%s  ✓ Setup complete%s\n' "$BOLD" "$GREEN" "$RESET"
  (( ${#WRITTEN_ENV[@]} ))    && note "wrote ${#WRITTEN_ENV[@]} value(s) to $ENV_FILE: ${WRITTEN_ENV[*]}"
  (( ${#WRITTEN_SECRET[@]} )) && note "set ${#WRITTEN_SECRET[@]} GitHub secret(s): ${WRITTEN_SECRET[*]}"
  if (( ${#SKIPPED[@]} )); then
    printf '\n'; warn "still to do by hand:"
    for s in "${SKIPPED[@]}"; do note "  - $s"; done
  fi
  printf '\n'
}

# ──────────────────────────────────────────────────────────────────────────
# STAGES - the box, from an empty account to an admin login on a high port.
# ──────────────────────────────────────────────────────────────────────────

TOTAL_STAGES=11

cd "$(dirname "$0")/.." || exit 1

# The record lands in the private tracker, never in this public repo, because
# `.scratch/` is gitignored here. It holds facts, never secrets: where the
# DuckDNS token lives is recorded, the token itself never is.
if [ "$ENV_FILE" = ".env" ]; then ENV_FILE=".scratch/ssh-site/box.env"; fi
RECORD_DIR="$(dirname "$ENV_FILE")"
RECORD_MD="$RECORD_DIR/box.md"
mkdir -p "$RECORD_DIR"

# ── helpers ───────────────────────────────────────────────────────────────

ok()   { printf '  %s✓%s %s\n' "$GREEN" "$RESET" "$1"; }
fail() { printf '  %s✗ %s%s\n' "$RED" "$1" "$RESET"; }

# die "..." - stop with an explanation. Nothing is lost: every value captured
# so far is already in the record, and a re-run offers it back as the default.
die() {
  printf '\n'; fail "$1"
  note "Values captured so far are in $ENV_FILE; re-running picks up from them."
  exit 1
}

need() { command -v "$1" >/dev/null 2>&1 || die "this wizard needs '$1' on your PATH"; }

# rung_header "Name" - a clean screen for the next rung of the capacity ladder
# without consuming a stage number. Same frame the library's stage() draws.
rung_header() {
  _clear
  printf '\n%s%s▸ Stage %s/%s · %s%s\n' \
    "$BOLD" "$BLUE" "$_STAGE_INDEX" "$TOTAL_STAGES" "$1" "$RESET"
}

# expand_tilde PATH - a leading ~ typed at a prompt is literal, not $HOME.
expand_tilde() { printf '%s' "${1/#\~/$HOME}"; }

# ask_default KEY "Prompt" DEFAULT - the library's ask(), plus a default for
# the first run. It suggests DEFAULT only when the record has nothing to offer,
# so a re-run shows one hint rather than two that disagree.
ask_default() {
  local key="$1" prompt="$2" default="$3"
  if [ -n "$(_existing "$key" || true)" ]; then
    ask "$key" "$prompt"
  else
    ask "$key" "$prompt [Enter for $default]"
  fi
  if [ -z "${!key}" ]; then printf -v "$key" '%s' "$default"; fi
}

SSH_BASE=(-o PasswordAuthentication=no -o KbdInteractiveAuthentication=no
          -o NumberOfPasswordPrompts=0 -o StrictHostKeyChecking=accept-new
          -o ConnectTimeout=15)

# box PORT CMD... - run a command on the box. -n so ssh never eats the
# keystrokes a later prompt is waiting for.
box() {
  local port="$1"; shift
  ssh -n "${SSH_BASE[@]}" -i "$ADMIN_KEY" -p "$port" "$REMOTE_USER@$BOX_IP" "$@"
}

# remote PORT ARG... <<'EOS' - run the script on stdin as root on the box, with
# ARGs as its positional parameters. No -n here: stdin is the script.
remote() {
  local port="$1"; shift
  ssh "${SSH_BASE[@]}" -i "$ADMIN_KEY" -p "$port" "$REMOTE_USER@$BOX_IP" \
    "sudo bash -s -- $*"
}

box_ok() { box "$1" true >/dev/null 2>&1; }

# wait_for_box PORT TRIES - poll until a fresh session opens on PORT.
wait_for_box() {
  local port="$1" tries="$2" i=1
  while [ "$i" -le "$tries" ]; do
    if box_ok "$port"; then return 0; fi
    printf '  %s· waiting for %s:%s (%s/%s)%s\n' \
      "$DIM" "$BOX_IP" "$port" "$i" "$tries" "$RESET"
    sleep 5
    i=$((i + 1))
  done
  return 1
}

resolve_a() {
  if command -v dig >/dev/null 2>&1; then
    dig +short "$1" A 2>/dev/null | grep -E '^[0-9.]+$' | head -1 || true
  elif command -v host >/dev/null 2>&1; then
    host -t A "$1" 2>/dev/null | awk '/has address/ {print $4; exit}' || true
  fi
}

# ── the capacity ladder, one function per rung ────────────────────────────
#
# A1.Flex in Ashburn answers "Out of host capacity" often enough that waiting
# for it is not a plan. Each rung is a whole box that works; the one taken is
# recorded, because slice 12 builds for whatever architecture it lands on.

attempt_oracle_a1() {
  rung_header "Create the instance - rung 1 of 3: Oracle A1.Flex"
  say "The shape the spec asks for: 1 OCPU and 6 GB, Always Free, in Ashburn."
  open_url "https://cloud.oracle.com/compute/instances"
  step "Check the region selector at the top right reads US East (Ashburn)."
  step "Click Create instance."
  step "Name it ssh-site."
  step "Placement: leave the default availability domain for now."
  step "Image and shape: Change image, pick Canonical Ubuntu 24.04."
  step "Then Change shape: Virtual machine, Ampere, VM.Standard.A1.Flex,"
  say  "  1 OCPU and 6 GB of memory."
  step "Check the shape card says Always Free eligible before going on."
  step "Networking: create a new VCN with its defaults, in a public subnet,"
  say  "  and leave Assign a public IPv4 address on."
  step "Add SSH keys: Paste public keys, and paste the key from stage 2."
  step "Boot volume: leave the default size."
  step "Click Create."
  note "If the console has moved things: https://docs.oracle.com/en-us/iaas/Content/Compute/Tasks/launchinginstance.htm"
  printf '\n'
  say "If it refuses with 'Out of host capacity' or 'Out of capacity for shape':"
  step "try the other two availability domains, once each"
  step "then stop and answer no below. Rung 2 is a working box, not a consolation."
}

attempt_oracle_e2() {
  rung_header "Create the instance - rung 2 of 3: Oracle E2.1.Micro"
  say "Same account, same console, same Always Free tier. Two fields change."
  open_url "https://cloud.oracle.com/compute/instances"
  step "Create instance, named ssh-site, as before."
  step "Change shape: VM.Standard.E2.1.Micro. It sits under AMD, or under"
  say  "  Specialty and previous generation if it is not there."
  step "Change image: Canonical Ubuntu 24.04, the x86_64 build."
  step "Everything else is identical: new VCN with defaults, public IPv4 on,"
  say  "  paste the same public key, default boot volume, Create."
  printf '\n'
  note "1/8 OCPU and 1 GB of memory rather than 1 OCPU and 6 GB. A text TUI"
  note "does not notice. It is x86_64 rather than arm64, which is the one thing"
  note "downstream cares about, and this wizard records it."
}

attempt_gcp() {
  rung_header "Create the instance - rung 3 of 3: Google e2-micro"
  warn "Plan B, and it comes with a warning worth reading before you click."
  step "The free e2-micro needs a billing account with a card attached, and"
  say  "  anything outside the free limits bills for real."
  step "Google's free-tier page lists the instance, 30 GB of standard disk and"
  say  "  1 GB of North America egress a month. It does not say the VM's"
  say  "  external IPv4 address is included, and since February 2024 external"
  say  "  IPv4 on standard VMs is charged. Check today's pricing and your own"
  say  "  billing report before relying on this rung: \$0/month is a hard"
  say  "  requirement, not a preference."
  note "Free tier terms: https://cloud.google.com/free/docs/free-cloud-features"
  printf '\n'
  open_url "https://console.cloud.google.com/compute/instancesAdd"
  step "Name it ssh-site."
  step "Region us-east1 - the free tier covers us-east1, us-central1 and"
  say  "  us-west1 only. Machine type e2-micro, series E2."
  step "Boot disk: Ubuntu 24.04 LTS, standard persistent disk, 30 GB or less."
  step "Security, then Manage access: Add manually generated SSH keys, and"
  say  "  paste the key from stage 2."
  step "Networking: leave the default ephemeral external IPv4 address."
  step "Create."
}

# ── the stages ────────────────────────────────────────────────────────────

banner "Provision the box for ssh snehanshn.duckdns.org"

# ── 1 ─────────────────────────────────────────────────────────────────────
stage "What this builds, and what only you can do"
say "This wizard builds the machine the site runs on and the address visitors"
say "type. It stops where the app server begins: slice 12 deploys the site onto"
say "what this leaves behind."
printf '\n'
say "Anything that needs an account is yours. You create the Oracle Cloud"
say "login, enter payment details, accept terms, and sign in to DuckDNS. The"
say "wizard opens each page and tells you exactly what to click. It never sees"
say "a password, and it never writes your DuckDNS token anywhere."
printf '\n'
say "Everything else - keys, firewall rules, moving sshd off port 22, proving"
say "the move worked before it is committed to - the wizard does itself."
printf '\n'
say "Four facts get recorded for slice 12:"
step "the box's public IP"
step "the admin sshd port"
step "where the admin SSH key lives"
step "where the DuckDNS token lives - the location, never the token"
printf '\n'
need ssh
need ssh-keygen
need curl
ok "ssh, ssh-keygen and curl are on your PATH"
if command -v dig >/dev/null 2>&1 || command -v host >/dev/null 2>&1; then
  ok "a DNS lookup tool is here for the address check"
else
  warn "no dig and no host - stage 10 will ask you to confirm DNS by hand"
fi
note "record: $ENV_FILE and $RECORD_MD (both private, both gitignored)"
pause "Press Enter to begin."

# ── 2 ─────────────────────────────────────────────────────────────────────
stage "The admin SSH key"
say "One key, for administering this box and nothing else. The private half"
say "never leaves this machine; only the public half is pasted into a console."
printf '\n'
ask_default ADMIN_KEY "Where should the private key live?" "~/.ssh/ssh-site-admin"
ADMIN_KEY="$(expand_tilde "$ADMIN_KEY")"
if [ -f "$ADMIN_KEY" ]; then
  ok "reusing the key already at $ADMIN_KEY"
else
  mkdir -p "$(dirname "$ADMIN_KEY")"
  say "Generating it now. Give it a passphrase - your ssh-agent will hold it."
  printf '\n'
  ssh-keygen -t ed25519 -a 100 -C "ssh-site admin" -f "$ADMIN_KEY" \
    || die "ssh-keygen failed"
fi
[ -f "$ADMIN_KEY.pub" ] || die "no public half at $ADMIN_KEY.pub"
write_env ADMIN_KEY "$ADMIN_KEY"

# Load it into the agent, so every later step can connect without a prompt.
if ssh-add -l >/dev/null 2>&1 &&
   ssh-add -l 2>/dev/null | grep -qF "$(ssh-keygen -lf "$ADMIN_KEY.pub" | awk '{print $2}')"; then
  ok "the key is already loaded in your ssh-agent"
else
  if [ "$(uname -s)" = "Darwin" ]; then
    ssh-add --apple-use-keychain "$ADMIN_KEY" 2>/dev/null || ssh-add "$ADMIN_KEY" || true
  else
    ssh-add "$ADMIN_KEY" || true
  fi
fi
printf '\n'
say "This is the public half. You will paste it into the console in stage 4:"
printf '\n%s%s%s\n\n' "$DIM" "$(cat "$ADMIN_KEY.pub")" "$RESET"
if command -v pbcopy >/dev/null 2>&1; then
  pbcopy < "$ADMIN_KEY.pub" && ok "copied to your clipboard"
fi
pause "Press Enter when you have it."

# ── 3 ─────────────────────────────────────────────────────────────────────
stage "The Oracle Cloud account"
if confirm "Do you already have an Oracle Cloud account whose home region is US East (Ashburn)?"; then
  ok "using the account you have"
else
  open_url "https://signup.cloud.oracle.com/"
  step "Sign up with your own details. The wizard never types them."
  step "Set Home Region to US East (Ashburn). This cannot be changed afterwards,"
  say  "  and Always Free resources only exist in the home region - getting this"
  say  "  wrong means starting the account over."
  step "Oracle asks for a card to verify identity. Always Free resources do not"
  say  "  bill; leaving the account on Free Tier is what keeps this at zero."
  printf '\n'
  note "Do not upgrade to Pay As You Go for this. It is the usual advice for"
  note "getting A1 capacity, and Always Free shapes do stay free on a PAYG"
  note "account - but anything outside them then bills for real, and \$0/month"
  note "is a hard requirement here. The capacity ladder in the next stage is"
  note "the answer to A1 being full."
  pause "Press Enter once the account exists and the console loads."
fi

# ── 4 ─────────────────────────────────────────────────────────────────────
stage "Create the instance"
say "Three rungs, in order, and each one is a box that works. Take the next"
say "rung the moment one refuses rather than waiting for capacity to appear."
printf '\n'
step "1. Oracle VM.Standard.A1.Flex, 1 OCPU / 6 GB, Ashburn - what the spec asks for"
step "2. Oracle VM.Standard.E2.1.Micro, same account - the instant fallback"
step "3. Google e2-micro in us-east1 - plan B, and it needs a billing account"
printf '\n'
say "Which rung the box lands on is recorded, because slice 12 builds a binary"
say "for its architecture."
pause "Press Enter to start at rung 1."

RUNG="${BOX_HOST:-}"
if [ -z "$RUNG" ]; then RUNG="$(_existing BOX_HOST || true)"; fi
if [ -z "$RUNG" ]; then RUNG="oracle-a1"; fi

while :; do
  case "$RUNG" in
    oracle-a1)
      attempt_oracle_a1
      printf '\n'
      if confirm "Is the instance running?"; then
        BOX_SHAPE="VM.Standard.A1.Flex"; BOX_REGION="us-ashburn-1"; break
      fi
      warn "A1 is full. Stepping down to E2.1.Micro on the same account."
      pause "Press Enter."
      RUNG="oracle-e2" ;;
    oracle-e2)
      attempt_oracle_e2
      printf '\n'
      if confirm "Is the instance running?"; then
        BOX_SHAPE="VM.Standard.E2.1.Micro"; BOX_REGION="us-ashburn-1"; break
      fi
      warn "Oracle is not cooperating. Stepping down to Google e2-micro."
      pause "Press Enter."
      RUNG="gcp" ;;
    gcp)
      attempt_gcp
      printf '\n'
      if confirm "Is the instance running?"; then
        BOX_SHAPE="e2-micro"; BOX_REGION="us-east1"; break
      fi
      die "all three rungs refused - nothing left to step down to" ;;
    *)
      RUNG="oracle-a1" ;;
  esac
done

BOX_HOST="$RUNG"
ok "landed on $BOX_HOST ($BOX_SHAPE, $BOX_REGION)"
printf '\n'
case "$BOX_HOST" in
  oracle-*) say "The public IP is on the instance page, under Instance access." ;;
  gcp)      say "The public IP is the External IP column on the VM instances list." ;;
esac
while :; do
  ask BOX_IP "Public IPv4 address of the instance:"
  if printf '%s' "$BOX_IP" | grep -qE '^([0-9]{1,3}\.){3}[0-9]{1,3}$'; then break; fi
  fail "that is not an IPv4 address"
done
ask_default REMOTE_USER "Login user on the image" "ubuntu"
write_env BOX_HOST "$BOX_HOST"
write_env BOX_SHAPE "$BOX_SHAPE"
write_env BOX_REGION "$BOX_REGION"
write_env BOX_IP "$BOX_IP"
write_env REMOTE_USER "$REMOTE_USER"

# ── 5 ─────────────────────────────────────────────────────────────────────
stage "First contact"
say "Signing in on port 22, where the image's own sshd is still listening."
say "A fresh instance takes a moment to finish booting."
printf '\n'
if ! wait_for_box 22 24; then
  fail "no session on $BOX_IP:22"
  note "Check the instance is Running, that it has the public IP you typed,"
  note "that the key you pasted matches $ADMIN_KEY.pub, and that the login user"
  note "is right (Ubuntu images use 'ubuntu')."
  die "cannot reach the box"
fi
ok "signed in as $REMOTE_USER@$BOX_IP on port 22"
BOX_OS="$(box 22 '. /etc/os-release; printf "%s %s" "$NAME" "$VERSION_ID"' 2>/dev/null || true)"
BOX_ARCH="$(box 22 'dpkg --print-architecture 2>/dev/null || uname -m' 2>/dev/null || true)"
if [ -z "$BOX_ARCH" ]; then BOX_ARCH="unknown"; fi
ok "${BOX_OS:-unknown OS} on $BOX_ARCH"
if box 22 'sudo -n true' >/dev/null 2>&1; then
  ok "passwordless sudo works - the wizard can do the rest itself"
else
  die "no passwordless sudo for $REMOTE_USER; the remaining stages need it"
fi
write_env BOX_OS "$BOX_OS"
write_env BOX_ARCH "$BOX_ARCH"
pause "Press Enter."

# ── 6 ─────────────────────────────────────────────────────────────────────
stage "Choose the admin port and open it at the cloud edge"
say "Port 22 belongs to the app server, so the admin daemon moves somewhere"
say "high. Two firewalls stand between you and that port: the cloud's, here,"
say "and the instance's own, in the next stage. Both have to be opened before"
say "sshd moves, or the move looks exactly like a lockout."
printf '\n'
SUGGESTED_PORT=$((20000 + RANDOM % 12000))
while :; do
  ask_default ADMIN_PORT "Admin sshd port" "$SUGGESTED_PORT"
  if ! printf '%s' "$ADMIN_PORT" | grep -qE '^[0-9]+$'; then
    fail "digits only"; continue
  fi
  if [ "$ADMIN_PORT" = "22" ]; then
    fail "22 is the app server's - that is the whole point of moving"; continue
  fi
  if [ "$ADMIN_PORT" -lt 1024 ] || [ "$ADMIN_PORT" -gt 65535 ]; then
    fail "pick something between 1024 and 65535"; continue
  fi
  if [ "$ADMIN_PORT" -ge 32768 ] && [ "$ADMIN_PORT" -le 60999 ]; then
    warn "that is inside Linux's ephemeral port range: an outbound connection"
    warn "can take it first and sshd then fails to bind on a reboot."
    if confirm "Use it anyway?"; then break; fi
    continue
  fi
  break
done
write_env ADMIN_PORT "$ADMIN_PORT"
printf '\n'
if [ "$BOX_HOST" = "gcp" ]; then
  open_url "https://console.cloud.google.com/networking/firewalls/list"
  step "VPC network, Firewall, Create firewall rule."
  step "Name allow-admin-ssh, direction Ingress, action Allow."
  step "Targets: all instances in the network."
  step "Source IPv4 ranges: 0.0.0.0/0"
  step "Protocols and ports: TCP, $ADMIN_PORT"
  printf '\n'
  note "Or, if you have gcloud set up, the same rule in one line:"
  note "  gcloud compute firewall-rules create allow-admin-ssh \\"
  note "    --allow=tcp:$ADMIN_PORT --source-ranges=0.0.0.0/0"
else
  open_url "https://cloud.oracle.com/networking/vcns"
  step "Open the VCN the instance is in, then its public subnet, then the"
  say  "  Default Security List. The quickest way there is the subnet link on"
  say  "  the instance page itself, under Primary VNIC."
  step "Add Ingress Rules, and add one rule:"
  say  "    Stateless: unchecked"
  say  "    Source Type: CIDR        Source CIDR: 0.0.0.0/0"
  say  "    IP Protocol: TCP         Destination Port Range: $ADMIN_PORT"
  say  "    Source Port Range: leave empty"
  printf '\n'
  note "Leave the existing TCP/22 rule alone. The default security list opens"
  note "22 to the world already, and that is what the app server needs."
  note "Docs: https://docs.oracle.com/en-us/iaas/Content/Network/Concepts/securitylists.htm"
fi
printf '\n'
while ! confirm "Is the ingress rule for TCP/$ADMIN_PORT saved?"; do
  warn "sshd cannot move until it is - the next stages depend on this rule."
  pause "Press Enter to check again."
done
ok "TCP/$ADMIN_PORT is open at the cloud edge"

# ── 7 ─────────────────────────────────────────────────────────────────────
stage "Open the port in the instance's own firewall"
say "Oracle's Ubuntu images ship an iptables ruleset of their own whose INPUT"
say "chain ends in REJECT. The security list rule you just added is necessary"
say "and not sufficient: without this stage the new port stays silently"
say "unreachable, and moving sshd onto it would look exactly like a lockout."
printf '\n'
if ! remote 22 "$ADMIN_PORT" <<'EOS'
set -eu
port="$1"

if ! iptables -C INPUT -p tcp --dport "$port" -m conntrack --ctstate NEW -j ACCEPT 2>/dev/null; then
    iptables -I INPUT -p tcp --dport "$port" -m conntrack --ctstate NEW -j ACCEPT
    echo "iptables: accepting new connections on $port"
else
    echo "iptables: rule for $port already present"
fi

if command -v ip6tables >/dev/null 2>&1; then
    if ! ip6tables -C INPUT -p tcp --dport "$port" -m conntrack --ctstate NEW -j ACCEPT 2>/dev/null; then
        ip6tables -I INPUT -p tcp --dport "$port" -m conntrack --ctstate NEW -j ACCEPT || true
        echo "ip6tables: accepting new connections on $port"
    fi
fi

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "^Status: active"; then
    ufw allow "$port"/tcp
    echo "ufw: allowed $port/tcp"
fi

# Persist, or the next reboot silently closes the port again.
if command -v netfilter-persistent >/dev/null 2>&1; then
    netfilter-persistent save >/dev/null
    echo "persisted with netfilter-persistent"
elif [ -d /etc/iptables ]; then
    iptables-save > /etc/iptables/rules.v4
    echo "persisted to /etc/iptables/rules.v4"
else
    echo "nothing to persist to: this image ships no saved ruleset, so its"
    echo "firewall is the cloud's alone"
fi

echo "--- INPUT chain now ---"
iptables -S INPUT
EOS
then
  die "could not update the instance firewall"
fi
printf '\n'
ok "TCP/$ADMIN_PORT is open on the instance too"
note "Slice 12 replaces this ruleset with nftables. What it inherits is an"
note "iptables ruleset with 22 and $ADMIN_PORT accepted."
pause "Press Enter."

# ── 8 ─────────────────────────────────────────────────────────────────────
stage "Start sshd on $ADMIN_PORT, while 22 still works"
say "This is the step that locks people out of fresh boxes, so it happens in"
say "two halves. Now: sshd listens on both 22 and $ADMIN_PORT, and the wizard"
say "proves a brand new session opens on $ADMIN_PORT. Only then, in stage 9,"
say "does it stop listening on 22."
printf '\n'
say "Belt and braces: open a second terminal and leave this connected in it."
printf '\n    %sssh -i %s %s@%s%s\n\n' "$BOLD" "$ADMIN_KEY" "$REMOTE_USER" "$BOX_IP" "$RESET"
say "The box also arms a rollback of its own: if this wizard does not cancel"
say "it, the box puts sshd back exactly as the image shipped it, on its own,"
say "without you."
printf '\n'
pause "Press Enter when that second session is connected."
printf '\n'
if ! remote 22 "$ADMIN_PORT" <<'EOS'
set -eu
port="$1"
cfg=/etc/ssh/sshd_config
drop=/etc/ssh/sshd_config.d/10-ssh-site-admin.conf

# One pristine copy of what the image shipped, taken once and never again.
[ -f "$cfg.pre-ssh-site" ] || cp "$cfg" "$cfg.pre-ssh-site"

mkdir -p /etc/ssh/sshd_config.d
if ! grep -qE '^[[:space:]]*Include[[:space:]]+/etc/ssh/sshd_config\.d/\*\.conf' "$cfg"; then
    printf 'Include /etc/ssh/sshd_config.d/*.conf\n' > "$cfg.new"
    cat "$cfg" >> "$cfg.new"
    mv "$cfg.new" "$cfg"
    echo "sshd_config had no Include line; added one"
fi

printf '%s\n' \
    '# Written by scripts/provision-box.sh (ssh-site slice 11).' \
    '# Both ports, while the new one is being proved. Stage 9 drops 22.' \
    'Port 22' "Port $port" > "$drop"
chmod 644 "$drop"
sshd -t

# The dead man's switch. If the wizard cannot reach the new port it never
# cancels this, and the box restores itself.
systemctl stop ssh-site-rollback.timer 2>/dev/null || true
systemctl reset-failed ssh-site-rollback.service ssh-site-rollback.timer 2>/dev/null || true
systemd-run --collect --unit=ssh-site-rollback --on-active=15min /bin/sh -c \
    "rm -f $drop; cp $cfg.pre-ssh-site $cfg; systemctl unmask ssh.socket; systemctl enable --now ssh.service; systemctl restart ssh.service" >/dev/null
echo "rollback armed"

# Ubuntu can socket-activate sshd, which hands every connection its own sshd
# process. That makes MaxStartups and PerSourceMaxStartups - both of which
# slice 12 sets - do nothing at all, and it owns port 22 independently of
# sshd_config. Take the classic listener instead, and mask the socket so an
# openssh upgrade cannot hand port 22 back to sshd behind the app server.
systemctl reset-failed ssh-site-portmove.service 2>/dev/null || true
systemd-run --collect --unit=ssh-site-portmove /bin/sh -c \
    "systemctl disable --now ssh.socket 2>/dev/null; systemctl mask ssh.socket 2>/dev/null; systemctl enable ssh.service; systemctl restart ssh.service" >/dev/null
echo "sshd restarting on 22 and $port"
EOS
then
  die "could not reconfigure sshd; nothing has moved, 22 is still yours"
fi
printf '\n'
if wait_for_box "$ADMIN_PORT" 12; then
  ok "a brand new session opened on $ADMIN_PORT"
else
  fail "nothing answers on $ADMIN_PORT"
  printf '\n'
  note "Port 22 still works, and the box rolls itself back shortly regardless."
  note "Worth checking before re-running: that the ingress rule really saved,"
  note "with protocol TCP and destination port $ADMIN_PORT; and, if the cloud"
  note "rule is right, the instance firewall output from stage 7."
  die "the new port is not reachable, so 22 stays exactly where it is"
fi
if box_ok 22; then ok "port 22 still answers too - both ports live"; fi
remote "$ADMIN_PORT" <<'EOS' || true
systemctl stop ssh-site-rollback.timer 2>/dev/null || true
systemctl reset-failed ssh-site-rollback.service ssh-site-rollback.timer 2>/dev/null || true
echo "rollback cancelled"
EOS
pause "Press Enter to free port 22."

# ── 9 ─────────────────────────────────────────────────────────────────────
stage "Free port 22 for the app server"
say "sshd drops 22 and keeps $ADMIN_PORT. You are already connected there, and"
say "the second terminal on 22 will drop - that is the point."
printf '\n'
if ! remote "$ADMIN_PORT" "$ADMIN_PORT" <<'EOS'
set -eu
port="$1"
drop=/etc/ssh/sshd_config.d/10-ssh-site-admin.conf
printf '%s\n' \
    '# Written by scripts/provision-box.sh (ssh-site slice 11).' \
    '# Port 22 belongs to the app server. The admin daemon never listens there.' \
    "Port $port" > "$drop"
sshd -t
systemctl reset-failed ssh-site-freeport.service 2>/dev/null || true
systemd-run --collect --unit=ssh-site-freeport /bin/sh -c \
    "sleep 1; systemctl restart ssh.service" >/dev/null
echo "sshd restarting on $port alone"
EOS
then
  die "could not drop port 22; sshd is unchanged and still on both ports"
fi
printf '\n'
if ! wait_for_box "$ADMIN_PORT" 12; then
  fail "lost the admin port during the restart"
  note "Recover from the cloud console's serial or VNC connection, or rebuild:"
  note "delete the instance and run this wizard again from the top."
  die "sshd did not come back on $ADMIN_PORT"
fi
ok "still signed in on $ADMIN_PORT after the restart"
printf '\n'
remote "$ADMIN_PORT" "$ADMIN_PORT" <<'EOS'
set -eu
port="$1"
if ss -ltnH "sport = :22" | grep -q .; then
    echo "PORT 22: still has a listener"
else
    echo "port 22: no listener - free for the app server"
fi
if ss -ltnH "sport = :$port" | grep -q .; then
    echo "port $port: sshd listening"
else
    echo "PORT $port: nothing listening"
fi
printf 'ssh.socket: %s\n' "$(systemctl is-enabled ssh.socket 2>&1 || true)"
printf 'ssh.service: %s\n' "$(systemctl is-enabled ssh.service 2>&1 || true)"
EOS
printf '\n'

# "Reachable with a key" is only half the claim; the other half is that a
# password cannot open this door. Ask sshd what it actually resolved to,
# rather than assuming what the image shipped.
SSHD_EFFECTIVE="$(box "$ADMIN_PORT" 'sudo sshd -T' 2>/dev/null || true)"
PASSWORD_AUTH="$(printf '%s\n' "$SSHD_EFFECTIVE" | awk '/^passwordauthentication /{print $2}')"
if [ -z "$PASSWORD_AUTH" ]; then PASSWORD_AUTH="unverified"; fi
printf '%s\n' "$SSHD_EFFECTIVE" |
  grep -E '^(port|passwordauthentication|pubkeyauthentication|permitrootlogin) ' |
  while read -r line; do note "$line"; done || true
case "$PASSWORD_AUTH" in
  no) ok "passwords cannot open the admin port - key-only, as the image shipped it" ;;
  unverified) warn "could not read sshd's effective config; check PasswordAuthentication by hand" ;;
  *) warn "PasswordAuthentication is '$PASSWORD_AUTH' on the admin port."
     warn "Slice 12 turns it off explicitly, but until then this door takes passwords." ;;
esac
printf '\n'
P22="$(box 22 true 2>&1 || true)"
case "$P22" in
  *"Connection refused"*)
    ok "from out here, port 22 refuses connections: open at the firewall, with"
    ok "nothing behind it. Exactly what the app server needs to bind." ;;
  *"imed out"*|*"Operation timed out"*)
    warn "port 22 times out rather than refusing, so something is filtering it."
    warn "The app server needs 22 reachable - check the security list still has"
    warn "its TCP/22 ingress rule before slice 12 deploys." ;;
  *)
    warn "port 22 answered something unexpected:"
    note "$P22" ;;
esac
ssh-keygen -R "$BOX_IP" >/dev/null 2>&1 || true
note "Dropped $BOX_IP from your known_hosts: the next thing to answer on port"
note "22 is the app server, with a different host key, and a stale entry would"
note "greet you with a man-in-the-middle warning."
printf '\n'
if confirm "Reboot the box now, to prove it comes back this way on its own?"; then
  remote "$ADMIN_PORT" <<'EOS' || true
systemd-run --collect --on-active=3 --unit=ssh-site-reboot systemctl reboot >/dev/null
echo "rebooting"
EOS
  sleep 15
  if wait_for_box "$ADMIN_PORT" 24; then
    ok "back up on $ADMIN_PORT after a reboot"
    remote "$ADMIN_PORT" "$ADMIN_PORT" <<'EOS'
set -eu
port="$1"
if ss -ltnH "sport = :22" | grep -q .; then echo "PORT 22: listener came back"; else echo "port 22: still free"; fi
if ss -ltnH "sport = :$port" | grep -q .; then echo "port $port: sshd listening"; else echo "PORT $port: nothing listening"; fi
iptables -S INPUT | grep -- "--dport $port" || echo "PORT $port: firewall rule did not survive the reboot"
EOS
  else
    die "the box did not come back on $ADMIN_PORT after rebooting"
  fi
fi
pause "Press Enter."

# ── 10 ────────────────────────────────────────────────────────────────────
stage "Claim the address"
say "DuckDNS gives the box the name visitors type. The label was unclaimed when"
say "the host was picked, and unclaimed again when this wizard was written -"
say "DuckDNS refuses it outright if someone has taken it since."
printf '\n'
open_url "https://www.duckdns.org/"
step "Sign in with one of the identity providers on the front page."
step "On the domains page, type the label into the sub domain box and click"
say  "  add domain. It appears in the table below if it was free."
printf '\n'
while :; do
  ask_default DUCKDNS_LABEL "The label you claimed" "snehanshn"
  if printf '%s' "$DUCKDNS_LABEL" | grep -qE '^[a-z0-9][a-z0-9-]*$'; then break; fi
  fail "lowercase letters, digits and dashes only"
done
ADDRESS="$DUCKDNS_LABEL.duckdns.org"
if [ "$DUCKDNS_LABEL" != "snehanshn" ]; then
  warn "the address is $ADDRESS, not snehanshn.duckdns.org."
  note "That string is written down in this repo's README and in the build"
  note "spec, and it becomes a links fact in the content pack in slice 13."
  note "All three follow this, not the other way round."
fi
printf '\n'
say "The token is the string at the top of that same domains page. It is a"
say "credential: it is never written into the record, never committed, and"
say "never typed anywhere this wizard can keep it."
printf '\n'
ask_default DUCKDNS_TOKEN_LOCATION "Where do you keep that token?" "~/.config/duckdns/token"
DUCK_TOKEN=""
DUCK_TOKEN_PATH="$(expand_tilde "$DUCKDNS_TOKEN_LOCATION")"
if [ -n "${DUCKDNS_TOKEN:-}" ]; then
  DUCK_TOKEN="$DUCKDNS_TOKEN"
  ok "using the token in \$DUCKDNS_TOKEN"
elif [ -f "$DUCK_TOKEN_PATH" ]; then
  DUCK_TOKEN="$(tr -d ' \t\n\r' < "$DUCK_TOKEN_PATH")"
  ok "read the token from $DUCKDNS_TOKEN_LOCATION"
else
  ask_secret DUCKDNS_TOKEN_TYPED "Paste the token (hidden, and kept only in memory):"
  DUCK_TOKEN="$DUCKDNS_TOKEN_TYPED"
  DUCKDNS_TOKEN_TYPED=""
fi
[ -n "$DUCK_TOKEN" ] || die "no DuckDNS token, so the address cannot be pointed anywhere"
write_env DUCKDNS_LABEL "$DUCKDNS_LABEL"
write_env ADDRESS "$ADDRESS"
write_env DUCKDNS_TOKEN_LOCATION "$DUCKDNS_TOKEN_LOCATION"
printf '\n'

# The token goes to curl on stdin rather than in an argument, so it never
# appears in this machine's process list.
DUCK_RESP="$(printf 'url = "https://www.duckdns.org/update?domains=%s&token=%s&ip=%s&verbose=true"\n' \
  "$DUCKDNS_LABEL" "$DUCK_TOKEN" "$BOX_IP" | curl -sS --config - || true)"
DUCK_TOKEN=""
case "$DUCK_RESP" in
  OK*) ok "DuckDNS accepted the update: $ADDRESS points at $BOX_IP" ;;
  KO*) die "DuckDNS rejected the update - the token or the label is wrong" ;;
  *)   fail "unexpected answer from DuckDNS:"; note "${DUCK_RESP:-(empty)}"
       die "could not point the address at the box" ;;
esac
printf '\n'
if command -v dig >/dev/null 2>&1 || command -v host >/dev/null 2>&1; then
  DNS_TRY=1
  while [ "$DNS_TRY" -le 12 ]; do
    RESOLVED="$(resolve_a "$ADDRESS")"
    if [ "$RESOLVED" = "$BOX_IP" ]; then break; fi
    printf '  %s· waiting for %s to resolve (%s/12)%s\n' "$DIM" "$ADDRESS" "$DNS_TRY" "$RESET"
    sleep 5
    DNS_TRY=$((DNS_TRY + 1))
  done
  if [ "${RESOLVED:-}" = "$BOX_IP" ]; then
    ok "$ADDRESS resolves to $BOX_IP"
  else
    warn "$ADDRESS resolves to '${RESOLVED:-nothing}', not $BOX_IP."
    warn "DuckDNS took the update, so this is usually a cached lookup - check"
    warn "again shortly before treating it as broken."
  fi
else
  note "No dig or host here, so check by hand that $ADDRESS resolves to $BOX_IP."
fi
if ssh -n "${SSH_BASE[@]}" -i "$ADMIN_KEY" -p "$ADMIN_PORT" "$REMOTE_USER@$ADDRESS" true >/dev/null 2>&1; then
  ok "signed in over the name: ssh -p $ADMIN_PORT $REMOTE_USER@$ADDRESS"
else
  warn "the name does not carry an admin session yet, though the IP does."
fi
pause "Press Enter."

# ── 11 ────────────────────────────────────────────────────────────────────
stage "The bill, and the record"
say "The box is only worth having if it is free. Check that now, while the"
say "console is still open."
printf '\n'
if [ "$BOX_HOST" = "gcp" ]; then
  open_url "https://console.cloud.google.com/billing"
  step "Open Billing, then Reports, and look at this month's cost."
  step "Look specifically for an external IPv4 line. If the address is billed,"
  say  "  this box does not meet the zero-dollar requirement and the right move"
  say  "  is back up the ladder, not onward."
else
  open_url "https://cloud.oracle.com/"
  step "Open the menu, then Billing & Cost Management, then Cost Analysis."
  step "This month should read zero."
  step "On the instance page, confirm the shape still carries the Always Free"
  say  "  eligible badge, and that no second instance is running that does not."
  note "The Free Trial credits expiring changes nothing here: Always Free"
  note "resources keep running when the trial ends. Allowances, for reference:"
  note "https://docs.oracle.com/en-us/iaas/Content/FreeTier/freetier_topic-Always_Free_Resources.htm"
fi
printf '\n'
if confirm "Does the bill read zero?"; then
  BILL="zero"
  ok "recorded as free"
else
  BILL="unresolved"
  warn "recorded as unresolved. Slice 12 should not deploy onto a box that bills."
fi
write_env BILL_STATUS "$BILL"
printf '\n'

cat > "$RECORD_MD" <<RECORD
# The box

Written by scripts/provision-box.sh in the ssh-site repo, on $(date -u '+%Y-%m-%d %H:%M UTC').
Do not hand-edit: re-running the wizard rewrites this file from what it verified.
Machine-readable values live beside it in box.env; this is the same facts, read
aloud. That file is KEY=value lines, one per line, values unquoted and free to
contain spaces - read a value out of it with grep and cut rather than sourcing it.

Private, like the rest of .scratch/. Nothing here is committed to the public repo,
and nothing here is a secret: the DuckDNS token's location is recorded, never the token.

## The four facts slice 12 needs

| Fact | Value |
| --- | --- |
| Public IP | $BOX_IP |
| Admin sshd port | $ADMIN_PORT |
| Admin SSH key | $ADMIN_KEY (public half at $ADMIN_KEY.pub) |
| DuckDNS token | $DUCKDNS_TOKEN_LOCATION |

Get in with:

    ssh -i $ADMIN_KEY -p $ADMIN_PORT $REMOTE_USER@$ADDRESS

## Which rung of the ladder this box is

| | |
| --- | --- |
| Host | $BOX_HOST |
| Shape | $BOX_SHAPE |
| Region | $BOX_REGION |
| Image | $BOX_OS |
| Architecture | $BOX_ARCH |
| Address | $ADDRESS |
| Login user | $REMOTE_USER |
| Bill | $BILL (checked in the console at the end of the wizard) |

The architecture is the fact slice 12 builds against: the Go binary is compiled
for $BOX_ARCH.

## The state slice 12 inherits

- Port 22 has no listener and is open at the cloud firewall, so the app server
  can bind it.
- The admin sshd listens on $ADMIN_PORT only. Its effective
  PasswordAuthentication, read back from sshd itself rather than assumed, is
  $PASSWORD_AUTH. Slice 12 sets the rest of the sshd_config baseline.
- sshd is configured through /etc/ssh/sshd_config.d/10-ssh-site-admin.conf,
  which carries the Port line. The image's own sshd_config is untouched, with a
  pristine copy at /etc/ssh/sshd_config.pre-ssh-site.
- ssh.socket is masked deliberately. Socket activation gives every connection
  its own sshd, which makes MaxStartups and PerSourceMaxStartups inert, and it
  would take port 22 back from the app server after an openssh upgrade.
- The instance firewall is iptables, with 22 and $ADMIN_PORT accepted, saved so
  it survives a reboot. Slice 12 replaces this with nftables and should expect
  to flush what is there rather than layer on top of it.
- The cloud firewall has one added ingress rule: TCP/$ADMIN_PORT from 0.0.0.0/0.
  22 was already open by default.
- Nothing else is installed. No nftables rules of the project's own, no
  fail2ban, no systemd unit for the app, no host key at /var/lib/ssh-site.

## Rebuilding this box

Delete the instance and run the wizard again from the top. It regenerates
nothing it does not have to: the same key is reused, the same label is
re-pointed at the new IP, and the values above come back as the defaults at
every prompt.
RECORD
ok "wrote $RECORD_MD"

# The library's closing summary guards its lines with `(( ... )) && note`,
# which returns non-zero when a list is empty - and no GitHub secrets are set
# here. Relax errexit for the summary, or set -e would cut it short.
set +e
finish
note "Next: slice 12 hardens this box and deploys the app server onto port 22."
printf '\n'

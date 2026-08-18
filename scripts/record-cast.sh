#!/usr/bin/env bash
# Records the README's terminal cast against the live site and converts it
# to the GIF actually embedded there.
#
# A developer tool, like `make art`: it needs network access to the live
# box and asciinema/agg/expect installed locally, so it never runs in CI -
# CI ships whatever cast/demo.gif already has committed. Re-run it whenever
# the walkthrough stops matching what's live.
#
#   bash scripts/record-cast.sh
#
# The walkthrough is driven by expect, waiting on the app's own page-kind
# chrome (list footers, detail footers, the arrival card's key legend)
# rather than on any one person's facts, so a content-pack push never
# breaks it: arrival, a work drill-down, back out, an award drilling into
# its project, then quit.
set -euo pipefail

ADDRESS="${1:-snehanshn.duckdns.org}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CAST_DIR="$ROOT/cast"
CAST_FILE="$CAST_DIR/demo.cast"
GIF_FILE="$CAST_DIR/demo.gif"
DRIVER="$(mktemp)"
trap 'rm -f "$DRIVER"' EXIT

for cmd in asciinema agg expect; do
  command -v "$cmd" >/dev/null 2>&1 || {
    echo "missing $cmd - install with: brew install asciinema agg (expect ships with macOS)" >&2
    exit 1
  }
done

mkdir -p "$CAST_DIR"

cat > "$DRIVER" <<EXP
#!/usr/bin/expect -f
set timeout 90
log_user 0
spawn ssh -tt -o UserKnownHostsFile=/dev/null $ADDRESS
log_user 1
stty rows 30 cols 100 < \$spawn_out(slave,name)

expect "continue connecting"
send "yes\r"

expect "\[q\] quit"
sleep 1.5
send "w"

expect "move*enter open"
sleep 1.2
send "\r"

expect "scroll*esc back"
sleep 2.0
send "\033"

expect "move*enter open"
sleep 1.0
send "\033"

expect "\[q\] quit"
sleep 1.2
send "a"

expect "move*enter open"
sleep 1.2
send "\r"

expect "scroll*esc back"
sleep 2.2
send "q"

sleep 1.5
expect eof
EXP
chmod +x "$DRIVER"

asciinema rec --headless --window-size 100x30 --idle-time-limit 2 --overwrite \
  --command "expect $DRIVER" \
  --title "ssh $ADDRESS" \
  "$CAST_FILE"

agg --idle-time-limit 1.5 --font-size 14 "$CAST_FILE" "$GIF_FILE"

echo "wrote $CAST_FILE and $GIF_FILE"

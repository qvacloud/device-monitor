#!/usr/bin/env bash
# Deploy the agent to remote Linux hosts (Orange Pi / any systemd box) over SSH.
# Usage: PI_PASS='secret' ARCH=arm64 ./scripts/deploy-pi.sh 192.168.1.49 192.168.1.50
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ARCH="${ARCH:-arm64}"
USER="${USER_NAME:-root}"
: "${PI_PASS:?export PI_PASS with the ssh password}"

BIN="$ROOT/dist/dm-agent-$ARCH"
UNIT="$ROOT/scripts/dm-agent.service"
[ -f "$BIN" ] || { echo "missing $BIN (run: make cross)"; exit 1; }

ASK="$(mktemp)"
printf '#!/bin/sh\necho "%s"\n' "$PI_PASS" > "$ASK"
chmod +x "$ASK"
trap 'rm -f "$ASK"' EXIT
export SSH_ASKPASS="$ASK" SSH_ASKPASS_REQUIRE=force DISPLAY=:0

SSHOPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null
  -o PreferredAuthentications=password -o PubkeyAuthentication=no -o ConnectTimeout=8)

REMOTE='install -m755 /tmp/dm-agent /usr/local/bin/dm-agent
install -m644 /tmp/dm-agent.service /etc/systemd/system/dm-agent.service
systemctl daemon-reload
systemctl enable dm-agent >/dev/null 2>&1
systemctl restart dm-agent
sleep 1
printf "state: "; systemctl is-active dm-agent
printf "health: "; curl -s --max-time 3 http://127.0.0.1:55555/health'

for host in "$@"; do
  echo "== $host =="
  scp "${SSHOPTS[@]}" "$BIN" "$USER@$host:/tmp/dm-agent"
  scp "${SSHOPTS[@]}" "$UNIT" "$USER@$host:/tmp/dm-agent.service"
  setsid -w ssh "${SSHOPTS[@]}" "$USER@$host" "$REMOTE"
done

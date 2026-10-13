#!/usr/bin/env bash
# Fork a Nyx chain into a local single-validator testnet.
#
# Copies the data of a stopped node into a new home, gives the copy a fresh
# validator key, a fresh operator account and an empty peer list, then runs
# `nyxd in-place-testnet` on it. The source home is never modified.
#
# usage: scripts/testnetify.sh -s <source-home> -o <fork-home> [options]
#   -s DIR   home of a cleanly stopped node (needs config/genesis.json and data/)
#   -o DIR   home to create for the fork (must not exist)
#   -c ID    chain id of the fork (default: nyx-fork-1)
#   -u NAME  upgrade to run on the first block, e.g. v0.61; required when the
#            binary is one upgrade ahead of the copied state (e.g. forking
#            mainnet before v0.61 with the v0.61 binary)
#   -f ADDRS comma-separated extra accounts to fund
#   -b BIN   nyxd binary (default: nyxd on PATH)
#   -n       prepare the fork but do not start it
#
# The fork node listens on the default ports (26656/26657/9090/1317); stop other
# local nodes first. Once in-place-testnet has committed its first block, stop it
# and restart with: nyxd start --home <fork-home>
set -euo pipefail

CHAIN_ID=nyx-fork-1
NYXD=nyxd
UPGRADE=""
EXTRA_ACCOUNTS=""
START=1
SRC=""
DST=""

usage() { sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'; exit 1; }

while getopts "s:o:c:u:f:b:nh" opt; do
  case "$opt" in
    s) SRC=$OPTARG ;;
    o) DST=$OPTARG ;;
    c) CHAIN_ID=$OPTARG ;;
    u) UPGRADE=$OPTARG ;;
    f) EXTRA_ACCOUNTS=$OPTARG ;;
    b) NYXD=$OPTARG ;;
    n) START=0 ;;
    *) usage ;;
  esac
done
[[ -n "$SRC" && -n "$DST" ]] || usage
[[ -f "$SRC/config/genesis.json" && -d "$SRC/data" ]] || { echo "error: $SRC has no config/genesis.json or data/" >&2; exit 1; }
[[ ! -e "$DST" ]] || { echo "error: $DST already exists" >&2; exit 1; }
command -v "$NYXD" >/dev/null || { echo "error: $NYXD not found" >&2; exit 1; }

# a running node holds LevelDB's LOCK file open read-write; read-only handles
# (e.g. Docker Desktop's file sharing after a container exits) are harmless
if [[ -f "$SRC/data/application.db/LOCK" ]] && command -v lsof >/dev/null &&
  lsof -F a "$SRC/data/application.db/LOCK" 2>/dev/null | grep -qE '^a[uw]'; then
  echo "error: a process still has $SRC/data/application.db open for writing; stop the node first" >&2
  exit 1
fi

echo "==> initialising $DST (fresh node and validator keys)"
"$NYXD" init fork-validator --chain-id "$CHAIN_ID" --home "$DST" >/dev/null 2>&1
cp "$SRC/config/genesis.json" "$DST/config/genesis.json"

echo "==> copying chain data from $SRC"
# keep the fresh priv_validator_state.json: the fork signs with a new key.
# skip the consensus WAL: it holds the source network's in-flight votes, which
# the fork would replay on start
rsync -a --exclude priv_validator_state.json --exclude cs.wal "$SRC/data/" "$DST/data/"
if [[ -d "$SRC/wasm" ]]; then
  rsync -a "$SRC/wasm/" "$DST/wasm/"
fi

echo "==> isolating the fork from the source network"
"$NYXD" config set config p2p.seeds "" -s --home "$DST" >/dev/null
"$NYXD" config set config p2p.persistent_peers "" -s --home "$DST" >/dev/null
"$NYXD" config set config p2p.pex false -s --home "$DST" >/dev/null
"$NYXD" config set config statesync.enable false -s --home "$DST" >/dev/null
"$NYXD" config set app api.enable true --home "$DST" >/dev/null
"$NYXD" config set client chain-id "$CHAIN_ID" --home "$DST" >/dev/null
"$NYXD" config set client keyring-backend test --home "$DST" >/dev/null

echo "==> creating the operator account (test keyring in $DST)"
"$NYXD" keys add operator --keyring-backend test --home "$DST" --output json > "$DST/operator-key.json" 2>&1
OPERATOR=$("$NYXD" keys show operator -a --keyring-backend test --home "$DST")
VALOPER=$("$NYXD" keys show operator -a --bech val --keyring-backend test --home "$DST")
FUND="$OPERATOR${EXTRA_ACCOUNTS:+,$EXTRA_ACCOUNTS}"

ARGS=(in-place-testnet "$CHAIN_ID" "$VALOPER" --home "$DST" --skip-confirmation
  --accounts-to-fund "$FUND" --voting-period 60s --expedited-voting-period 30s)
[[ -n "$UPGRADE" ]] && ARGS+=(--trigger-testnet-upgrade "$UPGRADE")

cat > "$DST/fork-info.txt" <<EOF
chain id:          $CHAIN_ID
source home:       $SRC
operator account:  $OPERATOR  (key "operator", keyring-backend test, mnemonic in operator-key.json)
validator:         $VALOPER
funded accounts:   $FUND
triggered upgrade: ${UPGRADE:-none}
command:           $NYXD ${ARGS[*]}
EOF
cat "$DST/fork-info.txt"

if [[ "$START" == 1 ]]; then
  echo "==> starting the fork"
  exec "$NYXD" "${ARGS[@]}"
fi

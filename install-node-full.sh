#!/usr/bin/env bash
set -Eeuo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

INSTALL_ROOT="/etc/xboard-node"
CONFIG_FILE="${INSTALL_ROOT}/config.yml"
TOKEN_FILE="${INSTALL_ROOT}/nodeguard-server-token"
CREDENTIALS_FILE="${INSTALL_ROOT}/credentials.env"
NODE_BINARY="/usr/local/bin/xboard-node"
GUARD_BINARY="/usr/local/bin/node-journal-guard"
NODE_SERVICE="xboard-node.service"
GUARD_SERVICE="node-journal-guard.service"
NODE_SERVICE_PATH="/etc/systemd/system/${NODE_SERVICE}"
GUARD_SERVICE_PATH="/etc/systemd/system/${GUARD_SERVICE}"
BACKUP_ROOT="/root"
DEFAULT_KERNEL="singbox"
DEFAULT_SOURCE_BASE="https://github.com/nonamezr/Xboard-Node/releases/download/v20260704-node-prod-full"
EXPECTED_NODE_SHA="abbc312bae81313e06dd7498faa2273d68447902d4201c31a19b6228b3401b6a"
EXPECTED_GUARD_SHA="fa8a7fecc751187719c1a7797f211588e0255af1621b6fea09a6b06e6ade3cc9"

MODE="machine"
PANEL_URL=""
TOKEN=""
INPUT_TOKEN_FILE=""
MACHINE_ID=""
NODE_ID=""
KERNEL_TYPE="${DEFAULT_KERNEL}"
SOURCE_BASE="${DEFAULT_SOURCE_BASE}"
NODE_BINARY_SOURCE=""
GUARD_BINARY_SOURCE=""
BUILD_LOCAL=0
RESTART_SERVICES=1
VERIFY_PORT=""
TMP_DIR=""
CLEANUP_DONE=0

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }
log_step()  { echo -e "${CYAN}[STEP]${NC} ${BOLD}$1${NC}"; }

cleanup_tmp() {
    if [ "$CLEANUP_DONE" -eq 1 ]; then return; fi
    CLEANUP_DONE=1
    if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then rm -rf "$TMP_DIR"; fi
}
trap cleanup_tmp EXIT

usage() {
    cat <<'HELP'
install-node-full.sh — install/reconcile xboard-node + node-journal-guard

Required:
  --panel URL              Panel base URL
  --machine-id N           Machine ID for xboard-node machine mode
  --node-id N              Node ID for node-journal-guard reporting
  --token TOKEN            Server/machine token (not printed), or use --token-file
  --token-file PATH        Read token from file instead of argv

Optional:
  --mode machine|node      xboard-node config mode (default: machine)
  --kernel singbox|xray    Kernel type for new config only (default: singbox)
  --source-base URL        Directory containing binaries; supports file://, http(s)://
                           default: file:///root/.openclaw/workspace/node-binaries-20260704
  --node-binary PATH       Use local staged xboard-node binary
  --guard-binary PATH      Use local staged node-journal-guard binary
  --build-local            Build both binaries from current repo instead of downloading
  --verify-port PORT       Verify TCP listen port after restart (optional)
  --no-restart             Install files/services but do not restart services
  --help                   Show help

Example:
  curl -fsSL https://example/install-node-full.sh | bash -s -- \
    --mode machine --panel https://panel.example.com --token TOKEN --machine-id 2 --node-id 70

Safety:
  - Idempotent: existing /etc/xboard-node/config.yml is preserved.
  - Does NOT stop/disable V2bX automatically; only warns if detected.
  - Does NOT print token to stdout.
  - Verifies pinned SHA256 for both installed binaries.
HELP
}

parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --mode) MODE="${2:-}"; shift 2 ;;
            --panel|--panel-url|-a) PANEL_URL="${2:-}"; shift 2 ;;
            --token|-t) TOKEN="${2:-}"; shift 2 ;;
            --token-file) INPUT_TOKEN_FILE="${2:-}"; shift 2 ;;
            --machine-id) MACHINE_ID="${2:-}"; shift 2 ;;
            --node-id|-n) NODE_ID="${2:-}"; shift 2 ;;
            --kernel|-k) KERNEL_TYPE="${2:-}"; shift 2 ;;
            --source-base) SOURCE_BASE="${2:-}"; shift 2 ;;
            --node-binary) NODE_BINARY_SOURCE="${2:-}"; shift 2 ;;
            --guard-binary) GUARD_BINARY_SOURCE="${2:-}"; shift 2 ;;
            --build-local) BUILD_LOCAL=1; shift ;;
            --verify-port) VERIFY_PORT="${2:-}"; shift 2 ;;
            --no-restart) RESTART_SERVICES=0; shift ;;
            --help|-h) usage; exit 0 ;;
            *) log_error "Unknown argument: $1"; usage; exit 2 ;;
        esac
    done
}

require_root() {
    if [ "$(id -u)" -ne 0 ]; then
        log_error "Run as root (or sudo)."
        exit 1
    fi
}

validate_args() {
    if [ "$MODE" != "machine" ] && [ "$MODE" != "node" ]; then
        log_error "--mode must be machine or node"
        exit 1
    fi
    if [ -z "$PANEL_URL" ]; then log_error "Missing --panel"; exit 1; fi
    if [ -z "$NODE_ID" ]; then log_error "Missing --node-id for node-journal-guard"; exit 1; fi
    if [ "$MODE" = "machine" ] && [ -z "$MACHINE_ID" ]; then log_error "Missing --machine-id"; exit 1; fi
    if [ "$MODE" = "node" ] && [ -z "$NODE_ID" ]; then log_error "Missing --node-id"; exit 1; fi
    if [ -n "$TOKEN" ] && [ -n "$INPUT_TOKEN_FILE" ]; then log_error "Use only one of --token or --token-file"; exit 1; fi
    if [ -z "$TOKEN" ] && [ -z "$INPUT_TOKEN_FILE" ]; then log_error "Missing --token or --token-file"; exit 1; fi
    if [ -n "$INPUT_TOKEN_FILE" ] && [ ! -f "$INPUT_TOKEN_FILE" ]; then log_error "Token file not found: $INPUT_TOKEN_FILE"; exit 1; fi
}

read_token() {
    if [ -n "$INPUT_TOKEN_FILE" ]; then
        TOKEN="$(tr -d '\r\n' < "$INPUT_TOKEN_FILE")"
    fi
    if [ -z "$TOKEN" ]; then
        log_error "Token is empty"
        exit 1
    fi
}

sha256_file() { sha256sum "$1" | awk '{print $1}'; }

verify_sha() {
    local path="$1" expected="$2" label="$3" actual
    actual="$(sha256_file "$path")"
    if [ "$actual" != "$expected" ]; then
        log_error "${label} SHA mismatch: expected ${expected}, got ${actual}"
        exit 1
    fi
    log_info "${label} SHA OK: ${actual:0:8}..."
}

copy_or_download() {
    local source="$1" dest="$2" label="$3"
    if [[ "$source" == file://* ]]; then
        local f="${source#file://}"
        if [ ! -f "$f" ]; then log_error "${label} source not found: $f"; exit 1; fi
        cp "$f" "$dest"
    elif [[ "$source" == http://* || "$source" == https://* ]]; then
        curl -fsSL "$source" -o "$dest"
    else
        if [ ! -f "$source" ]; then log_error "${label} source not found: $source"; exit 1; fi
        cp "$source" "$dest"
    fi
    chmod +x "$dest"
}

stage_binaries() {
    TMP_DIR="$(mktemp -d)"
    local staged_node="$TMP_DIR/xboard-node"
    local staged_guard="$TMP_DIR/node-journal-guard"

    if [ "$BUILD_LOCAL" -eq 1 ]; then
        log_step "Building binaries from current repo"
        local repo_dir
        repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        (cd "$repo_dir" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -tags "with_quic with_utls with_wireguard with_acme with_clash_api" -o "$staged_node" ./cmd/xboard-node)
        (cd "$repo_dir" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o "$staged_guard" ./cmd/node-journal-guard)
        chmod +x "$staged_node" "$staged_guard"
    else
        local node_src guard_src
        node_src="${NODE_BINARY_SOURCE:-${SOURCE_BASE%/}/xboard-node-linux-amd64}"
        guard_src="${GUARD_BINARY_SOURCE:-${SOURCE_BASE%/}/node-journal-guard-linux-amd64}"
        log_step "Staging pinned binaries"
        copy_or_download "$node_src" "$staged_node" "xboard-node"
        copy_or_download "$guard_src" "$staged_guard" "node-journal-guard"
    fi

    verify_sha "$staged_node" "$EXPECTED_NODE_SHA" "xboard-node staged"
    verify_sha "$staged_guard" "$EXPECTED_GUARD_SHA" "node-journal-guard staged"

    if ! "$staged_node" -v >/dev/null 2>&1; then
        log_error "xboard-node staged binary failed version check"
        exit 1
    fi
    if ! "$staged_guard" -h >/dev/null 2>&1; then
        log_error "node-journal-guard staged binary failed help check"
        exit 1
    fi
}

warn_v2bx() {
    local found=0
    if systemctl list-unit-files 'v2bx*.service' --no-legend 2>/dev/null | grep -qi .; then found=1; fi
    if pgrep -af 'V2bX|v2bx' >/dev/null 2>&1; then found=1; fi
    if [ "$found" -eq 1 ]; then
        log_warn "V2bX appears to be installed/running. This installer will NOT stop it automatically; decide manually before production use."
    fi
}

backup_existing() {
    local ts backup_path
    ts="$(date +%Y%m%d-%H%M%S)"
    backup_path="${BACKUP_ROOT}/xboard-node.backup-${ts}"
    mkdir -p "$backup_path"
    local copied=0
    for item in "$NODE_BINARY" "$GUARD_BINARY" "$CONFIG_FILE" "$CREDENTIALS_FILE" "$TOKEN_FILE" "$NODE_SERVICE_PATH" "$GUARD_SERVICE_PATH"; do
        if [ -e "$item" ]; then
            cp -a "$item" "$backup_path/"
            copied=1
        fi
    done
    if [ "$copied" -eq 1 ]; then
        log_info "Backup saved: $backup_path"
    else
        rmdir "$backup_path"
        log_info "No existing install files to backup"
    fi
}

install_binaries() {
    log_step "Installing binaries"
    install -m 755 "$TMP_DIR/xboard-node" "$NODE_BINARY"
    install -m 755 "$TMP_DIR/node-journal-guard" "$GUARD_BINARY"
    verify_sha "$NODE_BINARY" "$EXPECTED_NODE_SHA" "xboard-node installed"
    verify_sha "$GUARD_BINARY" "$EXPECTED_GUARD_SHA" "node-journal-guard installed"
}

write_config_if_missing() {
    mkdir -p "$INSTALL_ROOT"
    chmod 700 "$INSTALL_ROOT"
    if [ -f "$CONFIG_FILE" ]; then
        log_info "Preserving existing config: $CONFIG_FILE"
        return
    fi
    log_step "Creating config: $CONFIG_FILE"
    if [ "$MODE" = "machine" ]; then
        cat >"$CONFIG_FILE" <<EOF_CONFIG
kernel:
  type: "${KERNEL_TYPE}"
log:
  level: "info"
instances:
  - panel:
      url: "${PANEL_URL}"
    machine:
      machine_id: ${MACHINE_ID}
      token_env: "XBOARD_MACHINE_TOKEN"
EOF_CONFIG
        umask 077
        printf 'XBOARD_MACHINE_TOKEN=%s\n' "$TOKEN" > "$CREDENTIALS_FILE"
        chmod 600 "$CREDENTIALS_FILE"
    else
        cat >"$CONFIG_FILE" <<EOF_CONFIG
kernel:
  type: "${KERNEL_TYPE}"
log:
  level: "info"
panel:
  url: "${PANEL_URL}"
  token_env: "XBOARD_API_KEY"
  node_id: ${NODE_ID}
EOF_CONFIG
        umask 077
        printf 'XBOARD_API_KEY=%s\n' "$TOKEN" > "$CREDENTIALS_FILE"
        chmod 600 "$CREDENTIALS_FILE"
    fi
    chmod 600 "$CONFIG_FILE"
}

write_token_file() {
    log_step "Writing token file: $TOKEN_FILE"
    mkdir -p "$INSTALL_ROOT"
    umask 077
    printf '%s' "$TOKEN" > "$TOKEN_FILE"
    chmod 600 "$TOKEN_FILE"
}

write_services() {
    log_step "Writing systemd services"
    local env_line="EnvironmentFile=-${CREDENTIALS_FILE}"
    if [ -f "$NODE_SERVICE_PATH" ] && grep -q '^EnvironmentFile=' "$NODE_SERVICE_PATH"; then
        env_line="$(grep -m1 '^EnvironmentFile=' "$NODE_SERVICE_PATH")"
        log_info "Preserving existing xboard-node EnvironmentFile line"
    fi
    cat >"$NODE_SERVICE_PATH" <<EOF_UNIT
[Unit]
Description=Xboard Node Backend
Documentation=https://github.com/cedar2025/xboard-node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_ROOT}
${env_line}
ExecStart=${NODE_BINARY} -c ${CONFIG_FILE}
Restart=always
RestartSec=5
LimitNOFILE=1048576
NoNewPrivileges=true
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF_UNIT

    cat >"$GUARD_SERVICE_PATH" <<EOF_UNIT
[Unit]
Description=Node Journal Guard
After=network-online.target ${NODE_SERVICE}
Wants=network-online.target

[Service]
Type=simple
ExecStart=${GUARD_BINARY} -panel ${PANEL_URL} -node-id ${NODE_ID} -token-file ${TOKEN_FILE} -unit xboard-node -transport tcp_http -since now -dedup 60s
Restart=always
RestartSec=5
NoNewPrivileges=true
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF_UNIT

    chmod 644 "$NODE_SERVICE_PATH" "$GUARD_SERVICE_PATH"
    systemctl daemon-reload
    systemctl enable "$NODE_SERVICE" "$GUARD_SERVICE" >/dev/null
}

restart_and_verify() {
    if [ "$RESTART_SERVICES" -ne 1 ]; then
        log_warn "--no-restart set; skipping service restart/active checks"
        return
    fi
    log_step "Restarting services"
    systemctl restart "$NODE_SERVICE"
    systemctl restart "$GUARD_SERVICE"

    log_step "Verifying services"
    systemctl is-active --quiet "$NODE_SERVICE"
    systemctl is-active --quiet "$GUARD_SERVICE"
    log_info "${NODE_SERVICE} active"
    log_info "${GUARD_SERVICE} active"

    if [ -n "$VERIFY_PORT" ]; then
        local port_ok=0
        for _ in $(seq 1 30); do
            if command -v ss >/dev/null 2>&1 && ss -H -ltn | awk -v port=":${VERIFY_PORT}" '{ if ($4 == "*" port || $4 ~ port "$") found=1 } END { exit found ? 0 : 1 }'; then
                port_ok=1
                break
            fi
            sleep 1
        done
        if [ "$port_ok" -eq 1 ]; then
            log_info "Listen port ${VERIFY_PORT} detected"
        else
            log_error "Listen port ${VERIFY_PORT} not detected"
            ss -ltn || true
            exit 1
        fi
    else
        log_warn "Listen port verification skipped (pass --verify-port PORT to enforce)."
    fi

    local unknown_count
    unknown_count="$(journalctl -u "$NODE_SERVICE" -u "$GUARD_SERVICE" --since '2 minutes ago' --no-pager 2>/dev/null | grep -ci 'unknown version' || true)"
    if [ "$unknown_count" != "0" ]; then
        log_error "Found ${unknown_count} recent 'unknown version' log lines"
        exit 1
    fi
    log_info "No recent 'unknown version' logs"
}

print_summary() {
    cat <<EOF_SUMMARY

${GREEN}Install summary${NC}
  panel: ${PANEL_URL}
  mode: ${MODE}
  machine_id: ${MACHINE_ID:-n/a}
  node_id: ${NODE_ID}
  config: ${CONFIG_FILE} ($([ -f "$CONFIG_FILE" ] && echo present || echo missing))
  token_file: ${TOKEN_FILE} (token hidden)
  xboard-node: ${NODE_BINARY} sha=$(sha256_file "$NODE_BINARY" | cut -c1-8)...
  node-journal-guard: ${GUARD_BINARY} sha=$(sha256_file "$GUARD_BINARY" | cut -c1-8)...
  services: ${NODE_SERVICE}, ${GUARD_SERVICE}
  restart: $([ "$RESTART_SERVICES" -eq 1 ] && echo done || echo skipped)
EOF_SUMMARY
}

main() {
    parse_args "$@"
    require_root
    validate_args
    read_token
    warn_v2bx
    stage_binaries
    backup_existing
    install_binaries
    write_config_if_missing
    write_token_file
    write_services
    restart_and_verify
    print_summary
}

main "$@"

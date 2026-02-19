#!/usr/bin/env bash
set -euo pipefail

# NOCWALL SBC bootstrap/deploy script.
# Strategy:
# - Keep Docker Compose stack as source of truth.
# - Use this script as an operator wrapper for install/deploy/update/status/logs.
# - Support extension containers via compose fragments in extensions.d/.

APP_NAME="${APP_NAME:-nocwall}"
INSTALL_DIR="${INSTALL_DIR:-/opt/nocwall}"
STACK_FILE="${STACK_FILE:-$INSTALL_DIR/stack.yml}"
ENV_FILE="${ENV_FILE:-$INSTALL_DIR/.env}"
EXT_DIR="${EXT_DIR:-$INSTALL_DIR/extensions.d}"
PROJECT_NAME="${PROJECT_NAME:-nocwall}"

# Defaults for CE deployment on SBC.
NOCWALL_IMAGE="${NOCWALL_IMAGE:-predheadtx/nocwall:latest}"
NOCWALL_API_IMAGE="${NOCWALL_API_IMAGE:-predheadtx/nocwall-api:latest}"
TZ_VAL="${TZ_VAL:-America/Chicago}"
HTTP_PORT="${HTTP_PORT:-8088}"
GOTIFY_PORT="${GOTIFY_PORT:-18080}"
API_PORT="${API_PORT:-8080}"
UISP_URL="${UISP_URL:-}"
UISP_TOKEN="${UISP_TOKEN:-}"
GOTIFY_DEFAULTUSER_NAME="${GOTIFY_DEFAULTUSER_NAME:-admin}"
GOTIFY_DEFAULTUSER_PASS="${GOTIFY_DEFAULTUSER_PASS:-change_me_now}"
SHOW_TLS_UI="${SHOW_TLS_UI:-false}"
API_TOKEN="${API_TOKEN:-}"

# Runtime flags.
SUITE_PROFILE=0
PRO_PROFILE=0

usage() {
  cat <<'EOF'
Usage:
  ./NOCWALL.sh /install [options]
  ./NOCWALL.sh /init [options]
  ./NOCWALL.sh /deploy [options]
  ./NOCWALL.sh /update [options]
  ./NOCWALL.sh /status [options]
  ./NOCWALL.sh /logs [service]
  ./NOCWALL.sh /start [options]
  ./NOCWALL.sh /stop [options]
  ./NOCWALL.sh /down [options]
  ./NOCWALL.sh /help

Options:
  --dir <path>            Install directory (default: /opt/nocwall)
  --http-port <port>      CE web HTTP bind port (default: 8088)
  --gotify-port <port>    Embedded Gotify bind port (default: 18080)
  --api-port <port>       API bind port (suite profile; default: 8080)
  --image <image>         CE web image (default: predheadtx/nocwall:latest)
  --api-image <image>     API image (suite profile; default: predheadtx/nocwall-api:latest)
  --tz <tz>               Timezone (default: America/Chicago)
  --uisp-url <url>        Optional server-level UISP fallback URL
  --uisp-token <token>    Optional server-level UISP fallback token
  --show-tls-ui <bool>    SHOW_TLS_UI (default: false)
  --gotify-user <name>    Embedded Gotify admin username
  --gotify-pass <pass>    Embedded Gotify admin password
  --api-token <token>     Optional API bearer token for suite API container
  --suite                 Enable 'suite' compose profile (starts API service)
  --pro                   Enable 'pro' compose profile (for extension containers)
  -h, --help              Show this help

Examples:
  ./NOCWALL.sh /install
  ./NOCWALL.sh /init --http-port 8088
  ./NOCWALL.sh /deploy
  ./NOCWALL.sh /deploy --suite
  ./NOCWALL.sh /deploy --suite --pro
  ./NOCWALL.sh /logs nocwall

Extension protocol:
  Add *.yml compose fragments under:
    /opt/nocwall/extensions.d/
  Then run:
    ./NOCWALL.sh /deploy --pro
EOF
}

log() {
  printf '[NOCWALL] %s\n' "$*"
}

die() {
  printf '[NOCWALL][ERROR] %s\n' "$*" >&2
  exit 1
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

run_as_root() {
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    "$@"
  elif have_cmd sudo; then
    sudo "$@"
  else
    die "Root privileges required for: $* (install sudo or run as root)"
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dir)
        INSTALL_DIR="$2"
        STACK_FILE="$INSTALL_DIR/stack.yml"
        ENV_FILE="$INSTALL_DIR/.env"
        EXT_DIR="$INSTALL_DIR/extensions.d"
        shift 2
        ;;
      --http-port)
        HTTP_PORT="$2"; shift 2 ;;
      --gotify-port)
        GOTIFY_PORT="$2"; shift 2 ;;
      --api-port)
        API_PORT="$2"; shift 2 ;;
      --image)
        NOCWALL_IMAGE="$2"; shift 2 ;;
      --api-image)
        NOCWALL_API_IMAGE="$2"; shift 2 ;;
      --tz)
        TZ_VAL="$2"; shift 2 ;;
      --uisp-url)
        UISP_URL="$2"; shift 2 ;;
      --uisp-token)
        UISP_TOKEN="$2"; shift 2 ;;
      --show-tls-ui)
        SHOW_TLS_UI="$2"; shift 2 ;;
      --gotify-user)
        GOTIFY_DEFAULTUSER_NAME="$2"; shift 2 ;;
      --gotify-pass)
        GOTIFY_DEFAULTUSER_PASS="$2"; shift 2 ;;
      --api-token)
        API_TOKEN="$2"; shift 2 ;;
      --suite)
        SUITE_PROFILE=1; shift ;;
      --pro)
        PRO_PROFILE=1; shift ;;
      -h|--help)
        usage; exit 0 ;;
      *)
        break ;;
    esac
  done
  REMAINDER=("$@")
}

ensure_dirs() {
  run_as_root mkdir -p "$INSTALL_DIR" "$EXT_DIR"
}

write_env_file_if_missing() {
  if [[ -f "$ENV_FILE" ]]; then
    log "Keeping existing env file: $ENV_FILE"
    return
  fi
  cat >"$ENV_FILE" <<EOF
NOCWALL_IMAGE=$NOCWALL_IMAGE
NOCWALL_API_IMAGE=$NOCWALL_API_IMAGE
TZ=$TZ_VAL
HTTP_PORT=$HTTP_PORT
GOTIFY_PORT=$GOTIFY_PORT
API_PORT=$API_PORT
UISP_URL=$UISP_URL
UISP_TOKEN=$UISP_TOKEN
SHOW_TLS_UI=$SHOW_TLS_UI
GOTIFY_DEFAULTUSER_NAME=$GOTIFY_DEFAULTUSER_NAME
GOTIFY_DEFAULTUSER_PASS=$GOTIFY_DEFAULTUSER_PASS
API_TOKEN=$API_TOKEN
EOF
  log "Wrote env file: $ENV_FILE"
}

write_stack_file_if_missing() {
  if [[ -f "$STACK_FILE" ]]; then
    log "Keeping existing stack file: $STACK_FILE"
    return
  fi

  cat >"$STACK_FILE" <<'EOF'
name: nocwall

services:
  nocwall:
    image: ${NOCWALL_IMAGE:-predheadtx/nocwall:latest}
    container_name: NOCWALL
    restart: unless-stopped
    ports:
      - "${HTTP_PORT:-8088}:80"
      - "${GOTIFY_PORT:-18080}:18080"
    environment:
      TZ: ${TZ:-America/Chicago}
      UISP_URL: ${UISP_URL:-}
      UISP_TOKEN: ${UISP_TOKEN:-}
      GOTIFY_DEFAULTUSER_NAME: ${GOTIFY_DEFAULTUSER_NAME:-admin}
      GOTIFY_DEFAULTUSER_PASS: ${GOTIFY_DEFAULTUSER_PASS:-change_me_now}
      SHOW_TLS_UI: ${SHOW_TLS_UI:-false}
    volumes:
      - nocwall_cache:/var/www/html/cache
    networks:
      - nocwall_net

  nocwall-api:
    image: ${NOCWALL_API_IMAGE:-predheadtx/nocwall-api:latest}
    container_name: NOCWALL-API
    restart: unless-stopped
    profiles: ["suite"]
    ports:
      - "${API_PORT:-8080}:8080"
    environment:
      TZ: ${TZ:-America/Chicago}
      API_ADDR: :8080
      APP_ENV: prod
      API_TOKEN: ${API_TOKEN:-}
      DATA_FILE: /data/store.json
      UISP_URL: ${UISP_URL:-}
      UISP_TOKEN: ${UISP_TOKEN:-}
      UISP_DEVICES_PATH: /nms/api/v2.1/devices
      UISP_POLL_INTERVAL_SEC: 0
      UISP_POLL_RETRIES: 1
    volumes:
      - nocwall_api_data:/data
    networks:
      - nocwall_net

volumes:
  nocwall_cache:
  nocwall_api_data:

networks:
  nocwall_net:
    name: nocwall_net
EOF

  log "Wrote stack file: $STACK_FILE"
}

write_extension_readme_if_missing() {
  local ext_readme="$EXT_DIR/README.txt"
  if [[ -f "$ext_readme" ]]; then
    return
  fi
  cat >"$ext_readme" <<'EOF'
Place extension compose fragments (*.yml) in this directory.

Recommended:
- Keep CE in the base stack.
- Put PRO features in extension containers with profile "pro".
- Deploy with: ./NOCWALL.sh /deploy --pro

Example service block:
services:
  nocwall-pro-example:
    image: your-private-registry/nocwall-pro-example:latest
    profiles: ["pro"]
    restart: unless-stopped
    networks:
      - nocwall_net
EOF
  log "Wrote extension docs: $ext_readme"
}

install_docker() {
  if have_cmd docker; then
    log "Docker already installed."
  else
    if have_cmd apt-get; then
      run_as_root apt-get update
      run_as_root apt-get install -y docker.io docker-compose-plugin ca-certificates curl
    elif have_cmd dnf; then
      run_as_root dnf install -y docker docker-compose-plugin ca-certificates curl
    elif have_cmd yum; then
      run_as_root yum install -y docker docker-compose-plugin ca-certificates curl
    else
      die "Unsupported package manager. Install Docker + Compose plugin manually."
    fi
  fi

  run_as_root systemctl enable docker || true
  run_as_root systemctl start docker || true

  if have_cmd usermod && [[ "${EUID:-$(id -u)}" -eq 0 ]] && [[ -n "${SUDO_USER:-}" ]]; then
    run_as_root usermod -aG docker "$SUDO_USER" || true
    log "Added $SUDO_USER to docker group (re-login may be required)."
  fi
}

compose_files_args() {
  local args=()
  args+=(-f "$STACK_FILE")
  if [[ -d "$EXT_DIR" ]]; then
    local f
    shopt -s nullglob
    for f in "$EXT_DIR"/*.yml; do
      args+=(-f "$f")
    done
    shopt -u nullglob
  fi
  printf '%s\n' "${args[@]}"
}

compose_run() {
  have_cmd docker || die "docker command not found"
  docker info >/dev/null 2>&1 || die "docker daemon not reachable"

  local profiles=()
  if [[ "$SUITE_PROFILE" == "1" ]]; then
    profiles+=("suite")
  fi
  if [[ "$PRO_PROFILE" == "1" ]]; then
    profiles+=("pro")
  fi

  local profile_env=""
  if [[ "${#profiles[@]}" -gt 0 ]]; then
    local joined=""
    local p
    for p in "${profiles[@]}"; do
      if [[ -n "$joined" ]]; then
        joined="$joined,$p"
      else
        joined="$p"
      fi
    done
    profile_env="$joined"
  fi

  local files=()
  while IFS= read -r line; do
    files+=("$line")
  done < <(compose_files_args)

  if [[ -n "$profile_env" ]]; then
    COMPOSE_PROFILES="$profile_env" docker compose \
      --project-name "$PROJECT_NAME" \
      --env-file "$ENV_FILE" \
      "${files[@]}" \
      "${REMAINDER[@]}"
  else
    docker compose \
      --project-name "$PROJECT_NAME" \
      --env-file "$ENV_FILE" \
      "${files[@]}" \
      "${REMAINDER[@]}"
  fi
}

cmd_install() {
  install_docker
  ensure_dirs
  write_env_file_if_missing
  write_stack_file_if_missing
  write_extension_readme_if_missing
  log "Install complete."
}

cmd_init() {
  ensure_dirs
  write_env_file_if_missing
  write_stack_file_if_missing
  write_extension_readme_if_missing
  log "Init complete."
}

cmd_deploy() {
  cmd_init
  REMAINDER=(up -d --pull always)
  compose_run
  log "Deploy complete."
}

cmd_update() {
  cmd_init
  REMAINDER=(pull)
  compose_run
  REMAINDER=(up -d)
  compose_run
  log "Update complete."
}

cmd_status() {
  cmd_init
  REMAINDER=(ps)
  compose_run
}

cmd_logs() {
  cmd_init
  local service="${1:-}"
  if [[ -n "$service" ]]; then
    REMAINDER=(logs --tail=200 -f "$service")
  else
    REMAINDER=(logs --tail=200 -f)
  fi
  compose_run
}

cmd_start() {
  cmd_init
  REMAINDER=(start)
  compose_run
}

cmd_stop() {
  cmd_init
  REMAINDER=(stop)
  compose_run
}

cmd_down() {
  cmd_init
  REMAINDER=(down)
  compose_run
}

main() {
  local action="${1:-}"
  [[ -z "$action" ]] && { usage; exit 1; }
  shift || true
  parse_args "$@"

  case "$action" in
    /install|install) cmd_install ;;
    /init|init) cmd_init ;;
    /deploy|deploy) cmd_deploy ;;
    /update|update) cmd_update ;;
    /status|status|/ps|ps) cmd_status ;;
    /logs|logs) cmd_logs "${REMAINDER[0]:-}" ;;
    /start|start) cmd_start ;;
    /stop|stop) cmd_stop ;;
    /down|down) cmd_down ;;
    /help|help|-h|--help) usage ;;
    *)
      die "Unknown action: $action"
      ;;
  esac
}

main "$@"

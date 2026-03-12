#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://localhost}"
COOKIE_FILE="${COOKIE_FILE:-/tmp/nocwall_closed_beta_v1.cookies}"
NOCWALL_USER="${NOCWALL_USER:-cb_v1_smoke_$(date +%s)}"
NOCWALL_PASS="${NOCWALL_PASS:-Password12345}"
UISP_TEST_URL="${UISP_TEST_URL:-}"
UISP_TEST_TOKEN="${UISP_TEST_TOKEN:-}"
CISCO_TEST_URL="${CISCO_TEST_URL:-}"
CISCO_TEST_TOKEN="${CISCO_TEST_TOKEN:-}"
JUNIPER_TEST_URL="${JUNIPER_TEST_URL:-}"
JUNIPER_TEST_TOKEN="${JUNIPER_TEST_TOKEN:-}"
MERAKI_TEST_URL="${MERAKI_TEST_URL:-}"
MERAKI_TEST_TOKEN="${MERAKI_TEST_TOKEN:-}"
GENERIC_TEST_URL="${GENERIC_TEST_URL:-}"
GENERIC_TEST_TOKEN="${GENERIC_TEST_TOKEN:-}"
GENERIC_TEST_API_PATH="${GENERIC_TEST_API_PATH:-/feeds/generic/devices}"
GENERIC_TEST_AUTH_SCHEME="${GENERIC_TEST_AUTH_SCHEME:-none}"
MOCK_UISP="${MOCK_UISP:-1}"
MOCK_UISP_PORT="${MOCK_UISP_PORT:-18091}"
MOCK_UISP_CONTAINER="${MOCK_UISP_CONTAINER:-nocwall-cb4-uisp-mock-$$}"
MOCK_UISP_SOURCE_URL="${MOCK_UISP_SOURCE_URL:-http://host.docker.internal:${MOCK_UISP_PORT}}"
MOCK_UISP_HEALTH_URL="${MOCK_UISP_HEALTH_URL:-http://127.0.0.1:${MOCK_UISP_PORT}}"
DRY_RUN="${DRY_RUN:-0}"
MOCK_UISP_TMP=""
MOCK_UISP_STARTED=0

run_cmd() {
  if [ "$DRY_RUN" = "1" ]; then
    printf '[dry-run] %s\n' "$*"
    return 0
  fi
  # shellcheck disable=SC2086
  sh -c "$*"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "$1" >&2
    exit 1
  fi
}

cleanup() {
  if [ "$MOCK_UISP_STARTED" = "1" ] && command -v docker >/dev/null 2>&1; then
    docker rm -f "$MOCK_UISP_CONTAINER" >/dev/null 2>&1 || true
  fi
  if [ -n "$MOCK_UISP_TMP" ] && [ -d "$MOCK_UISP_TMP" ]; then
    rm -rf "$MOCK_UISP_TMP"
  fi
  rm -f "$COOKIE_FILE"
}

start_mock_uisp() {
  require_cmd docker
  require_cmd mktemp
  require_cmd mkdir
  require_cmd chmod
  require_cmd sleep

  MOCK_UISP_TMP="$(mktemp -d 2>/dev/null || mktemp -d -t nocwall_cb4_mock)"
  mkdir -p "$MOCK_UISP_TMP/nms/api/v2.1"
  mkdir -p "$MOCK_UISP_TMP/api/v1"
  mkdir -p "$MOCK_UISP_TMP/api/v1/cisco"
  mkdir -p "$MOCK_UISP_TMP/api/v1/juniper"
  mkdir -p "$MOCK_UISP_TMP/api/v1/organizations/mock-org/devices"
  mkdir -p "$MOCK_UISP_TMP/feeds/generic"
  cat > "$MOCK_UISP_TMP/nms/api/v2.1/devices" <<'JSON'
[{"identification":{"id":"mock-device-1"},"name":"Mock UISP Gateway","overview":{"status":"online"}},{"identification":{"id":"mock-device-2"},"name":"Mock UISP Access Point","overview":{"status":"offline"}}]
JSON
  cat > "$MOCK_UISP_TMP/api/v1/devices" <<'JSON'
[{"id":"mock-nms-core-1","name":"Mock NMS Core","siteId":"mock-site","status":"online","latencyMs":6},{"id":"mock-nms-edge-1","name":"Mock NMS Edge","siteId":"mock-site","status":"offline","latencyMs":144}]
JSON
  cat > "$MOCK_UISP_TMP/api/v1/cisco/devices" <<'JSON'
[{"id":"mock-cisco-router-1","name":"Mock Cisco Router","role":"router","siteName":"Mock Cisco Site","status":"online","ip":"198.51.100.10","latencyMs":5,"vendor":"Cisco","model":"ISR"}]
JSON
  cat > "$MOCK_UISP_TMP/api/v1/juniper/devices" <<'JSON'
[{"id":"mock-juniper-gateway-1","name":"Mock Juniper Gateway","role":"gateway","siteName":"Mock Juniper Site","status":"offline","ip":"198.51.100.11","latencyMs":155,"vendor":"Juniper","model":"SRX"}]
JSON
  cat > "$MOCK_UISP_TMP/api/v1/organizations/mock-org/devices/statuses" <<'JSON'
[{"name":"Mock Meraki MX","serial":"Q2XX-AAAA-0001","networkId":"mock-meraki-net","status":"online","productType":"appliance","model":"MX95","lanIp":"198.51.100.14"},{"name":"Mock Meraki MS","serial":"Q2XX-BBBB-0002","networkId":"mock-meraki-net","status":"offline","productType":"switch","model":"MS250","lanIp":"198.51.100.15"}]
JSON
  cat > "$MOCK_UISP_TMP/feeds/generic/devices" <<'JSON'
{"devices":[{"id":"mock-generic-edge-1","name":"Mock Generic Edge","role":"firewall","site":{"name":"Mock Generic Site"},"status":"online","ipAddress":"198.51.100.12","latencyMs":8,"manufacturer":"AnyVendor","model":"Edge Appliance"}]}
JSON
  cat > "$MOCK_UISP_TMP/devices" <<'JSON'
{"devices":[{"id":"mock-generic-default-1","name":"Mock Generic Default","role":"router","site":{"name":"Mock Generic Site"},"status":"online","ipAddress":"198.51.100.13","latencyMs":7,"manufacturer":"AnyVendor","model":"Edge Appliance"}]}
JSON
  chmod -R a+rX "$MOCK_UISP_TMP"

  docker rm -f "$MOCK_UISP_CONTAINER" >/dev/null 2>&1 || true
  docker run -d --rm --name "$MOCK_UISP_CONTAINER" -p "127.0.0.1:${MOCK_UISP_PORT}:80" -v "$MOCK_UISP_TMP:/usr/share/nginx/html:ro" nginx:alpine >/dev/null
  MOCK_UISP_STARTED=1

  UISP_TEST_URL="$MOCK_UISP_SOURCE_URL"
  if [ -z "$UISP_TEST_TOKEN" ]; then
    UISP_TEST_TOKEN="mock-token-123456"
  fi
  if [ -z "$CISCO_TEST_URL" ]; then
    CISCO_TEST_URL="$MOCK_UISP_SOURCE_URL"
  fi
  if [ -z "$CISCO_TEST_TOKEN" ]; then
    CISCO_TEST_TOKEN="mock-token-123456"
  fi
  if [ -z "$JUNIPER_TEST_URL" ]; then
    JUNIPER_TEST_URL="$MOCK_UISP_SOURCE_URL"
  fi
  if [ -z "$JUNIPER_TEST_TOKEN" ]; then
    JUNIPER_TEST_TOKEN="mock-token-123456"
  fi
  if [ -z "$MERAKI_TEST_URL" ]; then
    MERAKI_TEST_URL="$MOCK_UISP_SOURCE_URL/api/v1/organizations/mock-org"
  fi
  if [ -z "$MERAKI_TEST_TOKEN" ]; then
    MERAKI_TEST_TOKEN="mock-token-123456"
  fi
  if [ -z "$GENERIC_TEST_URL" ]; then
    GENERIC_TEST_URL="$MOCK_UISP_SOURCE_URL"
  fi

  attempts=0
  while [ "$attempts" -lt 20 ]; do
    if curl -sS "$MOCK_UISP_HEALTH_URL/nms/api/v2.1/devices" >/dev/null 2>&1 && \
       curl -sS "$MOCK_UISP_HEALTH_URL/api/v1/devices" >/dev/null 2>&1 && \
       curl -sS "$MOCK_UISP_HEALTH_URL/api/v1/cisco/devices" >/dev/null 2>&1 && \
       curl -sS "$MOCK_UISP_HEALTH_URL/api/v1/juniper/devices" >/dev/null 2>&1 && \
       curl -sS "$MOCK_UISP_HEALTH_URL/api/v1/organizations/mock-org/devices/statuses" >/dev/null 2>&1 && \
       curl -sS "$MOCK_UISP_HEALTH_URL/feeds/generic/devices" >/dev/null 2>&1; then
      return 0
    fi
    attempts=$((attempts + 1))
    sleep 1
  done

  printf '[fail] local mock UISP did not become ready at %s\n' "$MOCK_UISP_HEALTH_URL" >&2
  exit 1
}

run_source_onboarding_test() {
  source_type="$1"
  source_label="$2"
  source_name="$3"
  source_url="$4"
  source_token="$5"
  source_api_path="${6:-}"
  source_auth_scheme="${7:-}"

  csrf_src="$(fetch_dashboard_csrf)"
  src_save="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" -X POST "$BASE_URL/?ajax=sources_save" -H "X-CSRF-Token: $csrf_src" \
    -d "_csrf=$csrf_src" \
    --data-urlencode "type=$source_type" \
    --data-urlencode "name=$source_name" \
    --data-urlencode "url=$source_url" \
    --data-urlencode "token=$source_token" \
    --data-urlencode "api_path=$source_api_path" \
    --data-urlencode "auth_scheme=$source_auth_scheme" \
    -d "enabled=1")"
  check_contains "$src_save" '"ok":1' "$source_label source save succeeded"

  src_id="$(printf '%s' "$src_save" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | sed -n '1p')"
  if [ -z "$src_id" ]; then
    printf '[fail] unable to parse source id for %s\n' "$source_label" >&2
    exit 1
  fi

  csrf_test="$(fetch_dashboard_csrf)"
  src_test="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" -X POST "$BASE_URL/?ajax=sources_test" -H "X-CSRF-Token: $csrf_test" \
    -d "_csrf=$csrf_test" \
    --data-urlencode "id=$src_id")"
  case "$src_test" in
    *'"ok":true'*) printf '[ok] %s source connectivity test passed\n' "$source_label" ;;
    *)
      printf '[fail] %s source connectivity test passed\n' "$source_label" >&2
      printf 'Response: %s\n' "$src_test" >&2
      exit 1
      ;;
  esac
}

fetch_wallboard_devices() {
  curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=devices"
}

check_contains() {
  haystack="$1"
  needle="$2"
  msg="$3"
  case "$haystack" in
    *"$needle"*) printf '[ok] %s\n' "$msg" ;;
    *)
      printf '[fail] %s\n' "$msg" >&2
      printf 'Expected to find: %s\n' "$needle" >&2
      exit 1
      ;;
  esac
}

extract_csrf_any() {
  page="$1"
  token="$(printf '%s' "$page" | sed -n 's/.*name="_csrf"[[:space:]]\+value="\([^"]*\)".*/\1/p' | sed -n '1p')"
  if [ -z "$token" ]; then
    token="$(printf '%s' "$page" | sed -n 's/.*const csrfToken = \"\([^\"]*\)\".*/\1/p' | sed -n '1p')"
  fi
  printf '%s' "$token"
}

fetch_login_csrf() {
  page="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?login=1")"
  token="$(extract_csrf_any "$page")"
  if [ -z "$token" ]; then
    printf '[fail] unable to extract csrf token from login page\n' >&2
    exit 1
  fi
  printf '%s' "$token"
}

fetch_dashboard_page() {
  curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/"
}

fetch_dashboard_csrf() {
  page="$(fetch_dashboard_page)"
  token="$(extract_csrf_any "$page")"
  if [ -z "$token" ]; then
    printf '[fail] unable to extract csrf token from dashboard page\n' >&2
    exit 1
  fi
  printf '%s' "$token"
}

require_cmd curl
require_cmd sed
require_cmd tr
trap cleanup EXIT INT TERM

printf 'Running Closed Beta v1 smoke test against %s\n' "$BASE_URL"
rm -f "$COOKIE_FILE"

if [ "$DRY_RUN" = "1" ]; then
  printf '[dry-run] skipping live HTTP assertions\n'
  exit 0
fi

printf '\n[CB1] account create/login/session\n'
csrf_register="$(fetch_login_csrf)"
run_cmd "curl -sS -L -c '$COOKIE_FILE' -b '$COOKIE_FILE' -X POST '$BASE_URL/?action=register' -d '_csrf=$csrf_register' -d 'username=$NOCWALL_USER&password=$NOCWALL_PASS&password_confirm=$NOCWALL_PASS' >/dev/null"

dash_after_register="$(fetch_dashboard_page)"
check_contains "$dash_after_register" "User: $NOCWALL_USER" 'register flow lands on authenticated dashboard'

csrf_auth="$(fetch_dashboard_csrf)"
run_cmd "curl -sS -L -c '$COOKIE_FILE' -b '$COOKIE_FILE' '$BASE_URL/?action=logout&_csrf=$csrf_auth' >/dev/null"

csrf_login="$(fetch_login_csrf)"
run_cmd "curl -sS -L -c '$COOKIE_FILE' -b '$COOKIE_FILE' -X POST '$BASE_URL/?action=login' -d '_csrf=$csrf_login' -d 'username=$NOCWALL_USER&password=$NOCWALL_PASS' >/dev/null"

dash_after_login="$(fetch_dashboard_page)"
check_contains "$dash_after_login" "User: $NOCWALL_USER" 'login flow restores authenticated session'

printf '\n[CB2] dashboard load/wallboard baseline\n'
check_contains "$dash_after_login" "source-status-strip" 'dashboard renders source status strip'
check_contains "$dash_after_login" "Kiosk" 'dashboard renders kiosk wallboard control'

printf '\n[CB3] settings persistence/manageability\n'
settings_page="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?view=settings")"
check_contains "$settings_page" "Account Settings" 'settings page renders'
check_contains "$settings_page" "Redeem Beta Key" 'settings includes beta redemption control'
check_contains "$settings_page" "Add NMS Source" 'settings includes source management'
check_contains "$settings_page" "Source Type" 'settings includes source type selector'

prefs_get="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=prefs_get")"
check_contains "$prefs_get" '"ok":1' 'preferences fetch endpoint available'

csrf_prefs="$(fetch_dashboard_csrf)"
prefs_save="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" -X POST "$BASE_URL/?ajax=prefs_save" -H "X-CSRF-Token: $csrf_prefs" \
  -d "_csrf=$csrf_prefs" \
  --data-urlencode 'dashboard_settings={"density":"normal","default_tab":"gateways","sort_mode":"manual","group_mode":"none","refresh_interval":"normal","theme_preset":"classic","font_scale_preset":"normal","browser_notifications":false,"alert_sound_profile":"default","metrics":{"cpu":true,"ram":true,"temp":true,"latency":true,"uptime":true,"outage":true}}')"
check_contains "$prefs_save" '"ok":1' 'preferences save endpoint available'

printf '\n[CB5] CE free baseline (no beta key required)\n'
billing_status="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=billing_status")"
check_contains "$billing_status" '"ok":1' 'billing status endpoint available'
check_contains "$billing_status" '"entitlement_source":"ce"' 'entitlement source remains CE without beta key'

printf '\n[CB4] source onboarding/diagnostics\n'
cb4_mode="live"
if [ -z "$UISP_TEST_URL" ] || [ -z "$UISP_TEST_TOKEN" ]; then
  if [ "$MOCK_UISP" = "1" ]; then
    if command -v docker >/dev/null 2>&1; then
      printf '[info] CB4 live UISP credentials not provided; starting local mock UISP on port %s\n' "$MOCK_UISP_PORT"
      start_mock_uisp
      printf '[info] using mock source URL %s\n' "$UISP_TEST_URL"
      cb4_mode="mock"
    else
      printf '[skip] CB4 live creds missing and Docker is unavailable for local mock startup\n'
      cb4_mode="skip"
    fi
  else
    printf '[skip] CB4 requires UISP_TEST_URL + UISP_TEST_TOKEN, or set MOCK_UISP=1 for local mock validation\n'
    cb4_mode="skip"
  fi
fi

if [ "$cb4_mode" != "skip" ] && [ -n "$UISP_TEST_URL" ] && [ -n "$UISP_TEST_TOKEN" ]; then
  run_source_onboarding_test "uisp" "UISP" "Smoke UISP" "$UISP_TEST_URL" "$UISP_TEST_TOKEN"

  src_list="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=sources_list")"
  check_contains "$src_list" '"ok":1' 'source list endpoint available'

  diag="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=sources_diagnostics")"
  check_contains "$diag" '"ok":1' 'source diagnostics endpoint returned success'
  check_contains "$diag" '"type":"uisp"' 'diagnostics include UISP source type'
  if [ "$cb4_mode" = "mock" ]; then
    printf '[ok] CB4 validation completed with local mock UISP source\n'
  else
    printf '[ok] CB4 validation completed with provided UISP credentials\n'
  fi
fi

printf '\n[CBC8] Cisco and Juniper source onboarding from settings UI\n'
cbc8_mode="ready"
if [ -z "$CISCO_TEST_URL" ] || [ -z "$CISCO_TEST_TOKEN" ] || [ -z "$JUNIPER_TEST_URL" ] || [ -z "$JUNIPER_TEST_TOKEN" ]; then
  if [ "$MOCK_UISP" = "1" ] && [ "$MOCK_UISP_STARTED" = "1" ]; then
    cbc8_mode="ready"
  elif [ "$MOCK_UISP" = "1" ] && command -v docker >/dev/null 2>&1; then
    printf '[info] CBC8 live Cisco/Juniper credentials not provided; starting local mock NMS source\n'
    start_mock_uisp
    printf '[info] using mock Cisco URL %s and mock Juniper URL %s\n' "$CISCO_TEST_URL" "$JUNIPER_TEST_URL"
    cbc8_mode="ready"
  else
    printf '[skip] CBC8 requires Cisco/Juniper test creds, or set MOCK_UISP=1 with Docker available\n'
    cbc8_mode="skip"
  fi
fi

if [ "$cbc8_mode" = "ready" ]; then
  run_source_onboarding_test "cisco" "Cisco" "Smoke Cisco" "$CISCO_TEST_URL" "$CISCO_TEST_TOKEN" "/api/v1/cisco/devices" "bearer"
  run_source_onboarding_test "juniper" "Juniper" "Smoke Juniper" "$JUNIPER_TEST_URL" "$JUNIPER_TEST_TOKEN" "/api/v1/juniper/devices" "x-auth-token"
  diag_multi="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=sources_diagnostics")"
  check_contains "$diag_multi" '"type":"cisco"' 'diagnostics include Cisco source type'
  check_contains "$diag_multi" '"type":"juniper"' 'diagnostics include Juniper source type'
  printf '[ok] CBC8 source onboarding checks completed for Cisco + Juniper\n'
fi

printf '\n[CBC9] Generic HTTP source onboarding + wallboard ingestion\n'
cbc9_mode="ready"
if [ -z "$GENERIC_TEST_URL" ]; then
  if [ "$MOCK_UISP" = "1" ] && [ "$MOCK_UISP_STARTED" = "1" ]; then
    GENERIC_TEST_URL="$MOCK_UISP_SOURCE_URL"
  elif [ "$MOCK_UISP" = "1" ] && command -v docker >/dev/null 2>&1; then
    printf '[info] CBC9 live generic feed not provided; starting local mock Generic HTTP feed\n'
    start_mock_uisp
    printf '[info] using mock Generic HTTP URL %s\n' "$GENERIC_TEST_URL"
  else
    printf '[skip] CBC9 requires GENERIC_TEST_URL, or set MOCK_UISP=1 with Docker available\n'
    cbc9_mode="skip"
  fi
fi

if [ "$cbc9_mode" = "ready" ]; then
  run_source_onboarding_test "generic" "Generic HTTP" "Smoke Generic" "$GENERIC_TEST_URL" "$GENERIC_TEST_TOKEN" "$GENERIC_TEST_API_PATH" "$GENERIC_TEST_AUTH_SCHEME"
  diag_generic="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=sources_diagnostics")"
  check_contains "$diag_generic" '"type":"generic"' 'diagnostics include Generic HTTP source type'

  wallboard_devices="$(fetch_wallboard_devices)"
  check_contains "$wallboard_devices" '"http":200' 'wallboard device feed returns success with multi-source monitoring'
  check_contains "$wallboard_devices" '"name":"Mock Cisco Router"' 'wallboard includes Cisco router device'
  check_contains "$wallboard_devices" '"source_type":"cisco"' 'wallboard includes Cisco source attribution'
  check_contains "$wallboard_devices" '"name":"Mock Juniper Gateway"' 'wallboard includes Juniper gateway device'
  check_contains "$wallboard_devices" '"source_type":"juniper"' 'wallboard includes Juniper source attribution'
  check_contains "$wallboard_devices" '"name":"Mock Generic Edge"' 'wallboard includes Generic HTTP device'
  check_contains "$wallboard_devices" '"source_type":"generic"' 'wallboard includes Generic HTTP source attribution'
  printf '[ok] CBC9 generic source onboarding and wallboard ingestion checks completed\n'
fi

printf '\n[CBC10] Meraki source onboarding + wallboard ingestion\n'
cbc10_mode="ready"
if [ -z "$MERAKI_TEST_URL" ] || [ -z "$MERAKI_TEST_TOKEN" ]; then
  if [ "$MOCK_UISP" = "1" ] && [ "$MOCK_UISP_STARTED" = "1" ]; then
    cbc10_mode="ready"
  elif [ "$MOCK_UISP" = "1" ] && command -v docker >/dev/null 2>&1; then
    printf '[info] CBC10 live Meraki credentials not provided; starting local mock Meraki source\n'
    start_mock_uisp
    printf '[info] using mock Meraki URL %s\n' "$MERAKI_TEST_URL"
    cbc10_mode="ready"
  else
    printf '[skip] CBC10 requires MERAKI_TEST_URL + MERAKI_TEST_TOKEN, or set MOCK_UISP=1 with Docker available\n'
    cbc10_mode="skip"
  fi
fi

if [ "$cbc10_mode" = "ready" ]; then
  run_source_onboarding_test "meraki" "Meraki" "Smoke Meraki" "$MERAKI_TEST_URL" "$MERAKI_TEST_TOKEN" "/devices/statuses" "x-cisco-meraki-api-key"
  diag_meraki="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=sources_diagnostics")"
  check_contains "$diag_meraki" '"type":"meraki"' 'diagnostics include Meraki source type'

  wallboard_meraki="$(fetch_wallboard_devices)"
  check_contains "$wallboard_meraki" '"name":"Mock Meraki MX"' 'wallboard includes Meraki appliance device'
  check_contains "$wallboard_meraki" '"source_type":"meraki"' 'wallboard includes Meraki source attribution'
  printf '[ok] CBC10 Meraki source onboarding and wallboard ingestion checks completed\n'
fi

printf '\nClosed Beta v1 smoke checks completed.\n'
printf 'Optional follow-up: rerun with real UISP/Cisco/Juniper/Meraki/Generic HTTP endpoints for live connector verification.\n'

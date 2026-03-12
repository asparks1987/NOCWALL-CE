#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://localhost}"
COOKIE_FILE="${COOKIE_FILE:-/tmp/nocwall_ce_smoke.cookies}"
NOCWALL_USER="${NOCWALL_USER:-ce_smoke_$(date +%s)}"
NOCWALL_PASS="${NOCWALL_PASS:-Password12345}"
DRY_RUN="${DRY_RUN:-0}"

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

fetch_csrf_token() {
  page="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?login=1")"
  token="$(printf '%s' "$page" | sed -n 's/.*name="_csrf"[[:space:]]\+value="\([^"]*\)".*/\1/p' | sed -n '1p')"
  if [ -z "$token" ]; then
    printf '[fail] unable to extract csrf token from login page\n' >&2
    exit 1
  fi
  printf '%s' "$token"
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

check_not_contains() {
  haystack="$1"
  needle="$2"
  msg="$3"
  case "$haystack" in
    *"$needle"*)
      printf '[fail] %s\n' "$msg" >&2
      printf 'Did not expect to find: %s\n' "$needle" >&2
      exit 1
      ;;
    *) printf '[ok] %s\n' "$msg" ;;
  esac
}

require_cmd curl
require_cmd sed
require_cmd tr

printf 'Running CE feature-pack smoke test against %s\n' "$BASE_URL"
rm -f "$COOKIE_FILE"

csrf_register="$(fetch_csrf_token)"
run_cmd "curl -sS -L -c '$COOKIE_FILE' -b '$COOKIE_FILE' -X POST '$BASE_URL/?action=register' -d '_csrf=$csrf_register' -d 'username=$NOCWALL_USER&password=$NOCWALL_PASS&password_confirm=$NOCWALL_PASS' >/dev/null"
csrf_login="$(fetch_csrf_token)"
run_cmd "curl -sS -L -c '$COOKIE_FILE' -b '$COOKIE_FILE' -X POST '$BASE_URL/?action=login' -d '_csrf=$csrf_login' -d 'username=$NOCWALL_USER&password=$NOCWALL_PASS' >/dev/null"

if [ "$DRY_RUN" = "1" ]; then
  printf '[dry-run] skipped endpoint assertions\n'
  exit 0
fi

dashboard_page="$(curl -sS -L -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/")"
check_not_contains "$dashboard_page" 'data-tab="topology"' 'CE dashboard hides topology tab'
check_not_contains "$dashboard_page" 'Incident Commander Workspace' 'CE dashboard hides incident workspace'

billing_status="$(curl -sS -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=billing_status")"
check_contains "$billing_status" '"ok":1' 'billing status endpoint available'
check_contains "$billing_status" '"strict_ce":true' 'billing status reports strict CE mode'
check_contains "$billing_status" '"topology":false' 'billing status disables topology for CE'
check_contains "$billing_status" '"inventory":false' 'billing status disables inventory for CE'
check_contains "$billing_status" '"simulate":false' 'billing status disables simulate for CE'
check_contains "$billing_status" '"mobile":false' 'billing status disables mobile for CE'
check_contains "$billing_status" '"agents":false' 'billing status disables agents for CE'

csrf_demo="$(fetch_csrf_token)"
demo_on="$(curl -sS -c "$COOKIE_FILE" -b "$COOKIE_FILE" -X POST "$BASE_URL/?ajax=demo_mode_set" -H "X-CSRF-Token: $csrf_demo" -d "_csrf=$csrf_demo" -d "enabled=1")"
check_contains "$demo_on" '"ok":1' 'demo mode enable'
check_contains "$demo_on" '"enabled":true' 'demo mode true'

demo_get="$(curl -sS -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=demo_mode_get")"
check_contains "$demo_get" '"ok":1' 'demo mode read'

prefs_get="$(curl -sS -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=prefs_get")"
check_contains "$prefs_get" '"ok":1' 'preferences fetch'

csrf_prefs="$(fetch_csrf_token)"
prefs_save="$(curl -sS -c "$COOKIE_FILE" -b "$COOKIE_FILE" -X POST "$BASE_URL/?ajax=prefs_save" -H "X-CSRF-Token: $csrf_prefs" \
  -d "_csrf=$csrf_prefs" \
  --data-urlencode 'dashboard_settings={"density":"normal","default_tab":"topology","sort_mode":"manual","group_mode":"none","refresh_interval":"normal","theme_preset":"high_contrast","font_scale_preset":"large","browser_notifications":false,"alert_sound_profile":"soft","metrics":{"cpu":true,"ram":true,"temp":true,"latency":true,"uptime":true,"outage":true}}')"
check_contains "$prefs_save" '"ok":1' 'preferences save'
check_not_contains "$prefs_save" '"default_tab":"topology"' 'CE preferences reject topology as default tab'
check_contains "$prefs_save" '"default_tab":"gateways"' 'CE preferences fall back to gateways default tab'

prefs_get_after="$(curl -sS -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=prefs_get")"
check_contains "$prefs_get_after" '"default_tab":"gateways"' 'CE preference fetch keeps non-PRO default tab sanitized'

diag="$(curl -sS -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=sources_diagnostics")"
check_contains "$diag" '"ok":1' 'source diagnostics endpoint'

whats_new="$(curl -sS -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=whats_new")"
check_contains "$whats_new" '"ok":1' 'whats_new endpoint'
check_contains "$whats_new" '"notes"' 'whats_new notes payload'

pro_topology="$(curl -sS -o /tmp/nocwall_ce_topology.json -w "%{http_code}" -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=topology_overview")"
if [ "$pro_topology" != "403" ]; then
  printf '[fail] CE topology endpoint must be denied\n' >&2
  exit 1
fi
check_contains "$(cat /tmp/nocwall_ce_topology.json)" '"error":"pro_feature_required"' 'CE topology endpoint denied'

pro_inventory="$(curl -sS -o /tmp/nocwall_ce_inventory.json -w "%{http_code}" -c "$COOKIE_FILE" -b "$COOKIE_FILE" "$BASE_URL/?ajax=inventory_overview")"
if [ "$pro_inventory" != "403" ]; then
  printf '[fail] CE inventory endpoint must be denied\n' >&2
  exit 1
fi
check_contains "$(cat /tmp/nocwall_ce_inventory.json)" '"error":"pro_feature_required"' 'CE inventory endpoint denied'

csrf_sim="$(fetch_csrf_token)"
pro_simulate="$(curl -sS -o /tmp/nocwall_ce_simulate.json -w "%{http_code}" -c "$COOKIE_FILE" -b "$COOKIE_FILE" -X POST "$BASE_URL/?ajax=simulate&id=ce-smoke-device" -H "X-CSRF-Token: $csrf_sim" -d "_csrf=$csrf_sim")"
if [ "$pro_simulate" != "403" ]; then
  printf '[fail] CE simulate endpoint must be denied\n' >&2
  exit 1
fi
check_contains "$(cat /tmp/nocwall_ce_simulate.json)" '"error":"pro_feature_required"' 'CE simulate endpoint denied'

printf '\nCE smoke endpoints passed.\n'
printf 'Manual UI checks: see docs/ce_feature_pack_test_matrix.md\n'

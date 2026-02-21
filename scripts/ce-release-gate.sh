#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

fail=0

echo "[ce-gate] Running CE release guardrails..."

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

collect_paths() {
  local paths=(api assets web index.php docker-compose.yml Dockerfile start.sh .github)
  local out=()
  local p
  for p in "${paths[@]}"; do
    [[ -e "$p" ]] && out+=("$p")
  done
  printf '%s\n' "${out[@]}"
}

mapfile -t SCAN_PATHS < <(collect_paths)

# 1) Block suspicious filenames in CE repo.
blocked_files=""
while IFS= read -r file; do
  base="$(basename "$file")"
  lower="$(printf '%s' "$base" | tr '[:upper:]' '[:lower:]')"
  if [[ "$lower" == *pro_* ]]; then
    blocked_files+="$file"$'\n'
    continue
  fi
  if [[ "$lower" =~ (^|[_-])(billing|license|rbac|sso|tenant|scim)([_-]|$) ]]; then
    blocked_files+="$file"$'\n'
  fi
done < <(find . -type f \
  ! -path './.git/*' \
  ! -path './android/.gradle/*' \
  ! -path './node_modules/*' \
  ! -path './.idea/*')

if [[ -n "$blocked_files" ]]; then
  echo "[ce-gate] Blocked filenames detected:" >&2
  printf '%s' "$blocked_files" >&2
  fail=1
fi

# 2) Block imports/references to private namespaces from CE code.
if has_cmd rg; then
  pro_refs="$(rg -n --hidden --glob '!.git/**' --glob '*.go' --glob '*.php' --glob '*.js' --glob '*.ts' --glob '*.tsx' --glob '*.py' '(uisp_noc_pro|nocwall_pro|uisp-noc-pro)' "${SCAN_PATHS[@]}" || true)"
else
  pro_refs="$(grep -RInE '(uisp_noc_pro|nocwall_pro|uisp-noc-pro)' "${SCAN_PATHS[@]}" --include='*.go' --include='*.php' --include='*.js' --include='*.ts' --include='*.tsx' --include='*.py' || true)"
fi
if [[ -n "$pro_refs" ]]; then
  echo "[ce-gate] Private namespace references found:" >&2
  printf '%s\n' "$pro_refs" >&2
  fail=1
fi

# 3) Simple hardcoded secret sweep (non-example files).
secret_pattern="(API_KEY|TOKEN|SECRET|PASSWORD)[[:space:]]*[:=][[:space:]]*[\"']?[A-Za-z0-9._-]{16,}"
if has_cmd rg; then
  secret_hits="$(rg -n --hidden --glob '!.git/**' --glob '!*.md' --glob '!.env.example' "$secret_pattern" "${SCAN_PATHS[@]}" || true)"
else
  secret_hits="$(grep -RInE "$secret_pattern" "${SCAN_PATHS[@]}" --exclude='*.md' --exclude='.env.example' || true)"
fi
secret_hits="$(printf '%s\n' "$secret_hits" | grep -v '\${' | sed '/^$/d' || true)"
if [[ -n "$secret_hits" ]]; then
  echo "[ce-gate] Potential hardcoded secrets found:" >&2
  printf '%s\n' "$secret_hits" >&2
  fail=1
fi

# 4) CE profile must remain default in compose.
if [[ -f docker-compose.yml ]]; then
  if ! grep -qE 'NOCWALL_FEATURE_PROFILE:[[:space:]]*\$\{NOCWALL_FEATURE_PROFILE:-ce\}' docker-compose.yml; then
    echo "[ce-gate] docker-compose default must remain CE profile." >&2
    fail=1
  fi
fi

if [[ "$fail" -ne 0 ]]; then
  echo "[ce-gate] FAILED" >&2
  exit 1
fi

echo "[ce-gate] PASSED"

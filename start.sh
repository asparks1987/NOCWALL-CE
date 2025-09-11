#!/usr/bin/env bash
set -euo pipefail

# Ensure cache paths and permissions (handles mounted volumes)
mkdir -p /var/www/html/cache /var/www/html/cache/gotify
chown -R www-data:www-data /var/www/html/cache || true
chmod -R u+rwX,g+rwX /var/www/html/cache || true

# Start Gotify server in background if present
if command -v gotify >/dev/null 2>&1; then
  echo "[start] Launching embedded Gotify server on :18080"
  (
    # Pass through default user env if provided by compose
    export GOTIFY_DEFAULTUSER_NAME="${GOTIFY_DEFAULTUSER_NAME:-}"
    export GOTIFY_DEFAULTUSER_PASS="${GOTIFY_DEFAULTUSER_PASS:-}"
    exec gotify --config /etc/gotify/config.yml
  ) >/var/log/gotify.log 2>&1 &

  # Zero-touch bootstrap: create/connect app token automatically
  (
    set +e
    G_URL="${GOTIFY_URL:-http://127.0.0.1:18080}"
    G_USER="${GOTIFY_DEFAULTUSER_NAME:-admin}"
    G_PASS="${GOTIFY_DEFAULTUSER_PASS:-changeme}"
    TOKEN_FILE="/var/www/html/cache/gotify_app_token.txt"
    APP_NAME="UISP NOC"

    # Skip if token already present
    if [ -s "$TOKEN_FILE" ]; then
      echo "[gotify] Existing token file detected; skipping bootstrap"
      exit 0
    fi

    echo "[gotify] Waiting for Gotify API at ${G_URL}"
    for i in $(seq 1 60); do
      curl -fsS -u "$G_USER:$G_PASS" "$G_URL/application" >/dev/null 2>&1 && READY=1 && break
      sleep 1
    done
    if [ "${READY:-0}" != "1" ]; then
      echo "[gotify] WARNING: Could not reach Gotify API with default admin creds; skipping bootstrap"
      exit 0
    fi

    # Try to create the application (preferred: returns token)
    CREATE_JSON=$(curl -sS -u "$G_USER:$G_PASS" -H 'Content-Type: application/json' \
      -d "{\"name\":\"${APP_NAME}\",\"description\":\"UISP-NOC alerts\"}" \
      "$G_URL/application")
    TOKEN=$(printf '%s' "$CREATE_JSON" | php -r '$j=json_decode(stream_get_contents(STDIN),true); if(is_array($j) && isset($j["token"])) echo $j["token"];')

    if [ -z "$TOKEN" ]; then
      # Likely already exists; find the application id by name
      LIST_JSON=$(curl -sS -u "$G_USER:$G_PASS" "$G_URL/application")
      APP_ID=$(printf '%s' "$LIST_JSON" | php -r '$a=json_decode(stream_get_contents(STDIN),true); if(is_array($a)){foreach($a as $x){if(isset($x["name"]) && $x["name"]==="UISP NOC" && isset($x["id"])) {echo $x["id"]; break;}}} ')
      if [ -n "$APP_ID" ]; then
        # Try to generate a fresh token for the existing application
        REGEN_JSON=$(curl -sS -u "$G_USER:$G_PASS" -X POST "$G_URL/application/$APP_ID/token")
        TOKEN=$(printf '%s' "$REGEN_JSON" | php -r '$j=json_decode(stream_get_contents(STDIN),true); if(is_array($j) && isset($j["token"])) echo $j["token"];')
        if [ -z "$TOKEN" ]; then
          # Fallback: delete + recreate to obtain a token
          curl -sS -u "$G_USER:$G_PASS" -X DELETE "$G_URL/application/$APP_ID" >/dev/null 2>&1
          CREATE_JSON=$(curl -sS -u "$G_USER:$G_PASS" -H 'Content-Type: application/json' \
            -d "{\"name\":\"${APP_NAME}\",\"description\":\"UISP-NOC alerts\"}" \
            "$G_URL/application")
          TOKEN=$(printf '%s' "$CREATE_JSON" | php -r '$j=json_decode(stream_get_contents(STDIN),true); if(is_array($j) && isset($j["token"])) echo $j["token"];')
        fi
      fi
    fi

    if [ -n "$TOKEN" ]; then
      echo "$TOKEN" > "$TOKEN_FILE"
      chmod 600 "$TOKEN_FILE"
      echo "[gotify] Application token provisioned and stored"
    else
      echo "[gotify] WARNING: Failed to provision application token"
    fi
  ) &
else
  echo "[start] Gotify binary not found; skipping embedded notifications server"
fi

# Launch Apache in foreground
echo "[start] Launching Apache"
exec apache2-foreground

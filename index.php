<?php
error_reporting(E_ALL);
ini_set('display_errors', 1);
session_start();

date_default_timezone_set('America/Chicago');

// Config
$UISP_URL   = getenv("UISP_URL") ?: "https://changeme.unmsapp.com";
$UISP_TOKEN = getenv("UISP_TOKEN") ?: "changeme";
$NOCWALL_API_URL = rtrim((string)(getenv("NOCWALL_API_URL") ?: getenv("API_BASE_URL") ?: "http://api:8080"), "/");
$NOCWALL_API_TOKEN = (string)(getenv("API_TOKEN") ?: "");
$NOCWALL_FEATURE_PROFILE = strtolower(trim((string)(getenv("NOCWALL_FEATURE_PROFILE") ?: "ce")));
$NOCWALL_PRO_FEATURES = in_array($NOCWALL_FEATURE_PROFILE, ['pro','full','dev'], true);
$NOCWALL_PRO_OVERRIDE = strtolower(trim((string)getenv("NOCWALL_PRO_FEATURES")));
if(in_array($NOCWALL_PRO_OVERRIDE, ['1','true','yes','on'], true)){
    $NOCWALL_PRO_FEATURES = true;
} elseif(in_array($NOCWALL_PRO_OVERRIDE, ['0','false','no','off'], true)){
    $NOCWALL_PRO_FEATURES = false;
}
$NOCWALL_STRICT_CE = !$NOCWALL_PRO_FEATURES;
$NOCWALL_STRICT_OVERRIDE = strtolower(trim((string)getenv("NOCWALL_STRICT_CE")));
if(in_array($NOCWALL_STRICT_OVERRIDE, ['1','true','yes','on'], true)){
    $NOCWALL_STRICT_CE = true;
} elseif(in_array($NOCWALL_STRICT_OVERRIDE, ['0','false','no','off'], true)){
    $NOCWALL_STRICT_CE = false;
}
$NOCWALL_FEATURE_FLAGS = [
    'profile' => $NOCWALL_FEATURE_PROFILE,
    'strict_ce' => $NOCWALL_STRICT_CE,
    'pro_features' => $NOCWALL_PRO_FEATURES,
    'advanced_metrics' => !$NOCWALL_STRICT_CE && $NOCWALL_PRO_FEATURES,
    'advanced_actions' => $NOCWALL_PRO_FEATURES,
    'display_controls' => !$NOCWALL_STRICT_CE && $NOCWALL_PRO_FEATURES,
    'inventory' => $NOCWALL_PRO_FEATURES,
    'topology' => $NOCWALL_PRO_FEATURES,
    'history' => $NOCWALL_PRO_FEATURES,
    'ack' => $NOCWALL_PRO_FEATURES,
    'simulate' => $NOCWALL_PRO_FEATURES,
    'cpe_history' => $NOCWALL_PRO_FEATURES
];
// Feature flags / UI toggles
$SHOW_TLS_UI = in_array(strtolower((string)getenv('SHOW_TLS_UI')), ['1','true','yes'], true);

// Embedded Gotify (notifications)
$GOTIFY_URL   = getenv('GOTIFY_URL') ?: 'http://127.0.0.1:18080';
$GOTIFY_TOKEN = getenv('GOTIFY_TOKEN');
if(!$GOTIFY_TOKEN){
    $tokFile = __DIR__ . '/cache/gotify_app_token.txt';
    if(is_file($tokFile)){
        $t = trim(@file_get_contents($tokFile));
        if($t !== '') $GOTIFY_TOKEN = $t;
    }
}

$CACHE_DIR  = __DIR__ . "/cache";
$CACHE_FILE = $CACHE_DIR . "/status_cache.json";
$DB_FILE    = $CACHE_DIR . "/metrics.sqlite";
$AUTH_FILE  = $CACHE_DIR . "/auth.json";
$USERS_FILE = $CACHE_DIR . "/users.json";

$FIRST_OFFLINE_THRESHOLD = 30;
$FLAP_ALERT_THRESHOLD = 3;
$FLAP_ALERT_WINDOW = 900;
$FLAP_ALERT_SUPPRESS = 1800;
$LATENCY_ALERT_THRESHOLD = 200;
$LATENCY_ALERT_SUPPRESS = 900;
$LATENCY_ALERT_WINDOW = 900;
$LATENCY_ALERT_STREAK = 3;

// Ensure cache dir and basic permissions
if (!is_dir($CACHE_DIR)) @mkdir($CACHE_DIR, 0775, true);
if (!is_writable($CACHE_DIR)) @chmod($CACHE_DIR, 0775);

function read_json_file($file){
    if(!is_file($file)) return null;
    $raw = @file_get_contents($file);
    if($raw === false || trim($raw) === '') return null;
    $json = json_decode($raw, true);
    return is_array($json) ? $json : null;
}

function write_json_file($file, $data){
    $json = json_encode($data, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
    if($json === false) return false;
    return @file_put_contents($file, $json, LOCK_EX) !== false;
}

function normalize_username($value){
    return strtolower(trim((string)$value));
}

function default_dashboard_settings(){
    return [
        'density' => 'normal',
        'default_tab' => 'gateways',
        'sort_mode' => 'manual',
        'metrics' => [
            'cpu' => true,
            'ram' => true,
            'temp' => true,
            'latency' => true,
            'uptime' => true,
            'outage' => true
        ]
    ];
}

function normalize_dashboard_settings($input){
    $base = default_dashboard_settings();
    if(!is_array($input)) return $base;
    if(isset($input['density']) && $input['density'] === 'compact'){
        $base['density'] = 'compact';
    }
    $defaultTab = trim((string)($input['default_tab'] ?? ''));
    if(in_array($defaultTab, ['gateways','aps','routers','topology'], true)){
        $base['default_tab'] = $defaultTab;
    }
    $sortMode = trim((string)($input['sort_mode'] ?? ''));
    if(in_array($sortMode, ['manual','status_name','name_asc','last_seen_desc'], true)){
        $base['sort_mode'] = $sortMode;
    }
    if(isset($input['metrics']) && is_array($input['metrics'])){
        foreach(array_keys($base['metrics']) as $k){
            if(array_key_exists($k, $input['metrics'])){
                $base['metrics'][$k] = !!$input['metrics'][$k];
            }
        }
    }
    return $base;
}

function normalize_ap_siren_prefs($input){
    $out = [];
    if(!is_array($input)) return $out;
    foreach($input as $k => $v){
        $id = trim((string)$k);
        if($id === '' || strlen($id) > 180) continue;
        $out[$id] = !!$v;
    }
    return $out;
}

function default_tab_siren_prefs(){
    return [
        'gateways' => true,
        'aps' => false,
        'routers' => false
    ];
}

function normalize_tab_siren_prefs($input){
    $base = default_tab_siren_prefs();
    if(!is_array($input)) return $base;
    foreach(array_keys($base) as $k){
        if(array_key_exists($k, $input)){
            $base[$k] = !!$input[$k];
        }
    }
    return $base;
}

function default_card_order_prefs(){
    return [
        'gateways' => [],
        'aps' => [],
        'routers' => []
    ];
}

function normalize_card_order_prefs($input){
    $base = default_card_order_prefs();
    if(!is_array($input)) return $base;
    foreach(array_keys($base) as $k){
        if(!array_key_exists($k, $input) || !is_array($input[$k])) continue;
        $seen = [];
        $out = [];
        foreach($input[$k] as $id){
            $v = trim((string)$id);
            if($v === '' || strlen($v) > 180) continue;
            if(isset($seen[$v])) continue;
            $seen[$v] = true;
            $out[] = $v;
            if(count($out) >= 500) break;
        }
        $base[$k] = $out;
    }
    return $base;
}

function default_user_preferences(){
    return [
        'dashboard_settings' => default_dashboard_settings(),
        'ap_siren_prefs' => [],
        'tab_siren_prefs' => default_tab_siren_prefs(),
        'card_order_prefs' => default_card_order_prefs()
    ];
}

function normalize_user_preferences($input){
    $base = default_user_preferences();
    if(!is_array($input)) return $base;
    $base['dashboard_settings'] = normalize_dashboard_settings($input['dashboard_settings'] ?? null);
    $base['ap_siren_prefs'] = normalize_ap_siren_prefs($input['ap_siren_prefs'] ?? null);
    $base['tab_siren_prefs'] = normalize_tab_siren_prefs($input['tab_siren_prefs'] ?? null);
    $base['card_order_prefs'] = normalize_card_order_prefs($input['card_order_prefs'] ?? null);
    return $base;
}

function normalize_source_status_entry($row){
    if(!is_array($row)) return null;
    $id = trim((string)($row['id'] ?? ''));
    if($id === '') return null;
    $name = trim((string)($row['name'] ?? $id));
    $url = normalize_uisp_url($row['url'] ?? '');
    $lastPollAt = trim((string)($row['last_poll_at'] ?? ''));
    $httpCode = isset($row['http']) ? (int)$row['http'] : 0;
    if($httpCode < 0) $httpCode = 0;
    $latency = isset($row['latency_ms']) ? (int)$row['latency_ms'] : 0;
    if($latency < 0) $latency = 0;
    $deviceCount = isset($row['device_count']) ? (int)$row['device_count'] : 0;
    if($deviceCount < 0) $deviceCount = 0;
    $error = trim((string)($row['error'] ?? ''));
    return [
        'id' => $id,
        'name' => ($name !== '' ? $name : $id),
        'url' => $url,
        'ok' => !empty($row['ok']),
        'http' => $httpCode,
        'latency_ms' => $latency,
        'device_count' => $deviceCount,
        'error' => $error,
        'last_poll_at' => $lastPollAt
    ];
}

function normalize_user_source_status($input){
    $out = [];
    if(!is_array($input)) return $out;
    foreach($input as $row){
        $normalized = normalize_source_status_entry($row);
        if(!$normalized) continue;
        $out[] = $normalized;
    }
    return $out;
}

function get_user_source_status_map($user){
    $out = [];
    if(!is_array($user)) return $out;
    $rows = normalize_user_source_status($user['source_status'] ?? []);
    foreach($rows as $row){
        $out[$row['id']] = $row;
    }
    return $out;
}

function summarize_source_status_rows($rows){
    $summary = [
        'total' => 0,
        'healthy' => 0,
        'failed' => 0,
        'never_polled' => 0,
        'last_poll_at' => null
    ];
    if(!is_array($rows)) return $summary;
    $summary['total'] = count($rows);
    foreach($rows as $row){
        if(!is_array($row)) continue;
        if(!empty($row['last_poll_at'])){
            $ts = strtotime((string)$row['last_poll_at']);
            if($ts){
                if(empty($summary['last_poll_at']) || $ts > strtotime((string)$summary['last_poll_at'])){
                    $summary['last_poll_at'] = date('c', $ts);
                }
            }
        }
        if(empty($row['last_poll_at'])){
            $summary['never_polled']++;
            continue;
        }
        if(!empty($row['ok'])){
            $summary['healthy']++;
        } else {
            $summary['failed']++;
        }
    }
    return $summary;
}

function probe_uisp_source($src){
    $out = [
        'id' => (string)($src['id'] ?? ''),
        'name' => (string)($src['name'] ?? ''),
        'url' => normalize_uisp_url($src['url'] ?? ''),
        'ok' => false,
        'http' => 0,
        'latency_ms' => 0,
        'device_count' => 0,
        'error' => '',
        'last_poll_at' => date('c')
    ];
    $url = rtrim((string)$out['url'], '/');
    $token = trim((string)($src['token'] ?? ''));
    if($url === '' || $token === ''){
        $out['error'] = 'invalid_source';
        return $out;
    }
    $ch = curl_init();
    $start = microtime(true);
    curl_setopt_array($ch,[
        CURLOPT_URL => $url . '/nms/api/v2.1/devices',
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER => ['accept: application/json', 'x-auth-token: ' . $token],
        CURLOPT_TIMEOUT => 10
    ]);
    $resp = curl_exec($ch);
    $code = (int)curl_getinfo($ch, CURLINFO_HTTP_CODE);
    $lat = (int)round((microtime(true) - $start) * 1000);
    $err = (string)curl_error($ch);
    curl_close($ch);
    $rows = json_decode((string)$resp, true);
    $count = is_array($rows) ? count($rows) : 0;
    $ok = ($code >= 200 && $code < 300);
    $out['ok'] = $ok;
    $out['http'] = $code;
    $out['latency_ms'] = $lat;
    $out['device_count'] = $count;
    $out['error'] = ($ok ? '' : ($err !== '' ? $err : ('http_' . $code)));
    return $out;
}

function normalize_uisp_url($value){
    $url = trim((string)$value);
    if($url === '') return '';
    if(!preg_match('#^https?://#i', $url)){
        $url = 'https://' . $url;
    }
    return rtrim($url, '/');
}

function is_placeholder_uisp_url($url){
    $u = strtolower(trim((string)$url));
    if($u === '') return true;
    if(strpos($u, 'changeme') !== false) return true;
    if(strpos($u, 'example.unmsapp.com') !== false) return true;
    return false;
}

function generate_source_id(){
    try {
        return 'src_' . bin2hex(random_bytes(6));
    } catch (Exception $e) {
        return 'src_' . str_replace('.', '', uniqid('', true));
    }
}

function normalize_user_source($src, $fallbackName = 'UISP Source'){
    if(!is_array($src)) return null;
    $id = trim((string)($src['id'] ?? ''));
    if($id === '') $id = generate_source_id();
    $name = trim((string)($src['name'] ?? ''));
    if($name === '') $name = $fallbackName;
    $url = normalize_uisp_url($src['url'] ?? '');
    $token = trim((string)($src['token'] ?? ''));
    if($url === '' || $token === '') return null;
    $enabledRaw = $src['enabled'] ?? true;
    $enabled = !($enabledRaw === false || $enabledRaw === 0 || $enabledRaw === '0' || strtolower((string)$enabledRaw) === 'false');
    return [
        'id' => $id,
        'name' => $name,
        'url' => $url,
        'token' => $token,
        'enabled' => $enabled,
        'created_at' => (string)($src['created_at'] ?? date('c')),
        'updated_at' => (string)($src['updated_at'] ?? date('c'))
    ];
}

function get_stored_user_sources($user){
    $out = [];
    if(!is_array($user)) return $out;
    $sources = $user['sources'] ?? [];
    if(!is_array($sources)) return $out;
    foreach($sources as $src){
        $normalized = normalize_user_source($src);
        if($normalized) $out[] = $normalized;
    }
    return $out;
}

function get_effective_uisp_sources($user, $envUrl, $envToken){
    $sources = [];
    foreach(get_stored_user_sources($user) as $src){
        if(!empty($src['enabled'])) $sources[] = $src;
    }

    if(count($sources) > 0) return $sources;

    // Legacy compatibility fallback: account token + server UISP URL.
    $legacyToken = trim((string)($user['uisp_token'] ?? ''));
    $base = normalize_uisp_url($envUrl);
    if($legacyToken !== '' && $legacyToken !== 'changeme' && $base !== '' && !is_placeholder_uisp_url($base)){
        $sources[] = [
            'id' => 'legacy-account',
            'name' => 'Legacy Account UISP',
            'url' => $base,
            'token' => $legacyToken,
            'enabled' => true
        ];
    }
    if(count($sources) > 0) return $sources;

    // Server-wide fallback.
    $token = trim((string)$envToken);
    if($token !== '' && $token !== 'changeme' && $base !== '' && !is_placeholder_uisp_url($base)){
        $sources[] = [
            'id' => 'server-default',
            'name' => 'Server Default UISP',
            'url' => $base,
            'token' => $token,
            'enabled' => true
        ];
    }
    return $sources;
}

function default_admin_user(){
    $now = date('c');
    return [
        'username' => 'admin',
        'password_hash' => password_hash('admin', PASSWORD_DEFAULT),
        'uisp_token' => '',
        'sources' => [],
        'source_status' => [],
        'preferences' => default_user_preferences(),
        'created_at' => $now,
        'updated_at' => $now
    ];
}

function bootstrap_users_store($usersFile, $legacyAuthFile, $envUrl, $envToken){
    $store = read_json_file($usersFile);
    $didMutate = false;

    if(is_array($store) && isset($store['users']) && is_array($store['users']) && count($store['users']) > 0){
        foreach($store['users'] as $uname => $user){
            if(!is_array($user)) continue;
            if(!isset($store['users'][$uname]['sources']) || !is_array($store['users'][$uname]['sources'])){
                $store['users'][$uname]['sources'] = [];
                $didMutate = true;
            }
            $normalizedSourceStatus = normalize_user_source_status($store['users'][$uname]['source_status'] ?? null);
            if(json_encode($normalizedSourceStatus) !== json_encode($store['users'][$uname]['source_status'] ?? null)){
                $store['users'][$uname]['source_status'] = $normalizedSourceStatus;
                $didMutate = true;
            }
            $normalizedPrefs = normalize_user_preferences($store['users'][$uname]['preferences'] ?? null);
            if(json_encode($normalizedPrefs) !== json_encode($store['users'][$uname]['preferences'] ?? null)){
                $store['users'][$uname]['preferences'] = $normalizedPrefs;
                $didMutate = true;
            }
            // One-time migration path: legacy single token -> first source using server UISP URL.
            $legacyToken = trim((string)($store['users'][$uname]['uisp_token'] ?? ''));
            if($legacyToken !== '' && count(get_stored_user_sources($store['users'][$uname])) === 0){
                $base = normalize_uisp_url($envUrl);
                if($base !== '' && !is_placeholder_uisp_url($base)){
                    $store['users'][$uname]['sources'][] = [
                        'id' => generate_source_id(),
                        'name' => 'Primary UISP',
                        'url' => $base,
                        'token' => $legacyToken,
                        'enabled' => true,
                        'created_at' => date('c'),
                        'updated_at' => date('c')
                    ];
                    $didMutate = true;
                }
            }
            // Normalize source rows.
            $normalizedSources = get_stored_user_sources($store['users'][$uname]);
            if(json_encode($normalizedSources) !== json_encode($store['users'][$uname]['sources'])){
                $store['users'][$uname]['sources'] = $normalizedSources;
                $didMutate = true;
            }
        }
        if($didMutate){
            $store['updated_at'] = date('c');
            write_json_file($usersFile, $store);
        }
        return $store;
    }

    $users = [];
    $legacy = read_json_file($legacyAuthFile);
    if(is_array($legacy) && !empty($legacy['username']) && !empty($legacy['password_hash'])){
        $username = normalize_username($legacy['username']);
        $now = date('c');
        $migratedSources = [];
        $legacyBase = normalize_uisp_url($envUrl);
        if($legacyBase !== '' && !is_placeholder_uisp_url($legacyBase) && trim((string)$envToken) !== '' && trim((string)$envToken) !== 'changeme'){
            $migratedSources[] = [
                'id' => generate_source_id(),
                'name' => 'Primary UISP',
                'url' => $legacyBase,
                'token' => trim((string)$envToken),
                'enabled' => true,
                'created_at' => $now,
                'updated_at' => $now
            ];
        }
        $users[$username] = [
            'username' => $username,
            'password_hash' => (string)$legacy['password_hash'],
            'uisp_token' => '',
            'sources' => $migratedSources,
            'source_status' => [],
            'preferences' => default_user_preferences(),
            'created_at' => (string)($legacy['created_at'] ?? $legacy['updated_at'] ?? $now),
            'updated_at' => (string)($legacy['updated_at'] ?? $now)
        ];
    } else {
        $admin = default_admin_user();
        $users[$admin['username']] = $admin;
    }

    $store = ['users' => $users, 'updated_at' => date('c')];
    write_json_file($usersFile, $store);
    return $store;
}

function save_users_store($usersFile, &$store){
    $store['updated_at'] = date('c');
    return write_json_file($usersFile, $store);
}

function get_user_by_username($store, $username){
    $u = normalize_username($username);
    if($u === '') return null;
    if(!isset($store['users'][$u]) || !is_array($store['users'][$u])) return null;
    return $store['users'][$u];
}

function validate_username($username){
    return preg_match('/^[a-z0-9._-]{3,32}$/', $username) === 1;
}

function get_session_user($store){
    $u = normalize_username($_SESSION['auth_user'] ?? '');
    if($u === '') return null;
    return get_user_by_username($store, $u);
}

// Simple users store with legacy auth migration from auth.json.
$USERS_STORE = bootstrap_users_store($USERS_FILE, $AUTH_FILE, $UISP_URL, $UISP_TOKEN);

// Handle login/register/logout actions early
if(isset($_GET['action']) && $_GET['action']==='register' && $_SERVER['REQUEST_METHOD']==='POST'){
    $u = normalize_username($_POST['username'] ?? '');
    $p = (string)($_POST['password'] ?? '');
    $p2 = (string)($_POST['password_confirm'] ?? '');
    if(!validate_username($u)){
        $_SESSION['auth_err'] = 'Username must be 3-32 chars: a-z, 0-9, dot, underscore, hyphen.';
        header('Location: ./?login=1');
        exit;
    }
    if($p !== $p2){
        $_SESSION['auth_err'] = 'Password confirmation does not match.';
        header('Location: ./?login=1');
        exit;
    }
    if(strlen($p) < 8){
        $_SESSION['auth_err'] = 'Password must be at least 8 characters.';
        header('Location: ./?login=1');
        exit;
    }
    if(get_user_by_username($USERS_STORE, $u)){
        $_SESSION['auth_err'] = 'Username already exists.';
        header('Location: ./?login=1');
        exit;
    }
    $now = date('c');
    $USERS_STORE['users'][$u] = [
        'username' => $u,
        'password_hash' => password_hash($p, PASSWORD_DEFAULT),
        'uisp_token' => '',
        'sources' => [],
        'source_status' => [],
        'preferences' => default_user_preferences(),
        'created_at' => $now,
        'updated_at' => $now
    ];
    save_users_store($USERS_FILE, $USERS_STORE);
    $_SESSION['auth_ok'] = 1;
    $_SESSION['auth_user'] = $u;
    header('Location: ./');
    exit;
}
if(isset($_GET['action']) && $_GET['action']==='login' && $_SERVER['REQUEST_METHOD']==='POST'){
    $u = normalize_username($_POST['username'] ?? '');
    $p = (string)($_POST['password'] ?? '');
    $user = get_user_by_username($USERS_STORE, $u);
    if($user && password_verify($p, (string)($user['password_hash'] ?? ''))){
        $_SESSION['auth_ok'] = 1;
        $_SESSION['auth_user'] = $u;
        header('Location: ./');
        exit;
    } else {
        $_SESSION['auth_err'] = 'Invalid credentials';
        header('Location: ./?login=1');
        exit;
    }
}
if(isset($_GET['action']) && $_GET['action']==='logout'){
    session_destroy();
    header('Location: ./?login=1');
    exit;
}

// Legacy session fallback for prior single-user auth.
if(isset($_SESSION['auth_ok']) && empty($_SESSION['auth_user'])){
    if(isset($USERS_STORE['users']['admin'])){
        $_SESSION['auth_user'] = 'admin';
    } else {
        unset($_SESSION['auth_ok']);
    }
}

// For AJAX endpoints, require login except for a health or login check
function require_login_for_ajax(){
    global $USERS_STORE;
    if(!isset($_SESSION['auth_ok'])){
        http_response_code(401);
        header('Content-Type: application/json');
        echo json_encode(['error'=>'unauthorized']);
        exit;
    }
    $u = normalize_username($_SESSION['auth_user'] ?? '');
    if($u === '' || !isset($USERS_STORE['users'][$u])){
        http_response_code(401);
        header('Content-Type: application/json');
        echo json_encode(['error'=>'invalid_session']);
        exit;
    }
}

function require_pro_feature($featureKey){
    global $NOCWALL_FEATURE_FLAGS;
    $enabled = !empty($NOCWALL_FEATURE_FLAGS[$featureKey]);
    if($enabled){
        return;
    }
    http_response_code(403);
    header('Content-Type: application/json');
    echo json_encode([
        'ok' => 0,
        'error' => 'pro_feature_required',
        'message' => 'This feature is available in NOCWALL PRO.',
        'feature' => (string)$featureKey
    ]);
    exit;
}

// Load cache
$cache = file_exists($CACHE_FILE) ? json_decode(file_get_contents($CACHE_FILE), true) : [];
if (!is_array($cache)) $cache = [];

// SQLite init with robust error handling
try {
    if (!file_exists($DB_FILE)) {
        // Best-effort create the file so SQLite has a handle
        @touch($DB_FILE);
        @chmod($DB_FILE, 0664);
    }
    $db = new SQLite3($DB_FILE);
} catch (Exception $e) {
    http_response_code(500);
    header('Content-Type: text/plain');
    echo "Fatal: Unable to open SQLite database at: $DB_FILE\n";
    echo "Error: ".$e->getMessage()."\n\n";
    echo "Checks:\n";
    echo "- Ensure directory exists and is writable: $CACHE_DIR\n";
    echo "- If using a Docker volume, fix permissions (chown/chmod) for www-data.\n";
    echo "- Example inside container: chown -R www-data:www-data /var/www/html/cache && chmod -R u+rwX,g+rwX /var/www/html/cache\n";
    exit;
}
$db->exec('PRAGMA journal_mode = wal;');
$db->busyTimeout(5000);
$db->exec("CREATE TABLE IF NOT EXISTS metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT,
    name TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    cpu INTEGER,
    ram INTEGER,
    temp INTEGER,
    latency REAL,
    online INTEGER
)");
$db->exec("CREATE TABLE IF NOT EXISTS cpe_pings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT,
    name TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    latency REAL
)");
$db->exec("CREATE INDEX IF NOT EXISTS idx_cpe_pings_device_ts ON cpe_pings(device_id, timestamp)");

// Asset/version cache-busting
// Calculate a version based on the latest mtime of key files
$ASSET_VERSION = max(
    @filemtime(__FILE__) ?: 0,
    @filemtime(__DIR__ . '/assets/style.css') ?: 0,
    @filemtime(__DIR__ . '/assets/app.js') ?: 0,
    @filemtime(__DIR__ . '/buz.mp3') ?: 0
);

// Helpers
function device_key($dev){ $id=$dev['identification']??[]; return $id['mac'] ?? $id['id'] ?? $id['name'] ?? 'unknown'; }
function normalize_role($role){
    $role=strtolower(trim((string)$role));
    // Normalize common UISP wireless/base-station aliases
    $role=str_replace([' ','_'], '-', $role);
    $aliases=[
        'access-point'=>'ap',
        'accesspoint'=>'ap',
        'base-station'=>'ap',
        'basestation'=>'ap',
        'base'=>'ap',
        'cpe'=>'station',
        'client'=>'station',
        'subscriber'=>'station',
        'endpoint'=>'station',
    ];
    return $aliases[$role] ?? str_replace('-','',$role);
}
function device_role($dev){ return normalize_role($dev['identification']['role']??''); }
function device_role_label($role){
    $role=normalize_role($role);
    $labels=[
        'ap'=>'Access Point',
        'station'=>'Station',
        'ptp'=>'PTP',
        'gpon'=>'GPON',
        'homewifi'=>'Home WiFi',
        'wirelessdevice'=>'Wireless',
    ];
    if(isset($labels[$role])) return $labels[$role];
    $role=trim((string)$role);
    return $role!=='' ? ucfirst($role) : 'Device';
}
function is_gateway($d){ return device_role($d)==='gateway'; }
function is_router($d){ return device_role($d)==='router'; }
function is_switch($d){ return device_role($d)==='switch'; }
function is_ap($d){ $role=device_role($d); return in_array($role,['ap','wireless','homewifi','wirelessdevice','ptp'],true); }
function is_station($d){ $role=device_role($d); return in_array($role,['station'],true); }
function is_backbone($d){ $role=device_role($d); return in_array($role,['gateway','router','switch','ap','ptp'],true); }
function is_online($d){ $s=strtolower($d['overview']['status']??''); return in_array($s,['ok','online','active','connected','reachable','enabled']); }
function ping_host($ip){ if(!$ip) return null; $ip=preg_replace('/\/\d+$/','',$ip); $out=@shell_exec("ping -c 1 -W 1 ".escapeshellarg($ip)." 2>&1"); if(preg_match('/time=([\d\.]+)\s*ms/',$out,$m)) return floatval($m[1]); return null; }

function send_gotify($title,$message,$priority=5){
    global $GOTIFY_URL,$GOTIFY_TOKEN;
    if(!$GOTIFY_TOKEN){
        @file_put_contents(__DIR__.'/cache/gotify_log.txt', date('c')." missing GOTIFY_TOKEN\n", FILE_APPEND);
        return false;
    }
    $url = rtrim($GOTIFY_URL,'/').'/message';
    $payload = json_encode(['title'=>$title,'message'=>$message,'priority'=>$priority]);
    $ch=curl_init();
    curl_setopt_array($ch,[
        CURLOPT_URL=>$url,
        CURLOPT_POST=>true,
        CURLOPT_POSTFIELDS=>$payload,
        CURLOPT_HTTPHEADER=>[
            'Content-Type: application/json',
            'X-Gotify-Key: '.$GOTIFY_TOKEN
        ],
        CURLOPT_RETURNTRANSFER=>true,
        CURLOPT_TIMEOUT=>5
    ]);
    $resp = curl_exec($ch);
    $err  = curl_error($ch);
    $code = curl_getinfo($ch,CURLINFO_HTTP_CODE);
    curl_close($ch);
    if(!($code>=200 && $code<300)){
        @file_put_contents(__DIR__.'/cache/gotify_log.txt', date('c')." code=$code err=".($err?:'-')." resp=".$resp."\n", FILE_APPEND);
    }
    return $code>=200 && $code<300;
}

function api_get_json($baseUrl, $path, $token = '', $timeoutSec = 6){
    $base = rtrim((string)$baseUrl, '/');
    $uri = '/' . ltrim((string)$path, '/');
    $url = $base . $uri;
    $start = microtime(true);
    $ch = curl_init();
    $headers = ['accept: application/json'];
    if(trim((string)$token) !== ''){
        $headers[] = 'Authorization: Bearer ' . trim((string)$token);
    }
    curl_setopt_array($ch, [
        CURLOPT_URL => $url,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER => $headers,
        CURLOPT_TIMEOUT => max(2, (int)$timeoutSec),
    ]);
    $resp = curl_exec($ch);
    $err = curl_error($ch);
    $code = (int)curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);

    $latency = (int)round((microtime(true) - $start) * 1000);
    $json = json_decode((string)$resp, true);
    $ok = ($code >= 200 && $code < 300 && is_array($json));
    return [
        'ok' => $ok,
        'code' => $code,
        'latency_ms' => $latency,
        'error' => $ok ? '' : ($err ?: ('http_' . $code)),
        'json' => is_array($json) ? $json : null
    ];
}

// AJAX
if(isset($_GET['ajax'])){
    require_login_for_ajax();
    $currentUser = get_session_user($USERS_STORE);
    $effectiveSources = get_effective_uisp_sources($currentUser, $UISP_URL, $UISP_TOKEN);
    header("Content-Type: application/json");
    // Prevent caching of AJAX responses
    header('Cache-Control: no-store, no-cache, must-revalidate, max-age=0');
    header('Pragma: no-cache');

    if($_GET['ajax']==='mobile_config'){
        if(count($effectiveSources) === 0){
            http_response_code(503);
            echo json_encode([
                'error' => 'uisp_sources_not_configured',
                'message' => 'No UISP source has been configured for this account.'
            ]);
            exit;
        }
        $primary = $effectiveSources[0];
        echo json_encode([
            'uisp_base_url' => $primary['url'],
            'uisp_token' => $primary['token'],
            'sources' => $effectiveSources,
            'issued_at' => date('c')
        ]);
        exit;
    }

    if($_GET['ajax']==='devices'){
        if(count($effectiveSources) === 0){
            http_response_code(503);
            echo json_encode([
                'devices' => [],
                'http' => 503,
                'api_latency' => 0,
                'error' => 'uisp_sources_not_configured',
                'message' => 'Add one or more UISP sources in Account Settings.'
            ]);
            exit;
        }
        $deviceMap = [];
        $api_latency_sum = 0;
        $http_codes = [];
        $ok_sources = 0;
        foreach($effectiveSources as $src){
            $ch = curl_init();
            $start = microtime(true);
            curl_setopt_array($ch,[
                CURLOPT_URL=>rtrim((string)$src['url'],"/")."/nms/api/v2.1/devices",
                CURLOPT_RETURNTRANSFER=>true,
                CURLOPT_HTTPHEADER=>["accept: application/json","x-auth-token: ".$src['token']],
                CURLOPT_TIMEOUT=>10
            ]);
            $resp = curl_exec($ch);
            $lat = round((microtime(true)-$start)*1000);
            $api_latency_sum += $lat;
            $code = (int)curl_getinfo($ch,CURLINFO_HTTP_CODE);
            $http_codes[] = $code;
            curl_close($ch);

            $rows = json_decode((string)$resp, true);
            if(!is_array($rows)) continue;
            if($code >= 200 && $code < 300) $ok_sources++;

            foreach($rows as $d){
                if(!is_array($d)) continue;
                $id = device_key($d);
                $d['_source_id'] = $src['id'];
                $d['_source_name'] = $src['name'];
                $existing = $deviceMap[$id] ?? null;
                if($existing === null){
                    $deviceMap[$id] = $d;
                    continue;
                }
                // Prefer an online sample if duplicates exist across sources.
                if(is_online($d) && !is_online($existing)){
                    $deviceMap[$id] = $d;
                }
            }
        }

        $devices = array_values($deviceMap);
        $http_code = $ok_sources > 0 ? 200 : ((count($http_codes) > 0) ? max($http_codes) : 502);
        $api_latency = (count($effectiveSources) > 0) ? round($api_latency_sum / count($effectiveSources)) : 0;

        $now=time();
        $prev_cache = $cache; // snapshot to detect state transitions
        $out=[];
        $cache_changed=false;
        // Disable station/CPE ping batching to keep responses fast
        $cpe_ping_set = [];
        $ping_budget = 3; // ping at most 3 backbone/AP devices per poll to keep latency low
        foreach($devices as $d){
            $id=device_key($d);
            $name=$d['identification']['name']??$id;
            $role=device_role($d);
            $siteName = (string)($d['identification']['site']['name'] ?? $d['site']['name'] ?? $d['identification']['siteName'] ?? '');
            $siteId = (string)($d['identification']['site']['id'] ?? $d['identification']['siteId'] ?? '');
            $isGw=is_gateway($d);
            $isAp=is_ap($d);
            $isRouter=is_router($d);
            $isSwitch=is_switch($d);
            $isStation=is_station($d);
            $isBackbone=is_backbone($d);
            $on=is_online($d);
            $cpu=$d['overview']['cpu']??null;
            $ram=$d['overview']['ram']??null;
            $temp=$d['overview']['temperature']??null;
            // Uptime in seconds if available (UISP may expose different keys)
            $uptime=$d['overview']['uptime']
                ?? $d['overview']['uptimeSeconds']
                ?? $d['overview']['uptime_sec']
                ?? null;
            $lat=null;
            $cpe_lat=null;

            // Skip stations/CPEs entirely to keep UI focused on gateways/APs/routers/switches
            if(!$isGw && !$isAp && !$isRouter && !$isSwitch){
                continue;
            }

            if($isBackbone){
                // Ping no more than once per minute per backbone device (includes APs) and cap per-request to keep responses snappy
                $lastPingAt = $cache[$id]['last_ping_at'] ?? 0;
                $cachedLat  = $cache[$id]['last_ping_ms'] ?? null;
                if(($now - $lastPingAt) >= 60 || $cachedLat===null){
                    if($ping_budget > 0){
                        $lat=ping_host($d['ipAddress']??null);
                        $cache[$id]['last_ping_at']=$now;
                        $cache[$id]['last_ping_ms']=$lat;
                        $cache_changed=true;
                        $ping_budget--;
                    } else {
                        $lat=$cachedLat;
                    }
                } else {
                    $lat=$cachedLat;
                }
                if($now%60===0){
                    $stmt=$db->prepare("INSERT INTO metrics (device_id,name,cpu,ram,temp,latency,online) VALUES (?,?,?,?,?,?,?)");
                    $stmt->bindValue(1,$id,SQLITE3_TEXT);
                    $stmt->bindValue(2,$name,SQLITE3_TEXT);
                    $stmt->bindValue(3,$cpu,SQLITE3_INTEGER);
                    $stmt->bindValue(4,$ram,SQLITE3_INTEGER);
                    $stmt->bindValue(5,$temp,SQLITE3_INTEGER);
                    $stmt->bindValue(6,$lat,SQLITE3_FLOAT);
                    $stmt->bindValue(7,$on?1:0,SQLITE3_INTEGER);
                    @$stmt->execute();
                }
            } else {
                // Stations/CPEs are not displayed; skip ping work.
                continue;
            }

            $sim=!empty($cache[$id]['simulate']);
            if($sim) $on=false;

            $roleLabel = device_role_label($role);

            // Track offline start time to compute outage duration
            if(!isset($cache[$id])) $cache[$id]=[];
            if(!$on){
                if(empty($cache[$id]['offline_since'])){ $cache[$id]['offline_since']=$now; $cache_changed=true; }
                $offline_since=$cache[$id]['offline_since'];
                // Notify when gateway is offline: first after threshold, then every 10 minutes while still offline
                $threshold_met = ($now - ($cache[$id]['offline_since']??$now)) >= $FIRST_OFFLINE_THRESHOLD;
                $last_sent = $cache[$id]['gf_last_offline_notif'] ?? null;
                $should_repeat = ($last_sent && ($now - $last_sent) >= 600);
                if($isBackbone && $threshold_met && (!$last_sent || $should_repeat)){
                    @file_put_contents($CACHE_DIR.'/gotify_log.txt', date('c')." offline eval: id=$id name=$name role=$role threshold_met=$threshold_met last_sent=".($last_sent?:'null')." repeat=$should_repeat\n", FILE_APPEND);
                    if(send_gotify($roleLabel.' Offline', $name.' is OFFLINE', 8)){
                        $cache[$id]['gf_last_offline_notif']=$now; $cache_changed=true;
                    } else {
                        @file_put_contents($CACHE_DIR.'/gotify_log.txt', date('c')." offline send_gotify returned false for id=$id name=$name\n", FILE_APPEND);
                    }
                }
            } else {
                if(($cache[$id]['last_seen'] ?? 0) !== $now){ $cache[$id]['last_seen'] = $now; $cache_changed = true; }
                if(!empty($cache[$id]['offline_since'])){ unset($cache[$id]['offline_since']); $cache_changed=true; }
                $offline_since=null;
                // If previously offline, send recovery notification
                if(!empty($prev_cache[$id]['offline_since']) && $isBackbone){
                    @file_put_contents($CACHE_DIR.'/gotify_log.txt', date('c')." online eval: id=$id name=$name role=$role\n", FILE_APPEND);
                    if(send_gotify($roleLabel.' Online', $name.' is back ONLINE', 5)){
                        unset($cache[$id]['gf_last_offline_notif']); $cache_changed=true;
                    }
                } else {
                    if(isset($cache[$id]['gf_last_offline_notif'])){ unset($cache[$id]['gf_last_offline_notif']); $cache_changed=true; }
                }
            }

            $ack_until=$cache[$id]['ack_until']??null;
            $ack_active = $ack_until && $ack_until > $now;
            $last_seen = (int)($cache[$id]['last_seen'] ?? ($on ? $now : 0));

            $flap_history = $cache[$id]['flap_history'] ?? [];
            if(!empty($flap_history)){
                $filtered = [];
                foreach($flap_history as $ts){
                    if(($now - $ts) <= $FLAP_ALERT_WINDOW) $filtered[] = $ts;
                }
                if(count($filtered) !== count($flap_history)){
                    $flap_history = $filtered;
                    $cache[$id]['flap_history'] = $flap_history;
                    $cache_changed = true;
                }
            }
            if($isBackbone && !empty($prev_cache[$id]['offline_since']) && $on){
                $flap_history[] = $now;
                $cache[$id]['flap_history'] = $flap_history;
                $cache_changed = true;
            }
            $flaps_recent = count($flap_history);

            $flap_alert_active = ($isBackbone && $flaps_recent >= $FLAP_ALERT_THRESHOLD);
            $latency_alert_active = false;

            if($isBackbone){
                if($flap_alert_active && !$ack_active){
                    $last_flap_sent = $cache[$id]['flap_alert_sent_at'] ?? 0;
                    if(($now - $last_flap_sent) >= $FLAP_ALERT_SUPPRESS){
                        if(send_gotify($roleLabel.' Flapping', $name.' flapped '. $flaps_recent.' times in last '.(int)($FLAP_ALERT_WINDOW/60).' minutes', 6)){
                            $cache[$id]['flap_alert_sent_at'] = $now;
                            $cache_changed = true;
                        }
                    }
                } else {
                    if(isset($cache[$id]['flap_alert_sent_at']) && !$flap_alert_active){
                        unset($cache[$id]['flap_alert_sent_at']);
                        $cache_changed = true;
                    }
                }

                $streak = $cache[$id]['latency_high_streak'] ?? 0;
                if($lat !== null && is_numeric($lat)){
                    if($lat >= $LATENCY_ALERT_THRESHOLD){
                        $streak++;
                    } else {
                        $streak = 0;
                    }
                } else {
                    $streak = 0;
                }
                if(($cache[$id]['latency_high_streak'] ?? null) !== $streak){
                    $cache[$id]['latency_high_streak'] = $streak;
                    $cache_changed = true;
                }
                if($streak >= $LATENCY_ALERT_STREAK){
                    $latency_alert_active = true;
                    if(!$ack_active){
                        $last_lat_sent = $cache[$id]['latency_alert_sent_at'] ?? 0;
                        if(($now - $last_lat_sent) >= $LATENCY_ALERT_SUPPRESS){
                            $message = $lat !== null ? ($name.' latency '. $lat.' ms') : ($name.' latency sustained high');
                            if(send_gotify($roleLabel.' Latency High', $message, 5)){
                                $cache[$id]['latency_alert_sent_at'] = $now;
                                $cache_changed = true;
                            }
                        }
                    }
                } else {
                    if(isset($cache[$id]['latency_alert_sent_at']) && ($now - $cache[$id]['latency_alert_sent_at']) > $LATENCY_ALERT_SUPPRESS){
                        unset($cache[$id]['latency_alert_sent_at']);
                        $cache_changed = true;
                    }
                }
            } else {
                $flaps_recent = 0;
            }


            $out[]=[
                'id'=>$id,'name'=>$name,
                'gateway'=>$isGw,'ap'=>$isAp,'station'=>$isStation,
                'router'=>$isRouter,'switch'=>$isSwitch,'role'=>$role,'backbone'=>$isBackbone,
                'source_id'=>(string)($d['_source_id'] ?? ''),
                'source_name'=>(string)($d['_source_name'] ?? ''),
                'site_id'=>$siteId,
                'site'=>$siteName,
                'hostname'=>(string)($d['identification']['hostname'] ?? ''),
                'mac'=>(string)($d['identification']['mac'] ?? ''),
                'serial'=>(string)($d['identification']['serialNumber'] ?? ''),
                'vendor'=>(string)($d['identification']['vendor'] ?? ''),
                'model'=>(string)($d['identification']['model'] ?? ''),
                'online'=>$on,'cpu'=>$cpu,'ram'=>$ram,'temp'=>$temp,'latency'=>$lat,
                'cpe_latency'=>$cpe_lat,
                'uptime'=>$uptime,
                'last_seen'=>$last_seen,
                'offline_since'=>$offline_since,
                'flaps_recent'=>$flaps_recent,
                'latency_alert'=>$latency_alert_active,
                'flap_alert'=>$flap_alert_active,
                'simulate'=>$sim,'ack_until'=>$ack_until
            ];
        }

        if($cache_changed){ file_put_contents($CACHE_FILE,json_encode($cache)); }
        echo json_encode(['devices'=>$out,'http'=>$http_code,'api_latency'=>$api_latency]); exit;
    }

    if($_GET['ajax']==='inventory_overview'){
        require_pro_feature('inventory');
        $identsReq = api_get_json($NOCWALL_API_URL, '/inventory/identities', $NOCWALL_API_TOKEN, 6);
        if(!$identsReq['ok']){
            http_response_code(502);
            echo json_encode([
                'ok' => 0,
                'error' => 'inventory_unreachable',
                'message' => 'Inventory API is unavailable.',
                'details' => $identsReq['error'],
                'http' => $identsReq['code'],
                'api_latency' => $identsReq['latency_ms']
            ]);
            exit;
        }

        $identities = $identsReq['json']['identities'] ?? [];
        if(!is_array($identities)) $identities = [];

        $driftReq = api_get_json($NOCWALL_API_URL, '/inventory/drift?limit=1000', $NOCWALL_API_TOKEN, 6);
        $lifeReq = api_get_json($NOCWALL_API_URL, '/inventory/lifecycle?limit=1000', $NOCWALL_API_TOKEN, 6);

        $driftLatest = [];
        if($driftReq['ok']){
            $snapshots = $driftReq['json']['snapshots'] ?? [];
            if(is_array($snapshots)){
                foreach($snapshots as $snap){
                    if(!is_array($snap)) continue;
                    $identityId = trim((string)($snap['identity_id'] ?? ''));
                    if($identityId === '') continue;
                    $observedAt = (int)($snap['observed_at'] ?? 0);
                    if(!isset($driftLatest[$identityId]) || $observedAt >= (int)($driftLatest[$identityId]['observed_at'] ?? 0)){
                        $driftLatest[$identityId] = $snap;
                    }
                }
            }
        }

        $lifecycle = [];
        if($lifeReq['ok']){
            $scores = $lifeReq['json']['scores'] ?? [];
            if(is_array($scores)){
                foreach($scores as $score){
                    if(!is_array($score)) continue;
                    $identityId = trim((string)($score['identity_id'] ?? ''));
                    if($identityId === '') continue;
                    $lifecycle[$identityId] = $score;
                }
            }
        }

        $devices = [];
        foreach($identities as $ident){
            if(!is_array($ident)) continue;
            $primary = trim((string)($ident['primary_device_id'] ?? ''));
            if($primary === '') continue;
            $identityId = trim((string)($ident['identity_id'] ?? ''));
            $drift = $identityId !== '' ? ($driftLatest[$identityId] ?? null) : null;
            $life = $identityId !== '' ? ($lifecycle[$identityId] ?? null) : null;

            $devices[$primary] = [
                'identity_id' => $identityId,
                'name' => (string)($ident['name'] ?? ''),
                'role' => (string)($ident['role'] ?? ''),
                'site_id' => (string)($ident['site_id'] ?? ''),
                'last_seen' => (int)($ident['last_seen'] ?? 0),
                'drift_changed' => is_array($drift) ? !empty($drift['changed']) : false,
                'drift_observed_at' => is_array($drift) ? (int)($drift['observed_at'] ?? 0) : 0,
                'lifecycle_level' => is_array($life) ? (string)($life['level'] ?? '') : '',
                'lifecycle_score' => is_array($life) ? (int)($life['score'] ?? 0) : 0,
                'source_refs_count' => is_array($ident['source_refs'] ?? null) ? count($ident['source_refs']) : 0
            ];
        }

        echo json_encode([
            'ok' => 1,
            'fetched_at' => date('c'),
            'identities' => count($identities),
            'devices' => $devices,
            'api_latency' => [
                'identities' => $identsReq['latency_ms'],
                'drift' => $driftReq['latency_ms'],
                'lifecycle' => $lifeReq['latency_ms']
            ]
        ]);
        exit;
    }

    if($_GET['ajax']==='inventory_device'){
        require_pro_feature('inventory');
        $deviceId = trim((string)($_GET['id'] ?? ''));
        $deviceName = trim((string)($_GET['name'] ?? ''));
        if($deviceId === ''){
            http_response_code(400);
            echo json_encode(['ok'=>0,'error'=>'id_required']);
            exit;
        }

        $identsReq = api_get_json($NOCWALL_API_URL, '/inventory/identities', $NOCWALL_API_TOKEN, 6);
        if(!$identsReq['ok']){
            http_response_code(502);
            echo json_encode([
                'ok' => 0,
                'error' => 'inventory_unreachable',
                'message' => 'Inventory API is unavailable.',
                'details' => $identsReq['error'],
                'http' => $identsReq['code']
            ]);
            exit;
        }

        $identities = $identsReq['json']['identities'] ?? [];
        if(!is_array($identities)) $identities = [];
        $identity = null;
        foreach($identities as $ident){
            if(!is_array($ident)) continue;
            $primary = trim((string)($ident['primary_device_id'] ?? ''));
            if($primary !== '' && $primary === $deviceId){
                $identity = $ident;
                break;
            }
        }
        if($identity === null && $deviceName !== ''){
            $nameNeedle = strtolower($deviceName);
            foreach($identities as $ident){
                if(!is_array($ident)) continue;
                $name = strtolower(trim((string)($ident['name'] ?? '')));
                if($name !== '' && $name === $nameNeedle){
                    $identity = $ident;
                    break;
                }
            }
        }
        if($identity === null){
            echo json_encode([
                'ok' => 1,
                'device_id' => $deviceId,
                'identity' => null,
                'interfaces' => [],
                'neighbors' => [],
                'drift' => [],
                'lifecycle' => null,
                'message' => 'No inventory identity found yet for this device. Wait for telemetry ingest/source polling.'
            ]);
            exit;
        }

        $identityId = trim((string)($identity['identity_id'] ?? ''));
        $q = rawurlencode($identityId);
        $ifaceReq = api_get_json($NOCWALL_API_URL, '/inventory/interfaces?identity_id=' . $q . '&limit=200', $NOCWALL_API_TOKEN, 6);
        $neighReq = api_get_json($NOCWALL_API_URL, '/inventory/neighbors?identity_id=' . $q . '&limit=200', $NOCWALL_API_TOKEN, 6);
        $driftReq = api_get_json($NOCWALL_API_URL, '/inventory/drift?identity_id=' . $q . '&limit=40', $NOCWALL_API_TOKEN, 6);
        $lifeReq = api_get_json($NOCWALL_API_URL, '/inventory/lifecycle?identity_id=' . $q . '&limit=1', $NOCWALL_API_TOKEN, 6);

        $interfaces = ($ifaceReq['ok'] && is_array($ifaceReq['json']['interfaces'] ?? null)) ? $ifaceReq['json']['interfaces'] : [];
        $neighbors = ($neighReq['ok'] && is_array($neighReq['json']['neighbors'] ?? null)) ? $neighReq['json']['neighbors'] : [];
        $drift = ($driftReq['ok'] && is_array($driftReq['json']['snapshots'] ?? null)) ? $driftReq['json']['snapshots'] : [];
        $lifecycle = null;
        if($lifeReq['ok'] && is_array($lifeReq['json']['scores'] ?? null) && count($lifeReq['json']['scores']) > 0){
            $lifecycle = $lifeReq['json']['scores'][0];
        }

        $ifaceSummary = ['total' => 0, 'oper_up' => 0, 'oper_down' => 0];
        foreach($interfaces as $iface){
            if(!is_array($iface)) continue;
            $ifaceSummary['total']++;
            if(array_key_exists('oper_up', $iface) && $iface['oper_up'] === true){
                $ifaceSummary['oper_up']++;
            } else {
                $ifaceSummary['oper_down']++;
            }
        }

        usort($interfaces, function($a, $b){
            $an = strtolower(trim((string)($a['name'] ?? '')));
            $bn = strtolower(trim((string)($b['name'] ?? '')));
            return strcmp($an, $bn);
        });
        usort($drift, function($a, $b){
            $at = (int)($a['observed_at'] ?? 0);
            $bt = (int)($b['observed_at'] ?? 0);
            return $bt <=> $at;
        });

        echo json_encode([
            'ok' => 1,
            'device_id' => $deviceId,
            'identity' => $identity,
            'interface_summary' => $ifaceSummary,
            'interfaces' => array_slice($interfaces, 0, 80),
            'neighbors' => array_slice($neighbors, 0, 80),
            'drift' => array_slice($drift, 0, 20),
            'lifecycle' => $lifecycle,
            'api_latency' => [
                'identities' => $identsReq['latency_ms'],
                'interfaces' => $ifaceReq['latency_ms'],
                'neighbors' => $neighReq['latency_ms'],
                'drift' => $driftReq['latency_ms'],
                'lifecycle' => $lifeReq['latency_ms']
            ]
        ]);
        exit;
    }

    if($_GET['ajax']==='topology_overview'){
        require_pro_feature('topology');
        $limitNodes = (int)($_GET['nodes_limit'] ?? 1200);
        $limitEdges = (int)($_GET['edges_limit'] ?? 2000);
        if($limitNodes <= 0 || $limitNodes > 5000) $limitNodes = 1200;
        if($limitEdges <= 0 || $limitEdges > 8000) $limitEdges = 2000;

        $nodesReq = api_get_json($NOCWALL_API_URL, '/topology/nodes?limit=' . $limitNodes, $NOCWALL_API_TOKEN, 7);
        $edgesReq = api_get_json($NOCWALL_API_URL, '/topology/edges?limit=' . $limitEdges, $NOCWALL_API_TOKEN, 7);
        $healthReq = api_get_json($NOCWALL_API_URL, '/topology/health', $NOCWALL_API_TOKEN, 7);

        if(!$nodesReq['ok'] || !$edgesReq['ok'] || !$healthReq['ok']){
            http_response_code(502);
            echo json_encode([
                'ok' => 0,
                'error' => 'topology_unreachable',
                'message' => 'Topology API is unavailable.',
                'details' => [
                    'nodes' => ['ok'=>$nodesReq['ok'],'http'=>$nodesReq['code'],'error'=>$nodesReq['error']],
                    'edges' => ['ok'=>$edgesReq['ok'],'http'=>$edgesReq['code'],'error'=>$edgesReq['error']],
                    'health' => ['ok'=>$healthReq['ok'],'http'=>$healthReq['code'],'error'=>$healthReq['error']]
                ]
            ]);
            exit;
        }

        $nodes = $nodesReq['json']['nodes'] ?? [];
        $edges = $edgesReq['json']['edges'] ?? [];
        $health = $healthReq['json']['health'] ?? [];
        if(!is_array($nodes)) $nodes = [];
        if(!is_array($edges)) $edges = [];
        if(!is_array($health)) $health = [];

        echo json_encode([
            'ok' => 1,
            'nodes' => $nodes,
            'edges' => $edges,
            'health' => $health,
            'counts' => [
                'nodes' => count($nodes),
                'edges' => count($edges),
            ],
            'api_latency' => [
                'nodes' => $nodesReq['latency_ms'],
                'edges' => $edgesReq['latency_ms'],
                'health' => $healthReq['latency_ms']
            ],
            'fetched_at' => date('c')
        ]);
        exit;
    }

    if($_GET['ajax']==='topology_trace'){
        require_pro_feature('topology');
        $sourceNodeID = trim((string)($_GET['source_node_id'] ?? ''));
        $targetNodeID = trim((string)($_GET['target_node_id'] ?? ''));
        $sourceIdentityID = trim((string)($_GET['source_identity_id'] ?? ''));
        $targetIdentityID = trim((string)($_GET['target_identity_id'] ?? ''));

        $query = [];
        if($sourceNodeID !== '') $query[] = 'source_node_id=' . rawurlencode($sourceNodeID);
        if($targetNodeID !== '') $query[] = 'target_node_id=' . rawurlencode($targetNodeID);
        if($sourceIdentityID !== '') $query[] = 'source_identity_id=' . rawurlencode($sourceIdentityID);
        if($targetIdentityID !== '') $query[] = 'target_identity_id=' . rawurlencode($targetIdentityID);

        $path = '/topology/path';
        if(count($query) > 0) $path .= '?' . implode('&', $query);

        $traceReq = api_get_json($NOCWALL_API_URL, $path, $NOCWALL_API_TOKEN, 7);
        if(!$traceReq['ok']){
            http_response_code(502);
            echo json_encode([
                'ok' => 0,
                'error' => 'topology_trace_failed',
                'message' => 'Topology trace is unavailable.',
                'details' => $traceReq['error'],
                'http' => $traceReq['code']
            ]);
            exit;
        }

        $payload = $traceReq['json'];
        if(!is_array($payload)) $payload = [];
        $payload['ok'] = 1;
        $payload['api_latency'] = $traceReq['latency_ms'];
        echo json_encode($payload);
        exit;
    }

    if($_GET['ajax']==='history' && !empty($_GET['id'])){
        require_pro_feature('history');
        $id=$_GET['id'];
        $stmt=$db->prepare("SELECT timestamp,cpu,ram,temp,latency FROM metrics WHERE device_id=? ORDER BY timestamp DESC LIMIT 1440");
        $stmt->bindValue(1,$id,SQLITE3_TEXT);
        $res=$stmt->execute();
        $rows=[];
        while($r=$res->fetchArray(SQLITE3_ASSOC)) $rows[]=$r;
        echo json_encode(array_reverse($rows)); exit;
    }
    if($_GET['ajax']==='cpe_history'){
        require_pro_feature('cpe_history');
        $id=trim((string)($_GET['id'] ?? ''));
        $points=[];
        if($id!==''){
            $stmt=$db->prepare("SELECT strftime('%s', timestamp) AS ts, latency, device_id, name FROM cpe_pings WHERE device_id=? AND timestamp >= datetime('now','-7 days') ORDER BY timestamp ASC");
            $stmt->bindValue(1,$id,SQLITE3_TEXT);
        } else {
            $stmt=$db->prepare("SELECT strftime('%s', timestamp) AS ts, latency, device_id, name FROM cpe_pings WHERE timestamp >= datetime('now','-7 days') ORDER BY timestamp ASC");
        }
        $res=$stmt->execute();
        if($res){
            while($row=$res->fetchArray(SQLITE3_ASSOC)){
                $ts = isset($row['ts']) ? (int)$row['ts'] : null;
                $latVal = array_key_exists('latency',$row) ? $row['latency'] : null;
                $points[]=[
                    'device_id'=>$row['device_id'] ?? null,
                    'name'=>$row['name'] ?? null,
                    'ts_ms'=>$ts!==null ? $ts*1000 : null,
                    'latency'=>$latVal===null ? null : (float)$latVal
                ];
            }
        }
        echo json_encode([
            'device_id'=>$id ?: null,
            'range_days'=>7,
            'points'=>$points
        ]);
        exit;
    }

    if($_GET['ajax']==='gotifytest'){
        $ok = send_gotify('Test from UISP NOC','This is a test notification.', 5);
        echo json_encode(['ok'=>$ok?1:0]); exit;
    }

    // --- Caddy TLS helpers ---
    if($_GET['ajax']==='caddy_cfg'){
        $ch=curl_init();
        curl_setopt_array($ch,[
            CURLOPT_URL=>'http://caddy:2019/config/',
            CURLOPT_RETURNTRANSFER=>true,
            CURLOPT_TIMEOUT=>4
        ]);
        $resp=curl_exec($ch);
        $err=curl_error($ch);
        $code=curl_getinfo($ch,CURLINFO_HTTP_CODE);
        curl_close($ch);
        if($code>=200 && $code<300){ echo $resp; } else { echo json_encode(['error'=>'caddy_unreachable','code'=>$code,'err'=>$err]); }
        exit;
    }

    if($_GET['ajax']==='provision_tls' && $_SERVER['REQUEST_METHOD']==='POST'){
        $domain = trim($_POST['domain'] ?? '');
        $gdomain = trim($_POST['gotify_domain'] ?? '');
        $email = trim($_POST['email'] ?? '');
        $staging = !empty($_POST['staging']);
        if($domain===''){ echo json_encode(['ok'=>0,'error'=>'missing_domain']); exit; }
        if($email===''){ echo json_encode(['ok'=>0,'error'=>'missing_email']); exit; }
        $issuers = [['module'=>'acme','email'=>$email]];
        if($staging){ $issuers[0]['ca']='https://acme-staging-v02.api.letsencrypt.org/directory'; }

        $routes=[];
        $routes[] = [
            'match'=>[['host'=>[$domain]]],
            'handle'=>[[
                'handler'=>'reverse_proxy',
                'upstreams'=>[['dial'=>'uisp-noc:80']]
            ]]
        ];
        if($gdomain!==''){
            $routes[] = [
                'match'=>[['host'=>[$gdomain]]],
                'handle'=>[[
                    'handler'=>'reverse_proxy',
                    'upstreams'=>[['dial'=>'uisp-noc:18080']]
                ]]
            ];
        }
        $cfg = [
            'apps'=>[
                'tls'=>[
                    'automation'=>[
                        'policies'=>[[ 'issuers'=>$issuers ]]
                    ]
                ],
                'http'=>[
                    'servers'=>[
                        'srv0'=>[
                            'listen'=>[':80',':443'],
                            'routes'=>$routes
                        ]
                    ]
                ]
            ]
        ];
        $payload=json_encode($cfg);
        $ch=curl_init();
        curl_setopt_array($ch,[
            CURLOPT_URL=>'http://caddy:2019/load',
            CURLOPT_POST=>true,
            CURLOPT_POSTFIELDS=>$payload,
            CURLOPT_HTTPHEADER=>['Content-Type: application/json'],
            CURLOPT_RETURNTRANSFER=>true,
            CURLOPT_TIMEOUT=>10
        ]);
        $resp=curl_exec($ch);
        $err=curl_error($ch);
        $code=curl_getinfo($ch,CURLINFO_HTTP_CODE);
        curl_close($ch);
        if($code>=200 && $code<300){ echo json_encode(['ok'=>1,'message'=>'caddy_config_loaded']); }
        else { echo json_encode(['ok'=>0,'error'=>'caddy_load_failed','code'=>$code,'err'=>$err,'resp'=>$resp]); }
        exit;
    }

    if($_GET['ajax']==='changepw' && $_SERVER['REQUEST_METHOD']==='POST'){
        $cur = $_POST['current'] ?? '';
        $new = $_POST['new'] ?? '';
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        if(!$user){
            echo json_encode(['ok'=>0,'error'=>'invalid_session']); exit;
        }
        if(!password_verify($cur, (string)($user['password_hash'] ?? ''))){
            echo json_encode(['ok'=>0,'error'=>'current_password_incorrect']); exit;
        }
        if(strlen($new) < 8){
            echo json_encode(['ok'=>0,'error'=>'new_password_too_short']); exit;
        }
        $USERS_STORE['users'][$sessionUser]['password_hash'] = password_hash($new, PASSWORD_DEFAULT);
        $USERS_STORE['users'][$sessionUser]['updated_at'] = date('c');
        save_users_store($USERS_FILE, $USERS_STORE);
        echo json_encode(['ok'=>1]); exit;
    }

    if($_GET['ajax']==='prefs_get'){
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        if(!$user){
            echo json_encode(['ok'=>0,'error'=>'invalid_session']); exit;
        }
        $prefs = normalize_user_preferences($user['preferences'] ?? null);
        echo json_encode([
            'ok' => 1,
            'username' => $sessionUser,
            'preferences' => $prefs
        ]);
        exit;
    }

    if($_GET['ajax']==='prefs_save' && $_SERVER['REQUEST_METHOD']==='POST'){
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        if(!$user){
            echo json_encode(['ok'=>0,'error'=>'invalid_session']); exit;
        }

        $prefs = normalize_user_preferences($user['preferences'] ?? null);
        $hadInput = false;

        if(array_key_exists('dashboard_settings', $_POST)){
            $hadInput = true;
            $decoded = json_decode((string)$_POST['dashboard_settings'], true);
            if(!is_array($decoded)){
                echo json_encode(['ok'=>0,'error'=>'invalid_dashboard_settings']); exit;
            }
            $prefs['dashboard_settings'] = normalize_dashboard_settings($decoded);
        }

        if(array_key_exists('ap_siren_prefs', $_POST)){
            $hadInput = true;
            $decoded = json_decode((string)$_POST['ap_siren_prefs'], true);
            if(!is_array($decoded)){
                echo json_encode(['ok'=>0,'error'=>'invalid_ap_siren_prefs']); exit;
            }
            $prefs['ap_siren_prefs'] = normalize_ap_siren_prefs($decoded);
        }

        if(array_key_exists('tab_siren_prefs', $_POST)){
            $hadInput = true;
            $decoded = json_decode((string)$_POST['tab_siren_prefs'], true);
            if(!is_array($decoded)){
                echo json_encode(['ok'=>0,'error'=>'invalid_tab_siren_prefs']); exit;
            }
            $prefs['tab_siren_prefs'] = normalize_tab_siren_prefs($decoded);
        }

        if(array_key_exists('card_order_prefs', $_POST)){
            $hadInput = true;
            $decoded = json_decode((string)$_POST['card_order_prefs'], true);
            if(!is_array($decoded)){
                echo json_encode(['ok'=>0,'error'=>'invalid_card_order_prefs']); exit;
            }
            $prefs['card_order_prefs'] = normalize_card_order_prefs($decoded);
        }

        if(!$hadInput){
            echo json_encode(['ok'=>0,'error'=>'no_fields']); exit;
        }

        $USERS_STORE['users'][$sessionUser]['preferences'] = $prefs;
        $USERS_STORE['users'][$sessionUser]['updated_at'] = date('c');
        save_users_store($USERS_FILE, $USERS_STORE);
        echo json_encode(['ok'=>1,'preferences'=>$prefs]); exit;
    }

    if($_GET['ajax']==='sources_status'){
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        if(!$user){
            echo json_encode(['ok'=>0,'error'=>'invalid_session']); exit;
        }
        $statusMap = get_user_source_status_map($user);
        $effective = get_effective_uisp_sources($user, $UISP_URL, $UISP_TOKEN);
        $rows = [];
        foreach($effective as $src){
            $id = (string)($src['id'] ?? '');
            $saved = $statusMap[$id] ?? null;
            $rows[] = [
                'id' => $id,
                'name' => (string)($src['name'] ?? $id),
                'url' => normalize_uisp_url($src['url'] ?? ''),
                'enabled' => !empty($src['enabled']),
                'ok' => ($saved ? !empty($saved['ok']) : null),
                'http' => ($saved ? (int)($saved['http'] ?? 0) : null),
                'latency_ms' => ($saved ? (int)($saved['latency_ms'] ?? 0) : null),
                'device_count' => ($saved ? (int)($saved['device_count'] ?? 0) : null),
                'error' => ($saved ? (string)($saved['error'] ?? '') : ''),
                'last_poll_at' => ($saved ? (string)($saved['last_poll_at'] ?? '') : '')
            ];
        }
        $summary = summarize_source_status_rows($rows);
        echo json_encode([
            'ok' => 1,
            'username' => $sessionUser,
            'sources' => $rows,
            'summary' => $summary
        ]);
        exit;
    }

    if($_GET['ajax']==='sources_list'){
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        if(!$user){
            echo json_encode(['ok'=>0,'error'=>'invalid_session']); exit;
        }
        $sources = get_stored_user_sources($user);
        $view = [];
        foreach($sources as $src){
            $token = (string)$src['token'];
            $len = strlen($token);
            $tail = $len > 4 ? substr($token, -4) : $token;
            $view[] = [
                'id' => $src['id'],
                'name' => $src['name'],
                'url' => $src['url'],
                'enabled' => !empty($src['enabled']),
                'token_hint' => ($len > 0 ? str_repeat('*', max(4, min($len, 12))) . $tail : ''),
                'has_token' => ($len > 0),
                'updated_at' => $src['updated_at'] ?? null
            ];
        }
        echo json_encode([
            'ok' => 1,
            'username' => $sessionUser,
            'sources' => $view
        ]);
        exit;
    }

    if($_GET['ajax']==='sources_save' && $_SERVER['REQUEST_METHOD']==='POST'){
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        if(!$user){
            echo json_encode(['ok'=>0,'error'=>'invalid_session']); exit;
        }
        $sources = get_stored_user_sources($user);
        $id = trim((string)($_POST['id'] ?? ''));
        $name = trim((string)($_POST['name'] ?? ''));
        $url = normalize_uisp_url($_POST['url'] ?? '');
        $token = trim((string)($_POST['token'] ?? ''));
        $enabledRaw = $_POST['enabled'] ?? '1';
        $enabled = !($enabledRaw === false || $enabledRaw === 0 || $enabledRaw === '0' || strtolower((string)$enabledRaw) === 'false');

        if($url === '' || is_placeholder_uisp_url($url)){
            echo json_encode(['ok'=>0,'error'=>'invalid_url','message'=>'A valid UISP URL is required.']); exit;
        }
        if($name === ''){
            $parsed = parse_url($url);
            $name = (string)($parsed['host'] ?? 'UISP Source');
        }

        $found = -1;
        for($i=0; $i<count($sources); $i++){
            if((string)$sources[$i]['id'] === $id){
                $found = $i;
                break;
            }
        }

        if($found >= 0){
            if($token === ''){
                $token = (string)($sources[$found]['token'] ?? '');
            }
            if($token === ''){
                echo json_encode(['ok'=>0,'error'=>'token_required']); exit;
            }
            $sources[$found]['name'] = $name;
            $sources[$found]['url'] = $url;
            $sources[$found]['token'] = $token;
            $sources[$found]['enabled'] = $enabled;
            $sources[$found]['updated_at'] = date('c');
            $savedId = $sources[$found]['id'];
        } else {
            if($token === '' || strlen($token) < 12){
                echo json_encode(['ok'=>0,'error'=>'token_required','message'=>'Provide a valid UISP API token.']); exit;
            }
            $savedId = ($id !== '' ? $id : generate_source_id());
            $sources[] = [
                'id' => $savedId,
                'name' => $name,
                'url' => $url,
                'token' => $token,
                'enabled' => $enabled,
                'created_at' => date('c'),
                'updated_at' => date('c')
            ];
        }

        $USERS_STORE['users'][$sessionUser]['sources'] = $sources;
        $statusRows = normalize_user_source_status($user['source_status'] ?? null);
        foreach($statusRows as &$statusRow){
            if((string)($statusRow['id'] ?? '') !== (string)$savedId) continue;
            $statusRow['name'] = $name;
            $statusRow['url'] = $url;
        }
        unset($statusRow);
        $USERS_STORE['users'][$sessionUser]['source_status'] = $statusRows;
        $USERS_STORE['users'][$sessionUser]['updated_at'] = date('c');
        save_users_store($USERS_FILE, $USERS_STORE);
        echo json_encode(['ok'=>1, 'id'=>$savedId, 'count'=>count($sources)]); exit;
    }

    if($_GET['ajax']==='sources_delete' && $_SERVER['REQUEST_METHOD']==='POST'){
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        if(!$user){
            echo json_encode(['ok'=>0,'error'=>'invalid_session']); exit;
        }
        $id = trim((string)($_POST['id'] ?? ''));
        if($id === ''){
            echo json_encode(['ok'=>0,'error'=>'id_required']); exit;
        }
        $sources = get_stored_user_sources($user);
        $filtered = [];
        foreach($sources as $src){
            if((string)$src['id'] !== $id) $filtered[] = $src;
        }
        $USERS_STORE['users'][$sessionUser]['sources'] = $filtered;
        $statusRows = normalize_user_source_status($user['source_status'] ?? null);
        $statusFiltered = [];
        foreach($statusRows as $row){
            if((string)($row['id'] ?? '') !== $id) $statusFiltered[] = $row;
        }
        $USERS_STORE['users'][$sessionUser]['source_status'] = $statusFiltered;
        $USERS_STORE['users'][$sessionUser]['updated_at'] = date('c');
        save_users_store($USERS_FILE, $USERS_STORE);
        echo json_encode(['ok'=>1, 'count'=>count($filtered)]); exit;
    }

    if($_GET['ajax']==='sources_test' && $_SERVER['REQUEST_METHOD']==='POST'){
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        if(!$user){
            echo json_encode(['ok'=>0,'error'=>'invalid_session']); exit;
        }
        $id = trim((string)($_POST['id'] ?? ''));
        if($id === ''){
            echo json_encode(['ok'=>0,'error'=>'id_required']); exit;
        }
        $sources = get_effective_uisp_sources($user, $UISP_URL, $UISP_TOKEN);
        $target = null;
        foreach($sources as $src){
            if((string)$src['id'] === $id){
                $target = $src;
                break;
            }
        }
        if(!$target){
            echo json_encode(['ok'=>0,'error'=>'source_not_found']); exit;
        }
        $probe = probe_uisp_source($target);

        $statusRows = normalize_user_source_status($user['source_status'] ?? null);
        $statusSaved = false;
        for($i = 0; $i < count($statusRows); $i++){
            if((string)($statusRows[$i]['id'] ?? '') !== $id) continue;
            $statusRows[$i] = normalize_source_status_entry($probe);
            $statusSaved = true;
            break;
        }
        if(!$statusSaved){
            $statusRows[] = normalize_source_status_entry($probe);
        }
        $USERS_STORE['users'][$sessionUser]['source_status'] = normalize_user_source_status($statusRows);
        $USERS_STORE['users'][$sessionUser]['updated_at'] = date('c');
        save_users_store($USERS_FILE, $USERS_STORE);

        echo json_encode([
            'ok' => !empty($probe['ok']),
            'id' => (string)$probe['id'],
            'name' => (string)$probe['name'],
            'url' => (string)$probe['url'],
            'http' => (int)$probe['http'],
            'latency_ms' => (int)$probe['latency_ms'],
            'device_count' => (int)$probe['device_count'],
            'error' => ($probe['error'] !== '' ? $probe['error'] : null),
            'last_poll_at' => (string)$probe['last_poll_at']
        ]);
        exit;
    }

    if($_GET['ajax']==='token_status'){
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        $configuredSources = get_stored_user_sources($user);
        $effective = get_effective_uisp_sources($user, $UISP_URL, $UISP_TOKEN);
        $source = 'none';
        if(count($configuredSources) > 0){
            $source = 'account_sources';
        } elseif(count($effective) > 0){
            $source = 'server_default';
        }
        echo json_encode([
            'ok' => 1,
            'has_token' => (count($effective) > 0),
            'source_count' => count($configuredSources),
            'source' => $source,
            'username' => $sessionUser
        ]);
        exit;
    }

    if($_GET['ajax']==='save_uisp_token' && $_SERVER['REQUEST_METHOD']==='POST'){
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        if(!$user){
            echo json_encode(['ok'=>0,'error'=>'invalid_session']); exit;
        }
        $token = trim((string)($_POST['token'] ?? ''));
        if($token !== '' && strlen($token) < 12){
            echo json_encode(['ok'=>0,'error'=>'token_too_short']); exit;
        }
        $base = normalize_uisp_url($UISP_URL);
        $sources = get_stored_user_sources($user);
        if($token !== '' && ($base === '' || is_placeholder_uisp_url($base))){
            echo json_encode([
                'ok'=>0,
                'error'=>'uisp_url_required',
                'message'=>'Server UISP_URL is not configured. Use Account Settings to add full UISP sources.'
            ]); exit;
        }

        if($token !== ''){
            $legacyUpdated = false;
            for($i=0; $i<count($sources); $i++){
                if(($sources[$i]['id'] ?? '') === 'legacy-account'){
                    $sources[$i]['token'] = $token;
                    $sources[$i]['url'] = $base;
                    $sources[$i]['enabled'] = true;
                    $sources[$i]['updated_at'] = date('c');
                    $legacyUpdated = true;
                    break;
                }
            }
            if(!$legacyUpdated){
                $sources[] = [
                    'id' => 'legacy-account',
                    'name' => 'Legacy Account UISP',
                    'url' => $base,
                    'token' => $token,
                    'enabled' => true,
                    'created_at' => date('c'),
                    'updated_at' => date('c')
                ];
            }
        }

        $USERS_STORE['users'][$sessionUser]['sources'] = $sources;
        $USERS_STORE['users'][$sessionUser]['uisp_token'] = $token;
        $USERS_STORE['users'][$sessionUser]['updated_at'] = date('c');
        save_users_store($USERS_FILE, $USERS_STORE);
        $effective = get_effective_uisp_sources($USERS_STORE['users'][$sessionUser], $UISP_URL, $UISP_TOKEN);
        $fallbackSource = (count($effective) > 0) ? 'account_sources' : 'none';
        echo json_encode([
            'ok'=>1,
            'has_token'=>($fallbackSource !== 'none'),
            'source'=>$fallbackSource
        ]);
        exit;
    }

    if($_GET['ajax']==='ack' && !empty($_GET['id']) && !empty($_GET['dur'])){
        require_pro_feature('ack');
        $id=$_GET['id']; $dur=$_GET['dur'];
        $durmap=['30m'=>1800,'1h'=>3600,'6h'=>21600,'8h'=>28800,'12h'=>43200];
        $cache[$id]['ack_until']=time()+($durmap[$dur]??1800);
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
    if($_GET['ajax']==='clear' && !empty($_GET['id'])){
        require_pro_feature('ack');
        unset($cache[$_GET['id']]['ack_until']);
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
    if($_GET['ajax']==='simulate' && !empty($_GET['id'])){
        require_pro_feature('simulate');
        $cache[$_GET['id']]['simulate']=true;
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
    if($_GET['ajax']==='clearsim' && !empty($_GET['id'])){
        require_pro_feature('simulate');
        $did = $_GET['id'];
        if(isset($cache[$did]['simulate'])) unset($cache[$did]['simulate']);
        // Proactively clear any outage state created by simulation so UI snaps back immediately
        if(isset($cache[$did]['offline_since'])) unset($cache[$did]['offline_since']);
        if(isset($cache[$did]['gf_last_offline_notif'])) unset($cache[$did]['gf_last_offline_notif']);
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
    if($_GET['ajax']==='clearall'){
        require_pro_feature('ack');
        foreach($cache as $k=>&$c){
            if(is_array($c)){
                if(array_key_exists('ack_until',$c)) unset($c['ack_until']);
            }
        }
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
}

// For main HTML: prevent caching so index.php updates are reflected immediately
if(!isset($_GET['ajax'])){
    header('Cache-Control: no-store, no-cache, must-revalidate, max-age=0');
    header('Pragma: no-cache');
}

if(isset($_GET['view']) && $_GET['view']==='device'){
    if(!isset($_SESSION['auth_ok']) || empty($_SESSION['auth_user'])){
        header('Location: ./?login=1');
        exit;
    }
    if(empty($NOCWALL_FEATURE_FLAGS['history'])){
        header('Location: ./');
        exit;
    }
    $deviceId = trim((string)($_GET['id'] ?? ''));
    if($deviceId === ''){
        header('Location: ./');
        exit;
    }
    $nameHint = trim((string)($_GET['name'] ?? ''));
    $pageTitle = $nameHint !== '' ? $nameHint : $deviceId;
    $ackOptions = ['30m','1h','6h','8h','12h'];
    ?>
<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Device Detail - <?=htmlspecialchars($pageTitle, ENT_QUOTES)?> | UISP NOC</title>
  <link rel="stylesheet" href="assets/style.css?v=<?=$ASSET_VERSION?>">
</head>
<body class="detail-page">
  <header class="detail-header">
    <button class="btn-outline" onclick="window.location.href='./';">&larr; Dashboard</button>
    <div class="detail-title">
      <h1 id="deviceTitle"><?=htmlspecialchars($pageTitle, ENT_QUOTES)?></h1>
      <div id="deviceSubtitle" class="detail-subtitle"></div>
    </div>
    <div class="detail-header-right">
      <span id="detailUpdated" class="detail-updated">Last update: --</span>
    </div>
  </header>

  <main class="detail-main">
    <section class="detail-summary">
      <div class="detail-status-row">
        <span id="detailStatusBadge" class="status-pill status-pill--loading">Loading</span>
        <span id="detailAckBadge" class="status-pill status-pill--ack" style="display:none;"></span>
        <span id="detailOutageBadge" class="status-pill status-pill--outage" style="display:none;"></span>
      </div>
      <div id="detailBadges" class="detail-badges"></div>
      <div id="detailMessage" class="detail-message"></div>
    </section>

    <section class="detail-actions">
      <div class="detail-ack-controls">
        <span class="detail-actions-label">Acknowledge outage:</span>
        <div class="detail-ack-buttons" id="ackButtons">
          <?php foreach($ackOptions as $opt): ?>
            <button class="btn" data-ack="<?=$opt?>">Ack <?=$opt?></button>
          <?php endforeach; ?>
        </div>
        <button class="btn-outline" id="clearAckBtn" style="display:none;">Clear Ack</button>
      </div>
    </section>

    <section class="detail-history">
      <h2>Performance History</h2>
      <div class="chart-grid">
        <canvas id="cpuChart"></canvas>
        <canvas id="ramChart"></canvas>
        <canvas id="tempChart"></canvas>
        <canvas id="latChart"></canvas>
      </div>
      <div id="historyMessage" class="detail-message" style="margin-top:16px;"></div>
    </section>
  </main>

  <footer class="detail-footer" id="detailFooter">
    HTTP --, API latency --, Updated --
  </footer>

  <script>
    window.DEVICE_DETAIL = {
      id: <?=json_encode($deviceId)?>,
      nameHint: <?=json_encode($pageTitle)?>,
      ackOptions: <?=json_encode($ackOptions)?>,
      assetVersion: <?=json_encode($ASSET_VERSION)?>
    };
  </script>
  <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
  <script src="assets/device-detail.js?v=<?=$ASSET_VERSION?>"></script>
</body>
</html>
<?php
    exit;
}

if(isset($_GET['view']) && $_GET['view']==='settings'){
    if(!isset($_SESSION['auth_ok']) || empty($_SESSION['auth_user'])){
        header('Location: ./?login=1');
        exit;
    }
    $authUser = htmlspecialchars((string)($_SESSION['auth_user'] ?? ''), ENT_QUOTES);
    ?>
<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Account Settings | NOCWALL-CE</title>
  <style>
    body{font-family:system-ui,Segoe UI,Arial,sans-serif;background:#111;color:#eee;margin:0}
    header{display:flex;justify-content:space-between;align-items:center;padding:14px 18px;background:#1a1a1a;border-bottom:1px solid #2b2b2b}
    main{padding:18px;max-width:1100px;margin:0 auto}
    .card{background:#1a1a1a;border:1px solid #2e2e2e;border-radius:10px;padding:16px;margin-bottom:14px}
    .grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:10px}
    label{font-size:12px;color:#bdbdbd;display:block;margin-bottom:6px}
    input[type=text],input[type=url],input[type=password]{width:100%;box-sizing:border-box;background:#0f0f0f;border:1px solid #333;color:#eee;border-radius:7px;padding:9px}
    button{background:#2f6fef;color:#fff;border:none;border-radius:7px;padding:9px 12px;cursor:pointer}
    button.secondary{background:#444}
    table{width:100%;border-collapse:collapse}
    th,td{padding:10px;border-bottom:1px solid #2b2b2b;text-align:left;font-size:13px;vertical-align:middle}
    .status{font-size:12px;color:#9adf9a;min-height:20px}
    .warn{color:#ffb870}
    .error{color:#ff8f8f}
    .row-actions{display:flex;gap:6px;flex-wrap:wrap}
    .small{font-size:12px;color:#aaa}
  </style>
</head>
<body>
  <header>
    <div><strong>Account Settings</strong> <span class="small">User: <?=$authUser?></span></div>
    <button class="secondary" onclick="window.location.href='./';">Back To Dashboard</button>
  </header>
  <main>
    <section class="card">
      <h3 style="margin-top:0">Add UISP Source</h3>
      <div class="grid">
        <div>
          <label for="srcName">Source Name</label>
          <input id="srcName" type="text" placeholder="Main UISP">
        </div>
        <div>
          <label for="srcUrl">UISP Base URL</label>
          <input id="srcUrl" type="url" placeholder="https://isp.unmsapp.com" required>
        </div>
        <div>
          <label for="srcToken">UISP API Token</label>
          <input id="srcToken" type="password" placeholder="Paste API token" required>
        </div>
      </div>
      <div style="margin-top:10px;display:flex;align-items:center;gap:10px;flex-wrap:wrap">
        <label style="display:flex;align-items:center;gap:6px;margin:0"><input id="srcEnabled" type="checkbox" checked> Enabled</label>
        <button onclick="saveSource()">Save Source</button>
        <button class="secondary" id="cancelEditBtn" onclick="cancelEdit()" style="display:none">Cancel Edit</button>
      </div>
      <div class="small" style="margin-top:10px">Each account can store multiple UISP endpoints and tokens.</div>
      <div id="settingsStatus" class="status"></div>
    </section>

    <section class="card">
      <h3 style="margin-top:0">Configured UISP Sources</h3>
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>URL</th>
            <th>Status</th>
            <th>Token</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody id="sourcesBody">
          <tr><td colspan="5" class="small">Loading...</td></tr>
        </tbody>
      </table>
    </section>
  </main>

  <script>
    let editSourceId = '';
    let cachedSources = [];
    function setStatus(msg, kind){
      const el = document.getElementById('settingsStatus');
      if(!el) return;
      el.textContent = msg || '';
      el.className = 'status ' + (kind || '');
    }
    function esc(v){
      return String(v ?? '').replace(/[&<>"']/g, s => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[s]));
    }
    function cancelEdit(){
      editSourceId = '';
      document.getElementById('srcName').value = '';
      document.getElementById('srcUrl').value = '';
      document.getElementById('srcToken').value = '';
      document.getElementById('srcEnabled').checked = true;
      document.getElementById('cancelEditBtn').style.display = 'none';
      setStatus('');
    }
    async function loadSources(){
      const body = document.getElementById('sourcesBody');
      const r = await fetch('?ajax=sources_list&t='+Date.now(), {cache:'no-store'});
      if(r.status === 401){ location.href='./?login=1'; return; }
      const j = await r.json().catch(()=>null);
      if(!j || !j.ok){
        body.innerHTML = '<tr><td colspan="5" class="small">Failed to load sources.</td></tr>';
        return;
      }
      cachedSources = Array.isArray(j.sources) ? j.sources : [];
      if(cachedSources.length === 0){
        body.innerHTML = '<tr><td colspan="5" class="small">No UISP sources added yet.</td></tr>';
        return;
      }
      body.innerHTML = cachedSources.map(s=>{
        return `<tr>
          <td>${esc(s.name)}</td>
          <td>${esc(s.url)}</td>
          <td>${s.enabled ? 'Enabled' : 'Disabled'}</td>
          <td>${esc(s.token_hint || '(none)')}</td>
          <td class="row-actions">
            <button class="secondary" onclick="editSource('${esc(s.id)}')">Edit</button>
            <button class="secondary" onclick="testSource('${esc(s.id)}')">Test</button>
            <button class="secondary" onclick="deleteSource('${esc(s.id)}')">Delete</button>
          </td>
        </tr>`;
      }).join('');
    }
    function editSource(id){
      const src = cachedSources.find(x=>x.id===id);
      if(!src) return;
      editSourceId = id;
      document.getElementById('srcName').value = src.name || '';
      document.getElementById('srcUrl').value = src.url || '';
      document.getElementById('srcToken').value = '';
      document.getElementById('srcEnabled').checked = !!src.enabled;
      document.getElementById('cancelEditBtn').style.display = '';
      setStatus('Editing source "' + (src.name || id) + '". Leave token empty to keep existing token.', 'warn');
    }
    async function saveSource(){
      const name = document.getElementById('srcName').value.trim();
      const url = document.getElementById('srcUrl').value.trim();
      const token = document.getElementById('srcToken').value.trim();
      const enabled = document.getElementById('srcEnabled').checked ? '1' : '0';
      if(!url){ setStatus('UISP URL is required.', 'error'); return; }
      if(!editSourceId && !token){ setStatus('UISP API token is required for new sources.', 'error'); return; }
      const fd = new FormData();
      if(editSourceId) fd.append('id', editSourceId);
      fd.append('name', name);
      fd.append('url', url);
      fd.append('token', token);
      fd.append('enabled', enabled);
      const r = await fetch('?ajax=sources_save', { method:'POST', body:fd });
      if(r.status === 401){ location.href='./?login=1'; return; }
      const j = await r.json().catch(()=>null);
      if(!j || !j.ok){
        setStatus('Save failed: ' + ((j && (j.message || j.error)) || 'unknown'), 'error');
        return;
      }
      cancelEdit();
      setStatus('Source saved.');
      loadSources();
    }
    async function deleteSource(id){
      if(!confirm('Delete this UISP source?')) return;
      const fd = new FormData();
      fd.append('id', id);
      const r = await fetch('?ajax=sources_delete', { method:'POST', body:fd });
      if(r.status === 401){ location.href='./?login=1'; return; }
      const j = await r.json().catch(()=>null);
      if(!j || !j.ok){
        setStatus('Delete failed: ' + ((j && (j.error || j.message)) || 'unknown'), 'error');
        return;
      }
      setStatus('Source deleted.');
      loadSources();
    }
    async function testSource(id){
      setStatus('Testing source...');
      const fd = new FormData();
      fd.append('id', id);
      const r = await fetch('?ajax=sources_test', { method:'POST', body:fd });
      if(r.status === 401){ location.href='./?login=1'; return; }
      const j = await r.json().catch(()=>null);
      if(!j){
        setStatus('Test failed: bad response', 'error');
        return;
      }
      if(j.ok){
        setStatus('Test passed. HTTP ' + j.http + ', devices: ' + j.device_count + ', latency: ' + j.latency_ms + 'ms');
      } else {
        setStatus('Test failed. HTTP ' + j.http + (j.error ? (', err: ' + j.error) : ''), 'error');
      }
    }
    loadSources();
  </script>
</body>
</html>
<?php
    exit;
}
?>
<?php if(!isset($_SESSION['auth_ok']) || empty($_SESSION['auth_user'])): ?>
<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>NOCWALL-CE - Login</title>
  <style>
    body{font-family:system-ui,Segoe UI,Arial,sans-serif;background:#111;color:#eee;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
    .wrap{display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:16px;width:min(760px,92vw)}
    .login{background:#1b1b1b;padding:24px;border-radius:8px;box-shadow:0 0 0 1px #333;width:100%}
    .login h2{margin:0 0 12px 0;font-weight:600}
    .field{margin:10px 0}
    .field label{display:block;margin-bottom:6px;color:#ccc;font-size:12px}
    .field input{width:100%;padding:10px;border:1px solid #333;border-radius:6px;background:#0f0f0f;color:#eee}
    .btn{width:100%;padding:10px;margin-top:10px;background:#6c5ce7;border:none;border-radius:6px;color:#fff;font-weight:600;cursor:pointer}
    .hint{color:#aaa;font-size:12px;margin-top:8px}
    .err{color:#f55;margin-bottom:8px;font-size:13px}
  </style>
</head>
<body>
  <div class="wrap">
    <form class="login" method="post" action="?action=login">
      <h2>Sign in</h2>
      <?php if(!empty($_SESSION['auth_err'])){ echo '<div class="err">'.htmlspecialchars($_SESSION['auth_err']).'</div>'; unset($_SESSION['auth_err']); } ?>
      <div class="field">
        <label>Username</label>
        <input type="text" name="username" value="admin" autocomplete="username" required>
      </div>
      <div class="field">
        <label>Password</label>
        <input type="password" name="password" autocomplete="current-password" required>
      </div>
      <button class="btn" type="submit">Login</button>
      <div class="hint">Default bootstrap account is admin/admin until changed.</div>
    </form>

    <form class="login" method="post" action="?action=register">
      <h2>Create account</h2>
      <div class="field">
        <label>Username</label>
        <input type="text" name="username" pattern="[a-z0-9._-]{3,32}" autocomplete="username" required>
      </div>
      <div class="field">
        <label>Password</label>
        <input type="password" name="password" minlength="8" autocomplete="new-password" required>
      </div>
      <div class="field">
        <label>Confirm password</label>
        <input type="password" name="password_confirm" minlength="8" autocomplete="new-password" required>
      </div>
      <button class="btn" type="submit">Create account</button>
      <div class="hint">After signup, open Account Settings and add one or more UISP sources.</div>
    </form>
  </div>
</body>
</html>
<?php exit; endif; ?>
<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>NOCWALL-CE</title>
<link rel="stylesheet" href="assets/style.css?v=<?=$ASSET_VERSION?>">
</head>
<body>
<header>
  <?php $AUTH_USER = htmlspecialchars((string)($_SESSION['auth_user'] ?? ''), ENT_QUOTES); ?>
  <div class="brand">
    <span class="brand-title">NOCWALL-CE<?=!empty($NOCWALL_FEATURE_FLAGS['strict_ce']) ? ' <small style="font-size:12px;color:#f5b87c;">(Strict CE Mode)</small>' : ''?></span>
    <span id="overallSummary"></span>
  </div>
  <div class="header-actions">
    <span class="header-user">User: <?=$AUTH_USER?></span>
    <?php if($SHOW_TLS_UI): ?>
      <button onclick="openTLS()">TLS/Certs</button>
    <?php endif; ?>
    <button onclick="manageUispSources()">Account Settings</button>
    <button id="enableSoundBtn" class="btn-accent" onclick="enableSound()">Enable Sound</button>
    <?php if(!empty($NOCWALL_FEATURE_FLAGS['ack'])): ?>
      <button onclick="clearAll()">Clear All Acks</button>
    <?php endif; ?>
    <button onclick="changePassword()">Change Password</button>
    <button onclick="logout()">Logout</button>
  </div>
</header>
<div class="tabs">
    <button class="tablink active" data-tab="gateways" onclick="openTab('gateways', event)">Gateways</button>
    <button class="tablink" data-tab="aps" onclick="openTab('aps', event)">APs</button>
    <button class="tablink" data-tab="routers" onclick="openTab('routers', event)">Routers & Switches</button>
    <?php if(!empty($NOCWALL_FEATURE_FLAGS['topology'])): ?>
      <button class="tablink" data-tab="topology" onclick="openTab('topology', event)">Topology</button>
    <?php endif; ?>
</div>
<section class="source-status-strip" aria-label="UISP source status">
  <div class="source-status-head">
    <div id="sourceStatusSummary" class="source-status-summary">Checking source health...</div>
    <button id="pollAllSourcesBtn" type="button" class="btn-outline">Poll All Sources</button>
  </div>
  <div id="sourceStatusList" class="source-status-list"></div>
  <div id="sourceStatusNotice" class="source-status-notice"></div>
</section>
<section class="view-controls" aria-label="Device search and filter controls">
  <div class="view-controls-row">
    <label for="deviceSearchInput">Search Devices</label>
    <input id="deviceSearchInput" type="search" placeholder="Name, hostname, MAC, serial, site...">
    <button id="deviceSearchClearBtn" type="button" class="btn-outline">Clear</button>
  </div>
  <div class="view-controls-row">
    <span class="view-controls-label">Quick Filter</span>
    <button id="filterAllBtn" type="button" class="btn-outline filter-btn active" data-filter="all">All</button>
    <button id="filterOnlineBtn" type="button" class="btn-outline filter-btn" data-filter="online">Online</button>
    <button id="filterOfflineBtn" type="button" class="btn-outline filter-btn" data-filter="offline">Offline</button>
    <label for="sortModeSelect" class="view-controls-label">Sort</label>
    <select id="sortModeSelect">
      <option value="manual">Manual</option>
      <option value="status_name">Status + Name</option>
      <option value="name_asc">Name (A-Z)</option>
      <option value="last_seen_desc">Last Seen (Newest)</option>
    </select>
  </div>
  <div id="viewControlsSummary" class="view-controls-summary"></div>
</section>
<?php if(!empty($NOCWALL_FEATURE_FLAGS['display_controls'])): ?>
  <section class="display-controls" aria-label="Dashboard display controls">
    <div class="display-controls-title">Display Controls</div>
    <div class="display-controls-row">
      <label for="settingDensity">Card Density</label>
      <select id="settingDensity">
        <option value="normal">Normal</option>
        <option value="compact">Compact</option>
      </select>
      <button id="settingReset" type="button" class="btn-outline">Reset</button>
    </div>
    <div class="display-controls-row">
      <span class="display-controls-label">Visible Metrics</span>
      <label><input type="checkbox" id="settingMetricCpu" checked> CPU</label>
      <label><input type="checkbox" id="settingMetricRam" checked> RAM</label>
      <label><input type="checkbox" id="settingMetricTemp" checked> Temp</label>
      <label><input type="checkbox" id="settingMetricLatency" checked> Latency</label>
      <label><input type="checkbox" id="settingMetricUptime" checked> Uptime</label>
      <label><input type="checkbox" id="settingMetricOutage" checked> Outage</label>
    </div>
  </section>
<?php endif; ?>
<div id="gateways" class="tabcontent" style="display:block">
  <?php if(!empty($NOCWALL_FEATURE_FLAGS['advanced_actions'])): ?>
    <div class="grid-actions">
      <button id="gatewaySirenToggleBtn" type="button" class="btn-outline">Gateways Siren: On</button>
    </div>
  <?php endif; ?>
  <div id="gateGrid" class="grid"></div>
</div>
<div id="aps" class="tabcontent" style="display:none">
  <?php if(!empty($NOCWALL_FEATURE_FLAGS['advanced_actions'])): ?>
    <div class="grid-actions">
      <button id="apTabSirenToggleBtn" type="button" class="btn-outline">APs Siren: On</button>
    </div>
  <?php endif; ?>
  <div id="apGrid" class="grid"></div>
</div>
<div id="routers" class="tabcontent" style="display:none">
  <?php if(!empty($NOCWALL_FEATURE_FLAGS['advanced_actions'])): ?>
    <div class="grid-actions">
      <button id="routerSirenToggleBtn" type="button" class="btn-outline">Routers/Switches Siren: On</button>
    </div>
  <?php endif; ?>
  <div id="routerGrid" class="grid"></div>
</div>
<?php if(!empty($NOCWALL_FEATURE_FLAGS['topology'])): ?>
  <div id="topology" class="tabcontent" style="display:none">
    <div class="topology-toolbar">
      <button id="topologyRefreshBtn" type="button" class="btn-outline">Refresh Topology</button>
      <label>Source
        <select id="topologySourceSelect"></select>
      </label>
      <label>Target
        <select id="topologyTargetSelect"></select>
      </label>
      <button id="topologyTraceBtn" type="button" class="btn-outline">Trace Path</button>
      <button id="topologyClearTraceBtn" type="button" class="btn-outline">Clear Trace</button>
    </div>
    <div id="topologyHealthSummary" class="topology-health"></div>
    <div id="topologyStatus" class="topology-status">Loading topology...</div>
    <div id="topologyCanvasWrap" class="topology-canvas-wrap">
      <svg id="topologySvg" viewBox="0 0 1200 680" preserveAspectRatio="xMidYMid meet"></svg>
    </div>
  </div>
<?php endif; ?>
<footer id="footer"></footer>

<div id="tlsModal" class="modal">
  <div class="modal-content">
    <h3>TLS / Certificates</h3>
    <button class="modal-close" onclick="closeTLS()" aria-label="Close">&times;</button>
    <p>Provision HTTPS certificates via Caddy. Ensure your DNS points to this host and ports 80/443 are reachable from the internet.</p>
    <form id="tlsForm" onsubmit="return submitTLS();" style="display:block;max-width:560px">
      <label>Site Domain (UI)</label>
      <input id="tlsDomain" type="text" placeholder="noc.example.com" style="width:100%;padding:8px;border-radius:6px;border:1px solid #444;background:#111;color:#eee" required>
      <div style="height:8px"></div>
      <label>Gotify Domain (optional)</label>
      <input id="tlsGotify" type="text" placeholder="gotify.example.com" style="width:100%;padding:8px;border-radius:6px;border:1px solid #444;background:#111;color:#eee">
      <div style="height:8px"></div>
      <label>ACME Email</label>
      <input id="tlsEmail" type="email" placeholder="admin@example.com" style="width:100%;padding:8px;border-radius:6px;border:1px solid #444;background:#111;color:#eee" required>
      <div><label><input id="tlsStaging" type="checkbox"> Use Let's Encrypt Staging (testing)</label></div>
      <div style="height:10px"></div>
      <button class="btn-accent" type="submit">Provision / Reload Caddy</button>
    </form>
    <pre id="tlsStatus" style="background:#111;border:1px solid #333;border-radius:8px;padding:10px;color:#8ad;overflow:auto;max-height:260px"></pre>
  </div>
 </div>

<?php if(!empty($NOCWALL_FEATURE_FLAGS['cpe_history'])): ?>
  <div id="cpeHistoryModal" class="modal" onclick="if(event.target===this) closeCpeHistory();">
    <div class="modal-content">
      <button class="modal-close" onclick="closeCpeHistory()" aria-label="Close">&times;</button>
      <h3 id="cpeHistoryTitle">Station Ping History</h3>
      <p id="cpeHistorySubtitle" class="modal-subtitle">All recorded station pings for the last 7 days.</p>
      <div id="cpeHistoryStatus" class="history-status">Click "View All Station Ping History" to load samples.</div>
      <div class="history-chart-wrap">
        <canvas id="cpeHistoryChart" width="900" height="320"></canvas>
      </div>
      <div id="cpeHistoryEmpty" class="history-empty" style="display:none;">No ping samples recorded for this period.</div>
      <div id="cpeHistoryStats" class="history-stats"></div>
    </div>
   </div>
<?php endif; ?>

<?php if(!empty($NOCWALL_FEATURE_FLAGS['inventory'])): ?>
  <div id="inventoryModal" class="modal" onclick="if(event.target===this) closeInventory();">
    <div class="modal-content inventory-modal">
      <button class="modal-close" onclick="closeInventory()" aria-label="Close">&times;</button>
      <h3 id="inventoryTitle">Inventory</h3>
      <div id="inventoryStatus" class="inventory-status">Loading inventory...</div>
      <section class="inventory-section">
        <h4>Identity</h4>
        <div id="inventoryIdentity" class="inventory-identity"></div>
      </section>
      <section class="inventory-section">
        <h4>Interfaces</h4>
        <div id="inventoryInterfaces"></div>
      </section>
      <section class="inventory-section">
        <h4>Neighbors</h4>
        <div id="inventoryNeighbors"></div>
      </section>
      <section class="inventory-section">
        <h4>Drift</h4>
        <div id="inventoryDrift"></div>
      </section>
    </div>
  </div>
<?php endif; ?>

<div id="setupWizardModal" class="modal" aria-hidden="true">
  <div class="modal-content wizard-modal">
    <h3 style="margin:0 0 8px">Welcome to NOCWALL-CE</h3>
    <p class="wizard-subtitle">Add your first UISP source to start loading device telemetry on this dashboard.</p>
    <form id="setupWizardForm" onsubmit="return false;">
      <div class="wizard-grid">
        <div>
          <label for="wizardSourceName">Source Name</label>
          <input id="wizardSourceName" type="text" placeholder="Main UISP">
        </div>
        <div>
          <label for="wizardSourceUrl">UISP Base URL</label>
          <input id="wizardSourceUrl" type="url" placeholder="https://isp.unmsapp.com" required>
        </div>
        <div>
          <label for="wizardSourceToken">UISP API Token</label>
          <input id="wizardSourceToken" type="password" placeholder="Paste API token" required>
        </div>
      </div>
      <div class="wizard-actions">
        <button id="wizardSaveTestBtn" class="btn-accent" type="button">Save and Test Connection</button>
        <button id="wizardSkipBtn" class="btn-outline" type="button">Skip for Now</button>
        <button id="wizardOpenSettingsBtn" class="btn-outline" type="button">Open Full Settings</button>
      </div>
      <div id="setupWizardStatus" class="wizard-status"></div>
    </form>
  </div>
</div>

<audio id="siren" src="buz.mp3?v=<?=$ASSET_VERSION?>" preload="auto"></audio>

<script>
  window.NOCWALL_FEATURES = <?=json_encode($NOCWALL_FEATURE_FLAGS, JSON_UNESCAPED_SLASHES)?>;
</script>
<script src="assets/app.js?v=<?=$ASSET_VERSION?>"></script>
</body>
</html>




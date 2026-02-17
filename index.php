<?php
error_reporting(E_ALL);
ini_set('display_errors', 1);
session_start();

date_default_timezone_set('America/Chicago');

// Config
$UISP_URL   = getenv("UISP_URL") ?: "https://changeme.unmsapp.com";
$UISP_TOKEN = getenv("UISP_TOKEN") ?: "changeme";
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

function default_admin_user(){
    $now = date('c');
    return [
        'username' => 'admin',
        'password_hash' => password_hash('admin', PASSWORD_DEFAULT),
        'uisp_token' => '',
        'created_at' => $now,
        'updated_at' => $now
    ];
}

function bootstrap_users_store($usersFile, $legacyAuthFile){
    $store = read_json_file($usersFile);
    if(is_array($store) && isset($store['users']) && is_array($store['users']) && count($store['users']) > 0){
        return $store;
    }

    $users = [];
    $legacy = read_json_file($legacyAuthFile);
    if(is_array($legacy) && !empty($legacy['username']) && !empty($legacy['password_hash'])){
        $username = normalize_username($legacy['username']);
        $now = date('c');
        $users[$username] = [
            'username' => $username,
            'password_hash' => (string)$legacy['password_hash'],
            'uisp_token' => '',
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

function get_effective_uisp_token($user, $envToken){
    $userToken = trim((string)($user['uisp_token'] ?? ''));
    if($userToken !== '') return $userToken;
    return trim((string)$envToken);
}

// Simple users store with legacy auth migration from auth.json.
$USERS_STORE = bootstrap_users_store($USERS_FILE, $AUTH_FILE);

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

// AJAX
if(isset($_GET['ajax'])){
    require_login_for_ajax();
    $currentUser = get_session_user($USERS_STORE);
    $effectiveUispToken = get_effective_uisp_token($currentUser, $UISP_TOKEN);
    header("Content-Type: application/json");
    // Prevent caching of AJAX responses
    header('Cache-Control: no-store, no-cache, must-revalidate, max-age=0');
    header('Pragma: no-cache');

    if($_GET['ajax']==='mobile_config'){
        $base = rtrim((string)$UISP_URL, '/');
        if(!$effectiveUispToken || $effectiveUispToken === 'changeme'){
            http_response_code(503);
            echo json_encode([
                'error' => 'uisp_token_not_configured',
                'message' => 'UISP token has not been configured for this account.'
            ]);
            exit;
        }
        echo json_encode([
            'uisp_base_url' => $base,
            'uisp_token' => $effectiveUispToken,
            'issued_at' => date('c')
        ]);
        exit;
    }

    if($_GET['ajax']==='devices'){
        if(!$effectiveUispToken || $effectiveUispToken === 'changeme'){
            http_response_code(503);
            echo json_encode([
                'devices' => [],
                'http' => 503,
                'api_latency' => 0,
                'error' => 'uisp_token_not_configured',
                'message' => 'Set your UISP API token from the dashboard header.'
            ]);
            exit;
        }
        $ch=curl_init();
        $start=microtime(true);
        curl_setopt_array($ch,[
            CURLOPT_URL=>rtrim($UISP_URL,"/")."/nms/api/v2.1/devices",
            CURLOPT_RETURNTRANSFER=>true,
            CURLOPT_HTTPHEADER=>["accept: application/json","x-auth-token: $effectiveUispToken"],
            CURLOPT_TIMEOUT=>10
        ]);
        $resp=curl_exec($ch);
        $api_latency=round((microtime(true)-$start)*1000);
        $http_code=curl_getinfo($ch,CURLINFO_HTTP_CODE);
        curl_close($ch);
        $devices=json_decode($resp,true)?:[];

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
                'online'=>$on,'cpu'=>$cpu,'ram'=>$ram,'temp'=>$temp,'latency'=>$lat,
                'cpe_latency'=>$cpe_lat,
                'uptime'=>$uptime,
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

    if($_GET['ajax']==='history' && !empty($_GET['id'])){
        $id=$_GET['id'];
        $stmt=$db->prepare("SELECT timestamp,cpu,ram,temp,latency FROM metrics WHERE device_id=? ORDER BY timestamp DESC LIMIT 1440");
        $stmt->bindValue(1,$id,SQLITE3_TEXT);
        $res=$stmt->execute();
        $rows=[];
        while($r=$res->fetchArray(SQLITE3_ASSOC)) $rows[]=$r;
        echo json_encode(array_reverse($rows)); exit;
    }
    if($_GET['ajax']==='cpe_history'){
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

    if($_GET['ajax']==='token_status'){
        $sessionUser = normalize_username($_SESSION['auth_user'] ?? '');
        $user = get_user_by_username($USERS_STORE, $sessionUser);
        $userToken = trim((string)($user['uisp_token'] ?? ''));
        $envToken = trim((string)$UISP_TOKEN);
        $source = 'none';
        if($userToken !== ''){
            $source = 'account';
        } elseif($envToken !== '' && $envToken !== 'changeme'){
            $source = 'server_default';
        }
        echo json_encode([
            'ok' => 1,
            'has_token' => ($source !== 'none'),
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
        $USERS_STORE['users'][$sessionUser]['uisp_token'] = $token;
        $USERS_STORE['users'][$sessionUser]['updated_at'] = date('c');
        save_users_store($USERS_FILE, $USERS_STORE);
        $envToken = trim((string)$UISP_TOKEN);
        $fallbackSource = ($envToken !== '' && $envToken !== 'changeme') ? 'server_default' : 'none';
        echo json_encode([
            'ok'=>1,
            'has_token'=>($token !== '' || $fallbackSource !== 'none'),
            'source'=>($token !== '' ? 'account' : $fallbackSource)
        ]);
        exit;
    }

    if($_GET['ajax']==='ack' && !empty($_GET['id']) && !empty($_GET['dur'])){
        $id=$_GET['id']; $dur=$_GET['dur'];
        $durmap=['30m'=>1800,'1h'=>3600,'6h'=>21600,'8h'=>28800,'12h'=>43200];
        $cache[$id]['ack_until']=time()+($durmap[$dur]??1800);
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
    if($_GET['ajax']==='clear' && !empty($_GET['id'])){
        unset($cache[$_GET['id']]['ack_until']);
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
    if($_GET['ajax']==='simulate' && !empty($_GET['id'])){
        $cache[$_GET['id']]['simulate']=true;
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
    if($_GET['ajax']==='clearsim' && !empty($_GET['id'])){
        $did = $_GET['id'];
        if(isset($cache[$did]['simulate'])) unset($cache[$did]['simulate']);
        // Proactively clear any outage state created by simulation so UI snaps back immediately
        if(isset($cache[$did]['offline_since'])) unset($cache[$did]['offline_since']);
        if(isset($cache[$did]['gf_last_offline_notif'])) unset($cache[$did]['gf_last_offline_notif']);
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
    if($_GET['ajax']==='clearall'){
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
      <div class="hint">After signup, import your UISP API token from the dashboard header.</div>
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
    <span class="brand-title">NOCWALL-CE</span>
    <span id="overallSummary"></span>
  </div>
  <div class="header-actions">
    <span class="header-user">User: <?=$AUTH_USER?></span>
    <?php if($SHOW_TLS_UI): ?>
      <button onclick="openTLS()">TLS/Certs</button>
    <?php endif; ?>
    <button onclick="manageUispToken()">UISP Token</button>
    <button id="enableSoundBtn" class="btn-accent" onclick="enableSound()">Enable Sound</button>
    <button onclick="clearAll()">Clear All Acks</button>
    <button onclick="changePassword()">Change Password</button>
    <button onclick="logout()">Logout</button>
  </div>
</header>
<div class="tabs">
    <button class="tablink active" onclick="openTab('gateways', event)">Gateways</button>
    <button class="tablink" onclick="openTab('aps', event)">APs</button>
    <button class="tablink" onclick="openTab('routers', event)">Routers & Switches</button>
</div>
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
<div id="gateways" class="tabcontent" style="display:block"><div id="gateGrid" class="grid"></div></div>
<div id="aps" class="tabcontent" style="display:none">
  <div class="grid-actions"></div>
  <div id="apGrid" class="grid"></div>
</div>
<div id="routers" class="tabcontent" style="display:none"><div id="routerGrid" class="grid"></div></div>
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

<audio id="siren" src="buz.mp3?v=<?=$ASSET_VERSION?>" preload="auto"></audio>

<script src="assets/app.js?v=<?=$ASSET_VERSION?>"></script>
</body>
</html>




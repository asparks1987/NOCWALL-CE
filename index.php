<?php
error_reporting(E_ALL);
ini_set('display_errors', 1);

date_default_timezone_set('America/Chicago');

// Config
$UISP_URL   = getenv("UISP_URL") ?: "https://changeme.unmsapp.com";
$UISP_TOKEN = getenv("UISP_TOKEN") ?: "changeme";

$CACHE_DIR  = __DIR__ . "/cache";
$CACHE_FILE = $CACHE_DIR . "/status_cache.json";
$DB_FILE    = $CACHE_DIR . "/metrics.sqlite";

$FIRST_OFFLINE_THRESHOLD = 30;

// Ensure cache dir
if (!is_dir($CACHE_DIR)) mkdir($CACHE_DIR, 0775, true);

// Load cache
$cache = file_exists($CACHE_FILE) ? json_decode(file_get_contents($CACHE_FILE), true) : [];
if (!is_array($cache)) $cache = [];

// SQLite init
$db = new SQLite3($DB_FILE);
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
function is_gateway($d){ return strtolower($d['identification']['role']??'')==="gateway"; }
function is_online($d){ $s=strtolower($d['overview']['status']??''); return in_array($s,['ok','online','active','connected','reachable','enabled']); }
function ping_host($ip){ if(!$ip) return null; $ip=preg_replace('/\/\d+$/','',$ip); $out=@shell_exec("ping -c 1 -W 1 ".escapeshellarg($ip)." 2>&1"); if(preg_match('/time=([\d\.]+)\s*ms/',$out,$m)) return floatval($m[1]); return null; }

// AJAX
if(isset($_GET['ajax'])){
    header("Content-Type: application/json");
    // Prevent caching of AJAX responses
    header('Cache-Control: no-store, no-cache, must-revalidate, max-age=0');
    header('Pragma: no-cache');

    if($_GET['ajax']==='devices'){
        $ch=curl_init();
        $start=microtime(true);
        curl_setopt_array($ch,[
            CURLOPT_URL=>rtrim($UISP_URL,"/")."/nms/api/v2.1/devices",
            CURLOPT_RETURNTRANSFER=>true,
            CURLOPT_HTTPHEADER=>["accept: application/json","x-auth-token: $UISP_TOKEN"],
            CURLOPT_TIMEOUT=>10
        ]);
        $resp=curl_exec($ch);
        $api_latency=round((microtime(true)-$start)*1000);
        $http_code=curl_getinfo($ch,CURLINFO_HTTP_CODE);
        curl_close($ch);
        $devices=json_decode($resp,true)?:[];

        $now=time();
        $out=[];
        $cache_changed=false;
        foreach($devices as $d){
            $id=device_key($d);
            $name=$d['identification']['name']??$id;
            $isGw=is_gateway($d);
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

            if($isGw){
                $lat=ping_host($d['ipAddress']??null);
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
            }

            $sim=!empty($cache[$id]['simulate']);
            if($sim) $on=false;

            // Track offline start time to compute outage duration
            if(!isset($cache[$id])) $cache[$id]=[];
            if(!$on){
                if(empty($cache[$id]['offline_since'])){ $cache[$id]['offline_since']=$now; $cache_changed=true; }
                $offline_since=$cache[$id]['offline_since'];
            } else {
                if(!empty($cache[$id]['offline_since'])){ unset($cache[$id]['offline_since']); $cache_changed=true; }
                $offline_since=null;
            }

            $ack_until=$cache[$id]['ack_until']??null;

            $out[]=[
                'id'=>$id,'name'=>$name,'gateway'=>$isGw,
                'online'=>$on,'cpu'=>$cpu,'ram'=>$ram,'temp'=>$temp,'latency'=>$lat,
                'uptime'=>$uptime,
                'offline_since'=>$offline_since,
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
        $cache[$_GET['id']]['simulate']=false;
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
    if($_GET['ajax']==='clearall'){
        foreach($cache as &$c) $c['ack_until']=null;
        file_put_contents($CACHE_FILE,json_encode($cache));
        echo json_encode(['ok'=>1]); exit;
    }
}

// For main HTML: prevent caching so index.php updates are reflected immediately
if(!isset($_GET['ajax'])){
    header('Cache-Control: no-store, no-cache, must-revalidate, max-age=0');
    header('Pragma: no-cache');
}
?>
<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>UISP NOC</title>
<link rel="stylesheet" href="assets/style.css?v=<?=$ASSET_VERSION?>">
</head>
<body>
<header>
  UISP NOC 
  <span id="overallSummary"></span>
  <button id="enableSoundBtn" class="btn-accent" onclick="enableSound()" style="float:right;margin-right:10px;">Enable Sound</button>
  <button onclick="clearAll()" style="float:right;margin-right:10px;">Clear All Acks</button>
</header>
<div class="tabs">
  <button class="tablink active" onclick="openTab('gateways')">Gateways</button>
  <button class="tablink" onclick="openTab('cpes')">CPEs</button>
</div>
<div id="gateways" class="tabcontent" style="display:block"><div id="gateGrid" class="grid"></div></div>
<div id="cpes" class="tabcontent" style="display:none"><div id="cpeGrid" class="grid"></div></div>
<footer id="footer"></footer>

<div id="historyModal" class="modal">
  <div class="modal-content">
    <h3 id="histTitle"></h3>
    <button class="modal-close" onclick="closeModal()" aria-label="Close">&times;</button>
    <canvas id="cpuChart"></canvas>
    <canvas id="ramChart"></canvas>
    <canvas id="tempChart"></canvas>
    <canvas id="latChart"></canvas>
    <button onclick="closeModal()">Close</button>
  </div>
</div>

<audio id="siren" src="buz.mp3?v=<?=$ASSET_VERSION?>" preload="auto"></audio>

<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
<script src="assets/app.js?v=<?=$ASSET_VERSION?>"></script>
</body>
</html>

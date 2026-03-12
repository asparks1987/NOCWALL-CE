#!/usr/bin/env php
<?php
error_reporting(E_ALL);
ini_set('display_errors', '1');

function cli_write($text = ''){
    fwrite(STDOUT, (string)$text . PHP_EOL);
}

function cli_error($text){
    fwrite(STDERR, (string)$text . PHP_EOL);
}

function read_json_file($file){
    if(!is_file($file)) return null;
    $raw = @file_get_contents($file);
    if($raw === false || trim((string)$raw) === '') return null;
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

function normalize_beta_key_code($value){
    $code = strtoupper(trim((string)$value));
    if($code === ''){
        return '';
    }
    $code = preg_replace('/[^A-Z0-9-]+/', '', $code);
    $code = trim((string)$code, '-');
    if($code === ''){
        return '';
    }
    if(strlen($code) < 8 || strlen($code) > 64){
        return '';
    }
    return $code;
}

function beta_key_expiration_ts($value){
    $raw = trim((string)$value);
    if($raw === '') return 0;
    $ts = (int)strtotime($raw);
    return ($ts > 0) ? $ts : 0;
}

function beta_key_is_expired($key, $nowTs = null){
    $ts = beta_key_expiration_ts((string)($key['expires_at'] ?? ''));
    if($ts <= 0) return false;
    $now = ($nowTs === null) ? time() : (int)$nowTs;
    return $ts < $now;
}

function default_user_beta_access_state(){
    return [
        'code' => '',
        'status' => 'none',
        'redeemed_at' => '',
        'expires_at' => '',
        'updated_at' => ''
    ];
}

function normalize_user_beta_access_state($input){
    $base = default_user_beta_access_state();
    if(!is_array($input)) return $base;
    $code = normalize_beta_key_code((string)($input['code'] ?? ''));
    if($code !== ''){
        $base['code'] = $code;
    }
    $status = strtolower(trim((string)($input['status'] ?? '')));
    if(in_array($status, ['none','active','revoked','expired'], true)){
        $base['status'] = $status;
    }
    foreach(['redeemed_at','expires_at','updated_at'] as $k){
        if(array_key_exists($k, $input)){
            $base[$k] = trim((string)$input[$k]);
        }
    }
    if($base['code'] === '' && $base['status'] !== 'none'){
        $base['status'] = 'none';
    }
    return $base;
}

function default_beta_key_state($code = ''){
    $now = date('c');
    return [
        'code' => normalize_beta_key_code($code),
        'status' => 'active', // active|disabled|expired
        'max_redemptions' => 1,
        'redeemed_count' => 0,
        'expires_at' => '',
        'issued_by' => '',
        'notes' => '',
        'created_at' => $now,
        'updated_at' => $now,
        'redemptions' => []
    ];
}

function normalize_beta_key_redemptions($input){
    $out = [];
    if(!is_array($input)) return $out;
    foreach($input as $k => $v){
        $username = normalize_username($k);
        $redeemedAt = trim((string)$v);
        if($username === ''){
            if(is_array($v)){
                $username = normalize_username((string)($v['username'] ?? ''));
                $redeemedAt = trim((string)($v['redeemed_at'] ?? ''));
            } else {
                continue;
            }
        }
        if($username === ''){
            continue;
        }
        $out[$username] = ($redeemedAt !== '' ? $redeemedAt : date('c'));
    }
    ksort($out);
    return $out;
}

function normalize_beta_key_state($input, $fallbackCode = ''){
    $base = default_beta_key_state($fallbackCode);
    if(!is_array($input)){
        return $base;
    }
    $code = normalize_beta_key_code((string)($input['code'] ?? $fallbackCode));
    if($code === ''){
        return $base;
    }
    $base['code'] = $code;
    $status = strtolower(trim((string)($input['status'] ?? 'active')));
    if(in_array($status, ['active','disabled','expired'], true)){
        $base['status'] = $status;
    }
    $maxRedemptions = (int)($input['max_redemptions'] ?? 1);
    if($maxRedemptions < 1) $maxRedemptions = 1;
    if($maxRedemptions > 1000000) $maxRedemptions = 1000000;
    $base['max_redemptions'] = $maxRedemptions;
    foreach(['expires_at','issued_by','notes','created_at','updated_at'] as $k){
        if(array_key_exists($k, $input)){
            $base[$k] = trim((string)$input[$k]);
        }
    }
    $base['redemptions'] = normalize_beta_key_redemptions($input['redemptions'] ?? []);
    $base['redeemed_count'] = count($base['redemptions']);
    if($base['redeemed_count'] > $base['max_redemptions']){
        $base['redeemed_count'] = $base['max_redemptions'];
    }
    if($base['status'] === 'active' && beta_key_is_expired($base)){
        $base['status'] = 'expired';
    }
    if(trim((string)$base['updated_at']) === ''){
        $base['updated_at'] = date('c');
    }
    if(trim((string)$base['created_at']) === ''){
        $base['created_at'] = (string)$base['updated_at'];
    }
    return $base;
}

function normalize_beta_keys_store($input){
    $out = [];
    if(!is_array($input)) return $out;
    foreach($input as $k => $row){
        $code = normalize_beta_key_code($k);
        if($code === '' && is_array($row)){
            $code = normalize_beta_key_code((string)($row['code'] ?? ''));
        }
        if($code === ''){
            continue;
        }
        $out[$code] = normalize_beta_key_state($row, $code);
    }
    ksort($out);
    return $out;
}

function normalize_beta_key_audit_events($input){
    $out = [];
    if(!is_array($input)) return $out;
    foreach($input as $row){
        if(!is_array($row)) continue;
        $event = trim((string)($row['event'] ?? ''));
        $code = normalize_beta_key_code((string)($row['code'] ?? ''));
        $at = trim((string)($row['at'] ?? ''));
        if($event === '' || $code === '' || $at === ''){
            continue;
        }
        $out[] = [
            'event' => $event,
            'code' => $code,
            'username' => normalize_username((string)($row['username'] ?? '')),
            'actor' => trim((string)($row['actor'] ?? '')),
            'at' => $at,
            'meta' => is_array($row['meta'] ?? null) ? $row['meta'] : []
        ];
    }
    return $out;
}

function append_beta_key_audit_event(&$store, $event, $code, $username = '', $actor = '', $meta = []){
    $evt = trim((string)$event);
    $normalizedCode = normalize_beta_key_code($code);
    if($evt === '' || $normalizedCode === ''){
        return;
    }
    if(!isset($store['beta_key_audit']) || !is_array($store['beta_key_audit'])){
        $store['beta_key_audit'] = [];
    }
    $store['beta_key_audit'][] = [
        'event' => $evt,
        'code' => $normalizedCode,
        'username' => normalize_username($username),
        'actor' => trim((string)$actor),
        'at' => date('c'),
        'meta' => is_array($meta) ? $meta : []
    ];
    if(count($store['beta_key_audit']) > 1000){
        $store['beta_key_audit'] = array_slice($store['beta_key_audit'], -1000);
    }
}

function load_users_store($path){
    $store = read_json_file($path);
    if(!is_array($store)){
        $store = [];
    }
    if(!isset($store['users']) || !is_array($store['users'])){
        $store['users'] = [];
    }
    if(!isset($store['beta_keys']) || !is_array($store['beta_keys'])){
        $store['beta_keys'] = [];
    }
    if(!isset($store['beta_key_audit']) || !is_array($store['beta_key_audit'])){
        $store['beta_key_audit'] = [];
    }
    $store['beta_keys'] = normalize_beta_keys_store($store['beta_keys']);
    $store['beta_key_audit'] = normalize_beta_key_audit_events($store['beta_key_audit']);
    foreach($store['users'] as $uname => $user){
        if(!is_array($user)){
            continue;
        }
        $store['users'][$uname]['beta_access'] = normalize_user_beta_access_state($user['beta_access'] ?? null);
    }
    return $store;
}

function save_users_store($path, &$store){
    $store['updated_at'] = date('c');
    return write_json_file($path, $store);
}

function set_beta_key_status(&$store, $rawCode, $nextStatus, $actor = 'ops', $reason = ''){
    $code = normalize_beta_key_code($rawCode);
    if($code === ''){
        return false;
    }
    $normalizedStatus = strtolower(trim((string)$nextStatus));
    if($normalizedStatus === 'revoked'){
        $normalizedStatus = 'disabled';
    }
    if(!in_array($normalizedStatus, ['active','disabled','expired'], true)){
        return false;
    }
    $store['beta_keys'] = normalize_beta_keys_store($store['beta_keys'] ?? []);
    if(!isset($store['beta_keys'][$code]) || !is_array($store['beta_keys'][$code])){
        return false;
    }
    $now = date('c');
    $key = normalize_beta_key_state($store['beta_keys'][$code], $code);
    $key['status'] = $normalizedStatus;
    $key['updated_at'] = $now;
    $store['beta_keys'][$code] = $key;

    if(isset($store['users']) && is_array($store['users'])){
        foreach($key['redemptions'] as $uname => $redeemedAt){
            if(!isset($store['users'][$uname]) || !is_array($store['users'][$uname])){
                continue;
            }
            $access = normalize_user_beta_access_state($store['users'][$uname]['beta_access'] ?? null);
            if($access['code'] !== $code){
                continue;
            }
            if($normalizedStatus === 'active'){
                $access['status'] = 'active';
            } elseif($normalizedStatus === 'expired'){
                $access['status'] = 'expired';
            } else {
                $access['status'] = 'revoked';
            }
            if(trim((string)$access['redeemed_at']) === ''){
                $access['redeemed_at'] = trim((string)$redeemedAt);
            }
            $access['expires_at'] = trim((string)$key['expires_at']);
            $access['updated_at'] = $now;
            $store['users'][$uname]['beta_access'] = $access;
            $store['users'][$uname]['updated_at'] = $now;
        }
    }

    append_beta_key_audit_event(
        $store,
        ($normalizedStatus === 'active' ? 'beta_key_enable' : 'beta_key_revoke'),
        $code,
        '',
        $actor,
        ['status' => $normalizedStatus, 'reason' => trim((string)$reason)]
    );
    return true;
}

function generate_beta_key_code($prefix = 'NOCWALL-BETA'){
    $cleanPrefix = strtoupper(trim((string)$prefix));
    $cleanPrefix = preg_replace('/[^A-Z0-9-]+/', '-', $cleanPrefix);
    $cleanPrefix = trim((string)$cleanPrefix, '-');
    if($cleanPrefix === ''){
        $cleanPrefix = 'NOCWALL-BETA';
    }
    $raw = strtoupper(bin2hex(random_bytes(6)));
    $chunks = str_split($raw, 4);
    return $cleanPrefix . '-' . implode('-', $chunks);
}

function parse_cli_options($args){
    $out = ['_' => []];
    for($i=0; $i<count($args); $i++){
        $arg = (string)$args[$i];
        if(strpos($arg, '--') !== 0){
            $out['_'][] = $arg;
            continue;
        }
        $pair = substr($arg, 2);
        if($pair === ''){
            continue;
        }
        if(strpos($pair, '=') !== false){
            [$k, $v] = explode('=', $pair, 2);
            $out[strtolower(trim($k))] = $v;
            continue;
        }
        $next = ($i + 1 < count($args)) ? (string)$args[$i + 1] : '';
        if($next !== '' && strpos($next, '--') !== 0){
            $out[strtolower(trim($pair))] = $next;
            $i++;
        } else {
            $out[strtolower(trim($pair))] = true;
        }
    }
    return $out;
}

function usage(){
    cli_write('Usage: php scripts/beta_keys.php <command> [options]');
    cli_write('');
    cli_write('Commands:');
    cli_write('  generate   Create a new beta key');
    cli_write('  list       List beta keys');
    cli_write('  disable    Disable/revoke a beta key');
    cli_write('  enable     Re-enable a beta key');
    cli_write('');
    cli_write('Common options:');
    cli_write('  --users-file <path>      Path to users store (default: cache/users.json)');
    cli_write('  --json                   Output JSON');
    cli_write('');
    cli_write('generate options:');
    cli_write('  --code <value>           Optional custom key code');
    cli_write('  --prefix <value>         Prefix for generated key (default: NOCWALL-BETA)');
    cli_write('  --max-redemptions <n>    Redemption cap (default: 1)');
    cli_write('  --expires-at <iso8601>   Expiration timestamp');
    cli_write('  --ttl-days <n>           Expire key in N days');
    cli_write('  --issued-by <value>      Issuer value');
    cli_write('  --notes <value>          Notes field');
    cli_write('');
    cli_write('disable/enable options:');
    cli_write('  --code <value>           Key code (required)');
    cli_write('  --reason <value>         Optional reason for audit trail');
    cli_write('  --actor <value>          Actor for audit trail (default: cli)');
}

$argvItems = $_SERVER['argv'] ?? [];
array_shift($argvItems);
$command = strtolower(trim((string)($argvItems[0] ?? 'help')));
if($command === '' || in_array($command, ['help','-h','--help'], true)){
    usage();
    exit(0);
}
array_shift($argvItems);
$opts = parse_cli_options($argvItems);
$jsonMode = !empty($opts['json']);
$usersFile = trim((string)($opts['users-file'] ?? (__DIR__ . '/../cache/users.json')));
$usersFile = str_replace(['/', '\\'], DIRECTORY_SEPARATOR, $usersFile);

$usersDir = dirname($usersFile);
if(!is_dir($usersDir)){
    @mkdir($usersDir, 0775, true);
}

$store = load_users_store($usersFile);

if($command === 'generate'){
    $code = normalize_beta_key_code((string)($opts['code'] ?? ''));
    if($code === ''){
        $code = generate_beta_key_code((string)($opts['prefix'] ?? 'NOCWALL-BETA'));
    }
    if($code === ''){
        cli_error('Unable to generate a valid beta key code.');
        exit(1);
    }
    if(isset($store['beta_keys'][$code])){
        cli_error('Beta key already exists: ' . $code);
        exit(1);
    }
    $maxRedemptions = (int)($opts['max-redemptions'] ?? 1);
    if($maxRedemptions < 1) $maxRedemptions = 1;
    $expiresAt = trim((string)($opts['expires-at'] ?? ''));
    if($expiresAt === '' && isset($opts['ttl-days'])){
        $ttlDays = (int)$opts['ttl-days'];
        if($ttlDays > 0){
            $expiresAt = date('c', time() + ($ttlDays * 86400));
        }
    }
    if($expiresAt !== '' && beta_key_expiration_ts($expiresAt) <= 0){
        cli_error('Invalid --expires-at value. Use ISO8601 (example: 2026-06-01T00:00:00Z).');
        exit(1);
    }
    $issuedBy = trim((string)($opts['issued-by'] ?? 'ops'));
    $notes = trim((string)($opts['notes'] ?? ''));
    $now = date('c');
    $key = default_beta_key_state($code);
    $key['max_redemptions'] = $maxRedemptions;
    $key['expires_at'] = $expiresAt;
    $key['issued_by'] = $issuedBy;
    $key['notes'] = $notes;
    $key['created_at'] = $now;
    $key['updated_at'] = $now;
    if($expiresAt !== '' && beta_key_is_expired($key)){
        $key['status'] = 'expired';
    }
    $store['beta_keys'][$code] = normalize_beta_key_state($key, $code);
    append_beta_key_audit_event($store, 'beta_key_issue', $code, '', $issuedBy, [
        'max_redemptions' => $maxRedemptions,
        'expires_at' => $expiresAt
    ]);
    if(!save_users_store($usersFile, $store)){
        cli_error('Failed to write users store: ' . $usersFile);
        exit(1);
    }

    $out = [
        'ok' => 1,
        'command' => 'generate',
        'users_file' => $usersFile,
        'key' => $store['beta_keys'][$code]
    ];
    if($jsonMode){
        cli_write(json_encode($out, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));
    } else {
        cli_write('Generated beta key: ' . $code);
        cli_write('Status: ' . $store['beta_keys'][$code]['status']);
        cli_write('Max redemptions: ' . $store['beta_keys'][$code]['max_redemptions']);
        cli_write('Expires at: ' . ($store['beta_keys'][$code]['expires_at'] !== '' ? $store['beta_keys'][$code]['expires_at'] : '(none)'));
        cli_write('Users store: ' . $usersFile);
    }
    exit(0);
}

if($command === 'list'){
    $keys = normalize_beta_keys_store($store['beta_keys'] ?? []);
    if($jsonMode){
        cli_write(json_encode([
            'ok' => 1,
            'command' => 'list',
            'users_file' => $usersFile,
            'count' => count($keys),
            'keys' => array_values($keys)
        ], JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));
        exit(0);
    }
    cli_write('Users store: ' . $usersFile);
    cli_write('Beta keys: ' . count($keys));
    foreach($keys as $row){
        $line = sprintf(
            '- %s | %s | %d/%d | expires: %s',
            (string)$row['code'],
            (string)$row['status'],
            (int)$row['redeemed_count'],
            (int)$row['max_redemptions'],
            (string)($row['expires_at'] !== '' ? $row['expires_at'] : 'none')
        );
        cli_write($line);
    }
    exit(0);
}

if(in_array($command, ['disable','revoke','enable'], true)){
    $code = normalize_beta_key_code((string)($opts['code'] ?? ''));
    if($code === ''){
        cli_error('Missing or invalid --code.');
        exit(1);
    }
    $targetStatus = ($command === 'enable') ? 'active' : 'disabled';
    $actor = trim((string)($opts['actor'] ?? 'cli'));
    $reason = trim((string)($opts['reason'] ?? ''));
    if(!set_beta_key_status($store, $code, $targetStatus, $actor, $reason)){
        cli_error('Unable to update beta key status. Key may not exist.');
        exit(1);
    }
    if(!save_users_store($usersFile, $store)){
        cli_error('Failed to write users store: ' . $usersFile);
        exit(1);
    }
    $key = $store['beta_keys'][$code] ?? null;
    if(!is_array($key)){
        cli_error('Status updated but key lookup failed unexpectedly.');
        exit(1);
    }
    if($jsonMode){
        cli_write(json_encode([
            'ok' => 1,
            'command' => $command,
            'users_file' => $usersFile,
            'key' => $key
        ], JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));
    } else {
        cli_write('Updated beta key: ' . $code);
        cli_write('Status: ' . (string)$key['status']);
        cli_write('Users store: ' . $usersFile);
    }
    exit(0);
}

cli_error('Unknown command: ' . $command);
usage();
exit(1);

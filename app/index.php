<?php
require_once __DIR__ . '/../site/marketing.php';

$target = rtrim(nocwall_app_base_url(), '/') . '/';
$query = (string)($_SERVER['QUERY_STRING'] ?? '');
if ($query !== '') {
    $target .= '?' . $query;
}
nocwall_redirect($target, 302);


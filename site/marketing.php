<?php

function nocwall_current_host(): string {
    $host = (string)($_SERVER['HTTP_HOST'] ?? 'localhost');
    $host = strtolower(trim($host));
    $parts = explode(':', $host, 2);
    return $parts[0];
}

function nocwall_current_scheme(): string {
    if (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off') {
        return 'https';
    }
    $forwarded = strtolower(trim((string)($_SERVER['HTTP_X_FORWARDED_PROTO'] ?? '')));
    if ($forwarded === 'https') {
        return 'https';
    }
    return 'http';
}

function nocwall_app_domains(): array {
    $raw = trim((string)(getenv('NOCWALL_APP_DOMAINS') ?: 'app.nocwall.com,app.nocwall.org'));
    $items = array_filter(array_map('trim', explode(',', $raw)));
    $out = [];
    foreach ($items as $item) {
        $out[] = strtolower($item);
    }
    return array_values(array_unique($out));
}

function nocwall_is_app_host(): bool {
    $host = nocwall_current_host();
    if (in_array($host, nocwall_app_domains(), true)) {
        return true;
    }
    return strpos($host, 'app.') === 0;
}

function nocwall_marketing_base_url(): string {
    $configured = trim((string)(getenv('NOCWALL_MARKETING_BASE_URL') ?: ''));
    if ($configured !== '') {
        return rtrim($configured, '/');
    }
    $host = nocwall_current_host();
    $scheme = nocwall_current_scheme();
    if ($host === 'app.nocwall.org') {
        return 'https://nocwall.org';
    }
    if ($host === 'app.nocwall.com') {
        return 'https://nocwall.com';
    }
    return $scheme . '://' . $host;
}

function nocwall_app_base_url(): string {
    $configured = trim((string)(getenv('NOCWALL_APP_BASE_URL') ?: ''));
    if ($configured !== '') {
        return rtrim($configured, '/');
    }
    $host = nocwall_current_host();
    if (substr($host, -11) === 'nocwall.org') {
        return 'https://app.nocwall.org';
    }
    if (substr($host, -11) === 'nocwall.com') {
        return 'https://app.nocwall.com';
    }
    return nocwall_current_scheme() . '://' . $host;
}

function nocwall_redirect(string $url, int $status = 302): void {
    header('Location: ' . $url, true, $status);
    exit;
}

function nocwall_marketing_home_redirect_for_app(): void {
    if (nocwall_is_app_host()) {
        nocwall_redirect(nocwall_app_base_url() . '/');
    }
}

function nocwall_marketing_page_meta(string $route): array {
    $meta = [
        'home' => [
            'title' => 'NOCWALL | ISP & NOC Wallboard Monitoring',
            'description' => 'NOCWALL is a wall-mounted monitoring dashboard for ISPs and NOCs with fast online/offline visibility and alert-first operations.',
            'path' => '/',
        ],
        'pricing' => [
            'title' => 'Pricing | NOCWALL',
            'description' => 'Simple NOCWALL pricing by monitored device count for ISP and NOC teams.',
            'path' => '/pricing',
        ],
        'features' => [
            'title' => 'Features | NOCWALL',
            'description' => 'Explore NOCWALL features for device telemetry, wallboard display, alerting, and NMS integrations.',
            'path' => '/features',
        ],
        'docs' => [
            'title' => 'Docs | NOCWALL',
            'description' => 'NOCWALL getting started docs, connector setup guidance, and FAQ.',
            'path' => '/docs',
        ],
        'status' => [
            'title' => 'Status | NOCWALL',
            'description' => 'Current NOCWALL service status and incident communication page.',
            'path' => '/status',
        ],
        'contact' => [
            'title' => 'Contact | NOCWALL',
            'description' => 'Contact NOCWALL sales and support.',
            'path' => '/contact',
        ],
        'privacy' => [
            'title' => 'Privacy Policy | NOCWALL',
            'description' => 'NOCWALL privacy policy.',
            'path' => '/privacy',
        ],
        'terms' => [
            'title' => 'Terms of Service | NOCWALL',
            'description' => 'NOCWALL terms of service.',
            'path' => '/terms',
        ],
    ];
    return $meta[$route] ?? $meta['home'];
}

function nocwall_nav_items(): array {
    return [
        ['href' => '/', 'label' => 'Home', 'key' => 'home'],
        ['href' => '/features', 'label' => 'Features', 'key' => 'features'],
        ['href' => '/pricing', 'label' => 'Pricing', 'key' => 'pricing'],
        ['href' => '/docs', 'label' => 'Docs', 'key' => 'docs'],
        ['href' => '/status', 'label' => 'Status', 'key' => 'status'],
        ['href' => '/contact', 'label' => 'Contact', 'key' => 'contact'],
    ];
}

function nocwall_render_marketing_page(string $route, string $contentHtml): void {
    $meta = nocwall_marketing_page_meta($route);
    $appUrl = nocwall_app_base_url();
    $marketingBase = nocwall_marketing_base_url();
    $canonical = $marketingBase . $meta['path'];
    $ogImage = $marketingBase . '/site/assets/dashboard-placeholder.svg';
    $title = htmlspecialchars($meta['title'], ENT_QUOTES);
    $description = htmlspecialchars($meta['description'], ENT_QUOTES);

    $schema = [
        '@context' => 'https://schema.org',
        '@type' => 'SoftwareApplication',
        'name' => 'NOCWALL',
        'applicationCategory' => 'NetworkMonitoringApplication',
        'operatingSystem' => 'Web',
        'url' => $marketingBase,
        'offers' => [
            '@type' => 'AggregateOffer',
            'priceCurrency' => 'USD',
            'lowPrice' => '29',
            'highPrice' => '299',
            'offerCount' => '4',
        ],
        'provider' => [
            '@type' => 'Organization',
            'name' => 'NOCWALL',
            'url' => $marketingBase,
        ],
    ];
    ?>
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title><?=$title?></title>
  <meta name="description" content="<?=$description?>">
  <link rel="canonical" href="<?=htmlspecialchars($canonical, ENT_QUOTES)?>">
  <meta property="og:type" content="website">
  <meta property="og:title" content="<?=$title?>">
  <meta property="og:description" content="<?=$description?>">
  <meta property="og:url" content="<?=htmlspecialchars($canonical, ENT_QUOTES)?>">
  <meta property="og:image" content="<?=htmlspecialchars($ogImage, ENT_QUOTES)?>">
  <meta name="twitter:card" content="summary_large_image">
  <link rel="icon" href="/favicon.svg" type="image/svg+xml">
  <link rel="stylesheet" href="/site/assets/marketing.css?v=1">
  <script type="application/ld+json"><?=json_encode($schema, JSON_UNESCAPED_SLASHES)?></script>
</head>
<body>
  <header class="topbar">
    <a class="brand" href="/">NOCWALL</a>
    <nav class="nav">
      <?php foreach (nocwall_nav_items() as $item): ?>
        <?php $isActive = ($item['key'] === $route); ?>
        <a href="<?=$item['href']?>"<?=$isActive ? ' class="active"' : ''?>><?=$item['label']?></a>
      <?php endforeach; ?>
    </nav>
    <div class="cta">
      <a class="btn btn-ghost" href="<?=$appUrl?>/">Log In</a>
      <a class="btn btn-primary" href="<?=$appUrl?>/?login=1">Start Free Trial</a>
    </div>
  </header>

  <main>
    <?=$contentHtml?>
  </main>

  <footer class="footer">
    <div class="footer-inner">
      <div class="footer-brand">
        <div class="brand">NOCWALL</div>
        <p>Built for ISPs and NOCs. Wall-mounted visibility with alert-first operations.</p>
      </div>
      <div class="footer-links">
        <a href="/features">Features</a>
        <a href="/pricing">Pricing</a>
        <a href="/docs">Docs</a>
        <a href="/contact">Contact</a>
      </div>
      <div class="footer-links">
        <a href="/privacy">Privacy</a>
        <a href="/terms">Terms</a>
        <a href="/status">Status</a>
        <a href="<?=$appUrl?>/">Log In</a>
      </div>
    </div>
    <div class="footer-note">Copyright <?=date('Y')?> NOCWALL. Annual plans include 2 months free.</div>
  </footer>
</body>
</html>
<?php
}

function nocwall_store_contact_submission(array $input): bool {
    $name = trim((string)($input['name'] ?? ''));
    $email = trim((string)($input['email'] ?? ''));
    $company = trim((string)($input['company'] ?? ''));
    $message = trim((string)($input['message'] ?? ''));

    if ($name === '' || $email === '' || $message === '') {
        return false;
    }
    if (strlen($name) > 140 || strlen($email) > 200 || strlen($company) > 200 || strlen($message) > 4000) {
        return false;
    }
    if (!filter_var($email, FILTER_VALIDATE_EMAIL)) {
        return false;
    }

    $cacheDir = __DIR__ . '/../cache';
    if (!is_dir($cacheDir)) {
        @mkdir($cacheDir, 0775, true);
    }
    $file = $cacheDir . '/contact_submissions.json';
    $existing = [];
    if (is_file($file)) {
        $raw = @file_get_contents($file);
        $parsed = json_decode((string)$raw, true);
        if (is_array($parsed)) {
            $existing = $parsed;
        }
    }

    $existing[] = [
        'timestamp' => date('c'),
        'name' => $name,
        'email' => $email,
        'company' => $company,
        'message' => $message,
        'ip' => (string)($_SERVER['REMOTE_ADDR'] ?? ''),
        'user_agent' => (string)($_SERVER['HTTP_USER_AGENT'] ?? ''),
    ];

    $json = json_encode($existing, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
    if ($json === false) {
        return false;
    }
    return @file_put_contents($file, $json, LOCK_EX) !== false;
}


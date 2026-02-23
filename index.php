<?php
require_once __DIR__ . '/site/marketing.php';

if (nocwall_is_app_host()) {
    require __DIR__ . '/app.php';
    exit;
}

$path = parse_url((string)($_SERVER['REQUEST_URI'] ?? '/'), PHP_URL_PATH);
$path = is_string($path) ? $path : '/';
if ($path === '/app' || $path === '/app/' || strpos($path, '/app/') === 0) {
    $suffix = $path === '/app' ? '/' : substr($path, 4);
    if ($suffix === false || $suffix === '') {
        $suffix = '/';
    }
    $target = rtrim(nocwall_app_base_url(), '/') . $suffix;
    $query = (string)($_SERVER['QUERY_STRING'] ?? '');
    if ($query !== '') {
        $target .= '?' . $query;
    }
    nocwall_redirect($target, 302);
}

// Guard against old app-style query routes on marketing host.
if (isset($_GET['ajax']) || isset($_GET['action']) || isset($_GET['webhook']) || isset($_GET['login']) || isset($_GET['view'])) {
    $target = rtrim(nocwall_app_base_url(), '/') . '/';
    $query = (string)($_SERVER['QUERY_STRING'] ?? '');
    if ($query !== '') {
        $target .= '?' . $query;
    }
    nocwall_redirect($target, 302);
}

ob_start();
?>
<section class="hero">
  <div>
    <span class="eyebrow">Built for ISPs and NOCs</span>
    <h1>One Wallboard For Fast Network Awareness</h1>
    <p>NOCWALL gives operators a dense, glanceable view of network device state from vendor APIs and local agents. Keep your NOC screen useful, alert-first, and operationally clear.</p>
    <div class="hero-actions">
      <a class="btn btn-primary" href="<?=nocwall_app_base_url()?>/?login=1">Start Free Trial</a>
      <a class="btn btn-ghost" href="<?=nocwall_app_base_url()?>/">Log In</a>
      <a class="btn btn-ghost" href="/docs">View Docs</a>
    </div>
  </div>
  <div class="panel">
    <div class="stats">
      <div class="stat"><strong>API</strong><span class="muted">Multi-source ingestion</span></div>
      <div class="stat"><strong>Agent</strong><span class="muted">Linux/SBC telemetry</span></div>
      <div class="stat"><strong>Wall</strong><span class="muted">Alert-first dashboard</span></div>
    </div>
    <p class="muted" style="margin-top:12px">Current connector direction: UISP first, with broad NMS API support in-progress.</p>
  </div>
</section>

<section class="section">
  <h2>Why NOC Teams Choose NOCWALL</h2>
  <div class="grid-3">
    <article class="card">
      <h3>Single Card Per Device</h3>
      <p>Designed to fit meaningful device state into minimal visual space for at-a-glance decisions.</p>
    </article>
    <article class="card">
      <h3>Fast Operational Signal</h3>
      <p>Online/offline visibility and alert feedback tuned for wall-mounted NOC screens.</p>
    </article>
    <article class="card">
      <h3>Configurable Data Inputs</h3>
      <p>Use API keys from NMS platforms and local agents to collect broader telemetry.</p>
    </article>
  </div>
</section>

<section class="section panel">
  <h2>Dashboard Preview</h2>
  <p class="muted">Placeholder mockup area for production screenshot updates.</p>
  <img class="screenshot" src="/site/assets/dashboard-placeholder.svg" alt="NOCWALL dashboard placeholder">
</section>

<section class="section">
  <h2>Pricing</h2>
  <div class="grid-2">
    <article class="card"><h3>Starter (1-50)</h3><div class="price">$29/mo</div></article>
    <article class="card"><h3>Growth (51-99)</h3><div class="price">$79/mo</div></article>
    <article class="card"><h3>Pro (100-200)</h3><div class="price">$149/mo</div></article>
    <article class="card"><h3>Enterprise (Unlimited)</h3><div class="price">$299/mo</div></article>
  </div>
  <p class="pricing-note">Annual billing includes 2 months free.</p>
  <div class="hero-actions">
    <a class="btn btn-primary" href="/pricing">See Full Pricing</a>
    <a class="btn btn-ghost" href="/contact">Contact Sales</a>
  </div>
</section>

<section class="section grid-3">
  <article class="card testimonial">
    <h3>"Exactly what our wall needed"</h3>
    <p>Placeholder testimonial from a regional ISP operations lead.</p>
  </article>
  <article class="card testimonial">
    <h3>"High signal, low clutter"</h3>
    <p>Placeholder testimonial from a managed services network team.</p>
  </article>
  <article class="card testimonial">
    <h3>"Fast setup, clear status"</h3>
    <p>Placeholder testimonial from an enterprise NOC manager.</p>
  </article>
</section>
<?php
$content = ob_get_clean();
nocwall_render_marketing_page('home', $content);

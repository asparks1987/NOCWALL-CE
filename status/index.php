<?php
require_once __DIR__ . '/../site/marketing.php';
nocwall_marketing_home_redirect_for_app();

ob_start();
?>
<section class="section">
  <h1>NOCWALL Service Status</h1>
  <p class="muted">Status page placeholder. This route will be connected to live health and incident feeds in a future run.</p>
</section>

<section class="section grid-3">
  <article class="card">
    <h3>Dashboard</h3>
    <p><strong>Operational</strong><br><span class="muted">Last check: <?=htmlspecialchars(date('c'), ENT_QUOTES)?></span></p>
  </article>
  <article class="card">
    <h3>API Ingestion</h3>
    <p><strong>Operational</strong><br><span class="muted">Telemetry and polling endpoint status placeholder.</span></p>
  </article>
  <article class="card">
    <h3>Account Services</h3>
    <p><strong>Operational</strong><br><span class="muted">Auth and account management status placeholder.</span></p>
  </article>
</section>

<section class="section panel">
  <h3>Incident Communication</h3>
  <p class="muted">During active incidents, this page will publish timelines, affected components, and mitigation updates.</p>
</section>
<?php
$content = ob_get_clean();
nocwall_render_marketing_page('status', $content);


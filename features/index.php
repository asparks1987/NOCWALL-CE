<?php
require_once __DIR__ . '/../site/marketing.php';
nocwall_marketing_home_redirect_for_app();

ob_start();
?>
<section class="section">
  <h1>Built For High-Visibility NOC Operations</h1>
  <p class="muted">NOCWALL emphasizes dense, glanceable telemetry for operators managing large and distributed networks.</p>
</section>

<section class="section grid-3">
  <article class="card">
    <h3>Card-First Device Monitoring</h3>
    <p>Single-card snapshots for device state, role, and key health signals.</p>
  </article>
  <article class="card">
    <h3>NMS API Ingestion</h3>
    <p>Connect vendor systems through API keys to bring data into one wallboard.</p>
  </article>
  <article class="card">
    <h3>Agent Telemetry</h3>
    <p>Lightweight Linux/SBC agent path for on-network discovery and local data capture.</p>
  </article>
  <article class="card">
    <h3>Alert-First UI</h3>
    <p>Immediate visual state changes with configurable siren behavior at tab and card levels.</p>
  </article>
  <article class="card">
    <h3>Persistent User Settings</h3>
    <p>Per-account dashboard settings, source configuration, and ordering synced server-side.</p>
  </article>
  <article class="card">
    <h3>Open-Core Extension Model</h3>
    <p>CE wallboard core with extensible plugin boundaries for advanced workflows.</p>
  </article>
</section>

<section class="section panel">
  <h2>What Teams Use NOCWALL For</h2>
  <div class="grid-2">
    <div class="card">
      <h3>ISP NOCs</h3>
      <p>Monitor access layers, gateways, and edge devices from a single always-on display.</p>
    </div>
    <div class="card">
      <h3>Managed Network Providers</h3>
      <p>Operate multiple customer environments while keeping device state clear and actionable.</p>
    </div>
  </div>
</section>
<?php
$content = ob_get_clean();
nocwall_render_marketing_page('features', $content);


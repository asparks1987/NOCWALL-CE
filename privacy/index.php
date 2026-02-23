<?php
require_once __DIR__ . '/../site/marketing.php';
nocwall_marketing_home_redirect_for_app();

ob_start();
?>
<section class="section">
  <h1>Privacy Policy</h1>
  <p class="muted">Effective date: <?=htmlspecialchars(date('F j, Y'), ENT_QUOTES)?></p>
</section>

<section class="section panel">
  <h3>Summary</h3>
  <p>NOCWALL collects account information, operational telemetry, and configuration data required to provide monitoring services.</p>
  <h3>Data We Collect</h3>
  <ul>
    <li>Account identifiers and authentication metadata</li>
    <li>Device and network telemetry submitted via connector APIs and agents</li>
    <li>Operational logs used for reliability and support</li>
  </ul>
  <h3>Data Usage</h3>
  <ul>
    <li>Deliver the service and improve reliability</li>
    <li>Provide support and troubleshoot incidents</li>
    <li>Meet legal and security obligations</li>
  </ul>
  <h3>Contact</h3>
  <p>For privacy requests, use <a href="/contact">/contact</a>.</p>
</section>
<?php
$content = ob_get_clean();
nocwall_render_marketing_page('privacy', $content);


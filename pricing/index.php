<?php
require_once __DIR__ . '/../site/marketing.php';
nocwall_marketing_home_redirect_for_app();

ob_start();
?>
<section class="section">
  <h1>Simple Pricing For ISP and NOC Teams</h1>
  <p class="muted">Pick a plan by monitored device count. Annual billing includes <strong>2 months free</strong>.</p>
</section>

<section class="section grid-2">
  <article class="card">
    <h3>Starter (1-50)</h3>
    <div class="price">$29<span class="muted">/mo</span></div>
    <p>Wallboard visibility and basic health monitoring for smaller environments.</p>
  </article>
  <article class="card">
    <h3>Growth (51-99)</h3>
    <div class="price">$79<span class="muted">/mo</span></div>
    <p>Expanded monitoring for growing NOCs with more sites and devices.</p>
  </article>
  <article class="card">
    <h3>Pro (100-200)</h3>
    <div class="price">$149<span class="muted">/mo</span></div>
    <p>Advanced operations for larger teams and higher cardinality networks.</p>
  </article>
  <article class="card">
    <h3>Enterprise (Unlimited)</h3>
    <div class="price">$299<span class="muted">/mo</span></div>
    <p>Unlimited scale with enterprise onboarding and priority support options.</p>
  </article>
</section>

<section class="section panel">
  <h3>All Plans Include</h3>
  <ul>
    <li>Browser-based dashboard access</li>
    <li>NMS API connector support (UISP first, expanding connector catalog)</li>
    <li>Local agent onboarding model for on-network telemetry</li>
    <li>Email support and self-service docs</li>
  </ul>
  <p class="pricing-note">Annual billing: pay for 10 months, use all 12.</p>
  <div class="hero-actions">
    <a class="btn btn-primary" href="<?=nocwall_app_base_url()?>/?login=1">Start Free Trial</a>
    <a class="btn btn-ghost" href="/contact">Talk to Sales</a>
  </div>
</section>
<?php
$content = ob_get_clean();
nocwall_render_marketing_page('pricing', $content);


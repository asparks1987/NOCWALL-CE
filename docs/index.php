<?php
require_once __DIR__ . '/../site/marketing.php';
nocwall_marketing_home_redirect_for_app();

ob_start();
?>
<section class="section">
  <h1>Documentation</h1>
  <p class="muted">Quick start guide for account provisioning, connector setup, and dashboard access.</p>
</section>

<section class="section grid-2">
  <article class="panel">
    <h3>Getting Started</h3>
    <ol>
      <li>Create your account and sign in from the app login.</li>
      <li>Add one or more NMS API sources (UISP first, more connectors rolling in).</li>
      <li>Optionally deploy a local Linux/SBC agent on your network.</li>
      <li>Open your dashboard wall view and tune siren/preferences per tab and card.</li>
    </ol>
  </article>
  <article class="panel">
    <h3>FAQ</h3>
    <p><strong>Do I need to self-host?</strong><br>Primary direction is hosted access through nocwall.com.</p>
    <p><strong>Can I connect multiple NMS systems?</strong><br>Yes. UISP is first; connector coverage expands over time.</p>
    <p><strong>Can I still run CE locally?</strong><br>Yes, CE remains runnable for testing and development workflows.</p>
  </article>
</section>

<section class="section panel">
  <h3>Need More Detail?</h3>
  <p>For implementation notes and support, use <a href="/contact">/contact</a>.</p>
  <div class="hero-actions">
    <a class="btn btn-primary" href="<?=nocwall_app_base_url()?>/?login=1">Start Free Trial</a>
    <a class="btn btn-ghost" href="/features">View Features</a>
  </div>
</section>
<?php
$content = ob_get_clean();
nocwall_render_marketing_page('docs', $content);


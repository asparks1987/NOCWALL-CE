<?php
require_once __DIR__ . '/../site/marketing.php';
nocwall_marketing_home_redirect_for_app();

ob_start();
?>
<section class="section">
  <h1>Terms of Service</h1>
  <p class="muted">Effective date: <?=htmlspecialchars(date('F j, Y'), ENT_QUOTES)?></p>
</section>

<section class="section panel">
  <h3>Service Scope</h3>
  <p>NOCWALL provides network monitoring software and related hosted services under the selected plan.</p>
  <h3>Acceptable Use</h3>
  <ul>
    <li>No abusive, unlawful, or unauthorized use of the platform</li>
    <li>Customers remain responsible for credentials and API keys they provide</li>
    <li>Do not attempt to disrupt service operation or security controls</li>
  </ul>
  <h3>Billing</h3>
  <p>Paid plans are billed monthly or annually according to selected terms and provider checkout flow.</p>
  <h3>Support and Availability</h3>
  <p>Support channels and service levels vary by plan tier.</p>
</section>
<?php
$content = ob_get_clean();
nocwall_render_marketing_page('terms', $content);


<?php
require_once __DIR__ . '/../site/marketing.php';
nocwall_marketing_home_redirect_for_app();

$submitted = false;
$error = '';

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $submitted = nocwall_store_contact_submission($_POST);
    if (!$submitted) {
        $error = 'Please provide a valid name, email, and message.';
    } else {
        $_POST = [];
    }
}

ob_start();
?>
<section class="section">
  <h1>Contact NOCWALL</h1>
  <p class="muted">Questions about pricing, onboarding, integrations, or rollout plans? Send us a note.</p>
</section>

<section class="section grid-2">
  <article class="panel">
    <h3>Sales & Partnerships</h3>
    <p>Discuss plan sizing, enterprise onboarding, and deployment strategy for your NOC.</p>
    <h3>Technical Support</h3>
    <p>For product issues, include environment details and impacted components.</p>
  </article>
  <article class="panel">
    <?php if ($submitted): ?>
      <div class="alert">Message received. We will follow up shortly.</div>
    <?php endif; ?>
    <?php if ($error !== ''): ?>
      <div class="card" style="border-color:#6b2d2d;background:#261313;color:#ffcdcd;margin-bottom:10px;"><?=$error?></div>
    <?php endif; ?>
    <form method="post" action="/contact/">
      <label for="name">Name</label>
      <input id="name" name="name" required value="<?=htmlspecialchars((string)($_POST['name'] ?? ''), ENT_QUOTES)?>">
      <label for="email">Email</label>
      <input id="email" name="email" type="email" required value="<?=htmlspecialchars((string)($_POST['email'] ?? ''), ENT_QUOTES)?>">
      <label for="company">Company</label>
      <input id="company" name="company" value="<?=htmlspecialchars((string)($_POST['company'] ?? ''), ENT_QUOTES)?>">
      <label for="message">Message</label>
      <textarea id="message" name="message" required><?=htmlspecialchars((string)($_POST['message'] ?? ''), ENT_QUOTES)?></textarea>
      <button class="btn btn-primary" type="submit">Send Message</button>
    </form>
  </article>
</section>
<?php
$content = ob_get_clean();
nocwall_render_marketing_page('contact', $content);

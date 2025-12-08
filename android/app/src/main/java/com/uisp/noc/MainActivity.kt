package com.uisp.noc

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.content.ClipData
import android.content.ClipboardManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.PowerManager
import android.provider.Settings
import android.view.Menu
import android.view.MenuItem
import android.view.View
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.core.view.isVisible
import androidx.fragment.app.commit
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.snackbar.Snackbar
import com.uisp.noc.ui.DashboardFragment
import com.uisp.noc.ui.LoginFragment
import com.uisp.noc.ui.MainViewModel
import com.uisp.noc.ui.MainViewModel.DiagnosticMessage
import kotlinx.coroutines.launch
import android.widget.TextView
import android.widget.Button
import android.widget.ImageButton

class MainActivity : AppCompatActivity() {

    private lateinit var viewModel: MainViewModel

    private lateinit var toolbar: MaterialToolbar
    private lateinit var loadingOverlay: View
    private lateinit var errorBanner: View
    private lateinit var errorTitle: TextView
    private lateinit var errorDetail: TextView
    private lateinit var errorRequestId: TextView
    private lateinit var errorCopyButton: Button
    private lateinit var errorDismiss: ImageButton
    private var optionsMenu: Menu? = null

    private val requestPermissionLauncher =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { isGranted: Boolean ->
            if (!isGranted) {
                showMessage("Notifications are disabled. You will not receive gateway alerts.")
            }
        }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        toolbar = findViewById(R.id.toolbar)
        loadingOverlay = findViewById(R.id.loading_overlay)
        errorBanner = findViewById(R.id.error_banner)
        errorTitle = findViewById(R.id.error_title)
        errorDetail = findViewById(R.id.error_detail)
        errorRequestId = findViewById(R.id.error_request_id)
        errorCopyButton = findViewById(R.id.error_copy)
        errorDismiss = findViewById(R.id.error_dismiss)
        setSupportActionBar(toolbar)

        // Create the notification channel
        NotificationHelper.createNotificationChannel(this)

        // Synchronously create the ViewModel.
        val factory = Injector.provideViewModelFactory(applicationContext)
        viewModel = ViewModelProvider(this, factory).get(MainViewModel::class.java)
        collectFlows(viewModel)

        if (savedInstanceState == null) {
            supportFragmentManager.commit {
                replace(R.id.content_container, LoginFragment())
            }
        }

        errorDismiss.setOnClickListener { hideErrorBanner() }
        errorCopyButton.setOnClickListener { copyDiagnostics() }

        askNotificationPermission()
        askForBatteryOptimizations()
    }

    private fun askNotificationPermission() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) !=
                PackageManager.PERMISSION_GRANTED
            ) {
                requestPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
            }
        }
    }

    private fun askForBatteryOptimizations() {
        val packageName = packageName
        val pm = getSystemService(POWER_SERVICE) as PowerManager
        if (!pm.isIgnoringBatteryOptimizations(packageName)) {
            val intent = Intent()
            intent.action = Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS
            intent.data = Uri.parse("package:$packageName")
            startActivity(intent)
        }
    }

    private fun collectFlows(vm: MainViewModel) {
        lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                vm.sessionState.collect { renderSessionState(it) }
            }
        }
        lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                vm.mobileConfigState.collect { state ->
                    when (state) {
                        is MainViewModel.MobileConfigUiState.Failure -> showError(state.diagnostic)
                        is MainViewModel.MobileConfigUiState.Success -> {
                            hideErrorBanner()
                            state.config.environment?.let { env ->
                                showMessage(getString(R.string.config_loaded_env, env))
                            }
                        }
                        else -> {}
                    }
                }
            }
        }
        lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                vm.events.collect { event ->
                    when (event) {
                        is MainViewModel.UiEvent.Message -> showMessage(event.text)
                        is MainViewModel.UiEvent.Error -> showError(event.diagnostic)
                    }
                }
            }
        }
    }

    private fun renderSessionState(state: MainViewModel.SessionState) {
        updateMenu(state)
        when (state) {
            MainViewModel.SessionState.Loading -> {
                showLoading(true)
            }
            is MainViewModel.SessionState.Unauthenticated -> {
                showLoading(false)
                hideErrorBanner()
                showLogin()
                toolbar.subtitle = null
                // Stop the service when the user logs out
                stopService(Intent(this, GatewayStatusService::class.java))
            }
            is MainViewModel.SessionState.Authenticated -> {
                showLoading(false)
                hideErrorBanner()
                showDashboard()
                toolbar.subtitle = state.session.uispBaseUrl.toHostLabel()
                // Start the service when the user is authenticated
                val intent = Intent(this, GatewayStatusService::class.java)
                ContextCompat.startForegroundService(this, intent)
            }
        }
    }

    private fun showLogin() {
        val current = supportFragmentManager.findFragmentById(R.id.content_container)
        if (current !is LoginFragment) {
            supportFragmentManager.commit {
                replace(R.id.content_container, LoginFragment())
            }
        }
    }

    private fun showDashboard() {
        val current = supportFragmentManager.findFragmentById(R.id.content_container)
        if (current !is DashboardFragment) {
            supportFragmentManager.commit {
                replace(R.id.content_container, DashboardFragment())
            }
        }
    }

    private fun showLoading(show: Boolean) {
        loadingOverlay.isVisible = show
    }

    override fun onCreateOptionsMenu(menu: Menu): Boolean {
        menuInflater.inflate(R.menu.menu_main, menu)
        optionsMenu = menu
        updateMenu(viewModel.sessionState.value)
        return true
    }

    private fun updateMenu(state: MainViewModel.SessionState) {
        val logout = optionsMenu?.findItem(R.id.action_logout)
        logout?.isVisible = state is MainViewModel.SessionState.Authenticated
    }

    override fun onOptionsItemSelected(item: MenuItem): Boolean {
        return when (item.itemId) {
            R.id.action_logout -> {
                viewModel.logout()
                true
            }
            else -> super.onOptionsItemSelected(item)
        }
    }

    private fun showMessage(message: String) {
        val root = findViewById<View>(android.R.id.content)
        Snackbar.make(root, message, Snackbar.LENGTH_LONG).show()
    }

    private fun showError(diagnostic: DiagnosticMessage) {
        errorBanner.isVisible = true
        val title = "${diagnostic.code}: ${diagnostic.message}"
        errorTitle.text = title
        val detailText = diagnostic.detail?.takeIf { it.isNotBlank() } ?: "No extra detail"
        errorDetail.text = detailText
        errorRequestId.text = getString(R.string.error_request_id_label, diagnostic.requestId)
        // Save diagnostics in the tag for copy action
        errorBanner.tag = "code=${diagnostic.code}; message=${diagnostic.message}; detail=${diagnostic.detail ?: "-"}; requestId=${diagnostic.requestId}"
    }

    private fun hideErrorBanner() {
        errorBanner.isVisible = false
    }

    private fun copyDiagnostics() {
        val clipboard = getSystemService(CLIPBOARD_SERVICE) as ClipboardManager
        val text = errorBanner.tag as? String ?: "No diagnostics available"
        clipboard.setPrimaryClip(ClipData.newPlainText("Diagnostics", text))
        showMessage("Diagnostics copied to clipboard.")
    }

    private fun String.toHostLabel(): String {
        val parsed = runCatching { Uri.parse(this) }.getOrNull()
        return parsed?.host ?: this
    }
}

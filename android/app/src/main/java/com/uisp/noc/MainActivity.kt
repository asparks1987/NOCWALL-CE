package com.uisp.noc

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
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
import kotlinx.coroutines.launch

class MainActivity : AppCompatActivity() {

    private lateinit var viewModel: MainViewModel

    private lateinit var toolbar: MaterialToolbar
    private lateinit var loadingOverlay: View
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
                vm.events.collect { event ->
                    when (event) {
                        is MainViewModel.UiEvent.Message -> showMessage(event.text)
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
                showLogin()
                toolbar.subtitle = null
                // Stop the service when the user logs out
                stopService(Intent(this, GatewayStatusService::class.java))
            }
            is MainViewModel.SessionState.Authenticated -> {
                showLoading(false)
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

    private fun String.toHostLabel(): String {
        val parsed = runCatching { Uri.parse(this) }.getOrNull()
        return parsed?.host ?: this
    }
}

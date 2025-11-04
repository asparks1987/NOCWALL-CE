package com.uisp.noc

import android.net.Uri
import android.os.Bundle
import android.view.Menu
import android.view.MenuItem
import android.view.View
import androidx.activity.viewModels
import androidx.appcompat.app.AppCompatActivity
import androidx.core.view.isVisible
import androidx.fragment.app.commit
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.snackbar.Snackbar
import com.uisp.noc.data.SessionStore
import com.uisp.noc.data.UispRepository
import com.uisp.noc.ui.DashboardFragment
import com.uisp.noc.ui.LoginFragment
import com.uisp.noc.ui.MainViewModel
import kotlinx.coroutines.launch

class MainActivity : AppCompatActivity() {

    private val viewModel: MainViewModel by viewModels {
        MainViewModel.Factory(
            repository = UispRepository(),
            sessionStore = SessionStore(this)
        )
    }

    private lateinit var toolbar: MaterialToolbar
    private lateinit var loadingOverlay: View
    private var optionsMenu: Menu? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        toolbar = findViewById(R.id.toolbar)
        loadingOverlay = findViewById(R.id.loading_overlay)
        setSupportActionBar(toolbar)

        if (savedInstanceState == null) {
            supportFragmentManager.commit {
                replace(R.id.content_container, LoginFragment())
            }
        }

        collectFlows()
    }

    private fun collectFlows() {
        lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                viewModel.sessionState.collect { renderSessionState(it) }
            }
        }
        lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                viewModel.events.collect { event ->
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
            }
            is MainViewModel.SessionState.Authenticated -> {
                showLoading(false)
                showDashboard()
                toolbar.subtitle = state.session.uispBaseUrl.toHostLabel()
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
        val root = findViewById<View>(R.id.content_container)
        Snackbar.make(root, message, Snackbar.LENGTH_LONG).show()
    }

    private fun String.toHostLabel(): String {
        val parsed = runCatching { Uri.parse(this) }.getOrNull()
        return parsed?.host ?: this
    }
}

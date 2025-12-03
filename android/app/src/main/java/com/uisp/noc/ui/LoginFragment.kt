package com.uisp.noc.ui

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.core.widget.doAfterTextChanged
import androidx.fragment.app.Fragment
import androidx.fragment.app.activityViewModels
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import com.google.android.material.button.MaterialButton
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout
import com.uisp.noc.R
import kotlinx.coroutines.launch

class LoginFragment : Fragment() {

    private val viewModel: MainViewModel by activityViewModels()

    private lateinit var backendLayout: TextInputLayout
    private lateinit var backendInput: TextInputEditText
    private lateinit var apiTokenInput: TextInputEditText
    private lateinit var displayNameInput: TextInputEditText
    private lateinit var loginButton: MaterialButton
    private lateinit var progress: View

    private var isPrefilled = false

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        return inflater.inflate(R.layout.fragment_login, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        backendLayout = view.findViewById(R.id.input_backend_layout)
        backendInput = view.findViewById(R.id.input_backend)
        apiTokenInput = view.findViewById(R.id.input_username)
        displayNameInput = view.findViewById(R.id.input_password)
        loginButton = view.findViewById(R.id.button_login)
        progress = view.findViewById(R.id.login_progress)

        loginButton.setOnClickListener {
            backendLayout.error = null
            val backend = backendInput.text?.toString().orEmpty()
            val apiToken = apiTokenInput.text?.toString().orEmpty()
            val displayName = displayNameInput.text?.toString().orEmpty()
            viewModel.attemptLogin(backend, apiToken, displayName)
        }

        backendInput.doAfterTextChanged {
            backendLayout.error = null
        }

        collectState()
    }

    private fun collectState() {
        viewLifecycleOwner.lifecycleScope.launch {
            viewLifecycleOwner.repeatOnLifecycle(Lifecycle.State.STARTED) {
                viewModel.sessionState.collect { state ->
                    when (state) {
                        is MainViewModel.SessionState.Loading -> showLoading(true)
                        is MainViewModel.SessionState.Unauthenticated -> {
                            showLoading(false)
                            prefillInputs(state)
                        }
                        is MainViewModel.SessionState.Authenticated -> {
                            showLoading(false)
                        }
                    }
                }
            }
        }
    }

    private fun prefillInputs(state: MainViewModel.SessionState.Unauthenticated) {
        if (!isPrefilled) {
            state.lastBackendUrl?.let { backendInput.setText(it) }
            state.lastUsername?.let { displayNameInput.setText(it) }
            isPrefilled = true
        }
    }

    private fun showLoading(isLoading: Boolean) {
        progress.visibility = if (isLoading) View.VISIBLE else View.GONE
        loginButton.isEnabled = !isLoading
        backendInput.isEnabled = !isLoading
        apiTokenInput.isEnabled = !isLoading
        displayNameInput.isEnabled = !isLoading
    }
}

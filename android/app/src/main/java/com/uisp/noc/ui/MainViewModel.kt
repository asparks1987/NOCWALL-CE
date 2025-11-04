package com.uisp.noc.ui

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.uisp.noc.data.Session
import com.uisp.noc.data.SessionStore
import com.uisp.noc.data.UispRepository
import com.uisp.noc.data.model.DevicesSummary
import kotlinx.coroutines.channels.BufferOverflow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import java.io.IOException

class MainViewModel(
    private val repository: UispRepository,
    private val sessionStore: SessionStore
) : ViewModel() {

    private val _sessionState =
        MutableStateFlow<SessionState>(SessionState.Loading)
    val sessionState: StateFlow<SessionState> = _sessionState.asStateFlow()

    private val _dashboardState = MutableStateFlow(DashboardState())
    val dashboardState: StateFlow<DashboardState> = _dashboardState.asStateFlow()

    private val _events = MutableSharedFlow<UiEvent>(
        replay = 0,
        extraBufferCapacity = 1,
        onBufferOverflow = BufferOverflow.DROP_OLDEST
    )
    val events = _events.asSharedFlow()

    private var activeSession: Session? = null

    init {
        viewModelScope.launch {
            loadExistingSession()
        }
    }

    fun attemptLogin(backendUrl: String, username: String, password: String) {
        if (backendUrl.isBlank() || username.isBlank() || password.isBlank()) {
            viewModelScope.launch {
                _events.emit(UiEvent.Message("Please fill out server, username, and password."))
            }
            return
        }

        viewModelScope.launch {
            _sessionState.value = SessionState.Loading
            _dashboardState.value = DashboardState(isLoading = true)
            try {
                val session = repository.authenticate(
                    backendUrl = backendUrl,
                    username = username,
                    password = password
                )
                activeSession = session
                sessionStore.save(session)
                _sessionState.value = SessionState.Authenticated(session)
                refreshSummary()
            } catch (authEx: UispRepository.AuthException) {
                _events.emit(UiEvent.Message(authEx.message ?: "Invalid credentials"))
                _sessionState.value = SessionState.Unauthenticated(
                    lastBackendUrl = sessionStore.lastBackendUrl(),
                    lastUsername = sessionStore.lastUsername()
                )
                _dashboardState.value = DashboardState()
            } catch (ex: IOException) {
                _events.emit(
                    UiEvent.Message(
                        ex.message ?: "Unable to connect to backend. Check your URL and network."
                    )
                )
                _sessionState.value = SessionState.Unauthenticated(
                    lastBackendUrl = sessionStore.lastBackendUrl(),
                    lastUsername = sessionStore.lastUsername()
                )
                _dashboardState.value = DashboardState()
            }
        }
    }

    fun refreshSummary() {
        val session = activeSession
        if (session == null) {
            _dashboardState.value = DashboardState()
            return
        }

        viewModelScope.launch {
            _dashboardState.value = _dashboardState.value.copy(
                isLoading = true,
                errorMessage = null
            )
            try {
                val summary = repository.fetchSummary(session)
                _dashboardState.value = DashboardState(
                    isLoading = false,
                    summary = summary,
                    errorMessage = null
                )
            } catch (authEx: UispRepository.AuthException) {
                sessionStore.clear()
                activeSession = null
                _events.emit(UiEvent.Message("Session expired. Please sign in again."))
                _sessionState.value = SessionState.Unauthenticated(
                    lastBackendUrl = sessionStore.lastBackendUrl(),
                    lastUsername = sessionStore.lastUsername()
                )
                _dashboardState.value = DashboardState(
                    isLoading = false,
                    summary = null,
                    errorMessage = authEx.message
                )
            } catch (ex: IOException) {
                val previous = _dashboardState.value.summary
                _dashboardState.value = DashboardState(
                    isLoading = false,
                    summary = previous,
                    errorMessage = ex.message ?: "Unable to refresh data."
                )
                _events.emit(UiEvent.Message("Unable to refresh data."))
            }
        }
    }

    fun logout() {
        viewModelScope.launch {
            sessionStore.clear()
            activeSession = null
            _sessionState.value = SessionState.Unauthenticated(
                lastBackendUrl = sessionStore.lastBackendUrl(),
                lastUsername = sessionStore.lastUsername()
            )
            _dashboardState.value = DashboardState()
        }
    }

    private suspend fun loadExistingSession() {
        val session = sessionStore.load()
        if (session != null) {
            activeSession = session
            _sessionState.value = SessionState.Authenticated(session)
            refreshSummary()
        } else {
            _sessionState.value = SessionState.Unauthenticated(
                lastBackendUrl = sessionStore.lastBackendUrl(),
                lastUsername = sessionStore.lastUsername()
            )
        }
    }

    data class DashboardState(
        val isLoading: Boolean = false,
        val summary: DevicesSummary? = null,
        val errorMessage: String? = null
    )

    sealed class SessionState {
        object Loading : SessionState()
        data class Unauthenticated(
            val lastBackendUrl: String?,
            val lastUsername: String?
        ) : SessionState()

        data class Authenticated(val session: Session) : SessionState()
    }

    sealed class UiEvent {
        data class Message(val text: String) : UiEvent()
    }

    class Factory(
        private val repository: UispRepository,
        private val sessionStore: SessionStore
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            if (modelClass.isAssignableFrom(MainViewModel::class.java)) {
                return MainViewModel(repository, sessionStore) as T
            }
            throw IllegalArgumentException("Unknown ViewModel class: $modelClass")
        }
    }
}

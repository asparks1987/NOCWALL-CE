package com.uisp.noc.ui

import android.app.Application
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
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import java.io.IOException
import java.util.UUID

class MainViewModel(
    private val application: Application,
    private val repository: UispRepository,
    private val sessionStore: SessionStore
) : ViewModel() {

    private val _sessionState =
        MutableStateFlow<SessionState>(SessionState.Loading)
    val sessionState: StateFlow<SessionState> = _sessionState

    private val _isHistoryAvailable = MutableStateFlow(false)
    val isHistoryAvailable: StateFlow<Boolean> = _isHistoryAvailable.asStateFlow()

    val dashboardState: StateFlow<DashboardState> = repository.summaryFlow
        .map { summary -> DashboardState(summary = summary) }
        .stateIn(viewModelScope, SharingStarted.Lazily, DashboardState())


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
            _sessionState.collect {
                checkHistoryAvailability()
            }
        }
    }

    fun attemptLogin(uispUrl: String, apiToken: String, displayName: String) {
        if (uispUrl.isBlank() || apiToken.isBlank()) {
            viewModelScope.launch {
                _events.emit(
                    UiEvent.Error(
                        diagnostic(
                            code = "auth_missing",
                            message = "Please provide your UISP URL and API token.",
                            detail = "Missing UISP URL or API token"
                        )
                    )
                )
            }
            return
        }

        viewModelScope.launch {
            _sessionState.value = SessionState.Loading
            try {
                val session = repository.authenticate(
                    uispUrl = uispUrl,
                    displayName = displayName,
                    apiToken = apiToken
                )
                activeSession = session
                sessionStore.save(session)
                _sessionState.value = SessionState.Authenticated(session)
                refreshSummary()
            } catch (authEx: UispRepository.AuthException) {
                _events.emit(
                    UiEvent.Error(
                        diagnostic(
                            code = "auth_invalid",
                            message = "Invalid UISP credentials",
                            detail = authEx.message
                        )
                    )
                )
                _sessionState.value = SessionState.Unauthenticated(
                    lastBackendUrl = sessionStore.lastBackendUrl(),
                    lastUsername = sessionStore.lastUsername()
                )
            } catch (ex: IOException) {
                _events.emit(
                    UiEvent.Error(
                        diagnostic(
                            code = "network_unreachable",
                            message = "Unable to connect to backend. Check your URL and network.",
                            detail = ex.message
                        )
                    )
                )
                _sessionState.value = SessionState.Unauthenticated(
                    lastBackendUrl = sessionStore.lastBackendUrl(),
                    lastUsername = sessionStore.lastUsername()
                )
            }
        }
    }

    fun refreshSummary() {
        val session = activeSession
        if (session == null) {
            return
        }

        viewModelScope.launch {
            try {
                repository.fetchSummary(session, application)
                checkHistoryAvailability()
            } catch (authEx: UispRepository.AuthException) {
                sessionStore.clear()
                activeSession = null
                _events.emit(
                    UiEvent.Error(
                        diagnostic(
                            code = "auth_expired",
                            message = "Session expired. Please sign in again.",
                            detail = authEx.message
                        )
                    )
                )
                _sessionState.value = SessionState.Unauthenticated(
                    lastBackendUrl = sessionStore.lastBackendUrl(),
                    lastUsername = sessionStore.lastUsername()
                )
            } catch (ex: IOException) {
                _events.emit(
                    UiEvent.Error(
                        diagnostic(
                            code = "refresh_failed",
                            message = "Unable to refresh data.",
                            detail = ex.message
                        )
                    )
                )
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
        }
    }

    fun simulateGatewayOutage() {
        viewModelScope.launch {
            repository.triggerSimulatedGatewayOutage(application)
            _events.emit(UiEvent.Message("Simulated outage triggered."))
        }
    }

    fun clearSimulatedGatewayOutage() {
        viewModelScope.launch {
            repository.clearSimulatedGatewayOutage()
            _events.emit(UiEvent.Message("Simulated outage cleared."))
        }
    }

    fun acknowledgeDevice(deviceId: String, durationMinutes: Int) {
        viewModelScope.launch {
            repository.acknowledgeDevice(deviceId, durationMinutes)
            val label = if (durationMinutes >= 60) {
                val hours = durationMinutes / 60
                "${hours}h"
            } else {
                "${durationMinutes}m"
            }
            _events.emit(UiEvent.Message("Acknowledged for $label."))
        }
    }

    fun clearAcknowledgement(deviceId: String) {
        viewModelScope.launch {
            repository.clearAcknowledgement(deviceId)
            _events.emit(UiEvent.Message("Acknowledgement cleared."))
        }
    }

    private fun checkHistoryAvailability() {
        viewModelScope.launch {
            _isHistoryAvailable.value = repository.isServerReachable(application, activeSession)
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
        data class Error(val diagnostic: DiagnosticMessage) : UiEvent()
    }

    class Factory(
        private val application: Application,
        private val repository: UispRepository,
        private val sessionStore: SessionStore
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            if (modelClass.isAssignableFrom(MainViewModel::class.java)) {
                return MainViewModel(application, repository, sessionStore) as T
            }
            throw IllegalArgumentException("Unknown ViewModel class: $modelClass")
        }
    }

    data class DiagnosticMessage(
        val code: String,
        val message: String,
        val detail: String?,
        val requestId: String,
        val timestampMillis: Long
    )

    private fun diagnostic(code: String, message: String, detail: String?): DiagnosticMessage =
        DiagnosticMessage(
            code = code,
            message = message,
            detail = detail,
            requestId = UUID.randomUUID().toString(),
            timestampMillis = System.currentTimeMillis()
        )
}

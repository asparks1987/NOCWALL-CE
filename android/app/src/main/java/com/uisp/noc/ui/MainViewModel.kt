package com.uisp.noc.ui

import android.app.Application
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.google.firebase.messaging.FirebaseMessaging
import com.uisp.noc.BuildConfig
import com.uisp.noc.data.NewApiRepository
import com.uisp.noc.data.Session
import com.uisp.noc.data.SessionStore
import com.uisp.noc.data.UispRepository
import com.uisp.noc.data.model.DeviceStatus
import com.uisp.noc.data.model.DevicesSummary
import com.uisp.noc.network.ApiClient
import com.uisp.noc.network.ApiResult
import com.uisp.noc.network.DeviceSummaryDto
import com.uisp.noc.network.DiagnosticError
import com.uisp.noc.network.IncidentDto
import com.uisp.noc.network.MobileConfigDto
import kotlinx.coroutines.channels.BufferOverflow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import java.io.IOException
import java.util.Locale
import java.util.UUID

class MainViewModel(
    private val application: Application,
    private val repository: UispRepository,
    private val sessionStore: SessionStore
) : ViewModel() {

    private val _sessionState = MutableStateFlow<SessionState>(SessionState.Loading)
    val sessionState: StateFlow<SessionState> = _sessionState

    private val _isHistoryAvailable = MutableStateFlow(false)
    val isHistoryAvailable: StateFlow<Boolean> = _isHistoryAvailable.asStateFlow()

    private val apiSummaryState = MutableStateFlow<DevicesSummary?>(null)
    private val apiIncidentsState = MutableStateFlow<List<IncidentDto>>(emptyList())

    val dashboardState: StateFlow<DashboardState> = combine(
        repository.summaryFlow,
        apiSummaryState,
        apiIncidentsState
    ) { legacySummary, apiSummary, incidents ->
        val chosen = apiSummary ?: legacySummary
        val source = if (apiSummary != null) DataSource.API else DataSource.LEGACY
        val incidentsCount = if (apiSummary != null) incidents.size else null
        DashboardState(summary = chosen, source = source, incidentsCount = incidentsCount)
    }.stateIn(viewModelScope, SharingStarted.Lazily, DashboardState())

    private val _mobileConfigState =
        MutableStateFlow<MobileConfigUiState>(MobileConfigUiState.Idle)
    val mobileConfigState: StateFlow<MobileConfigUiState> = _mobileConfigState.asStateFlow()

    private val _events = MutableSharedFlow<UiEvent>(
        replay = 0,
        extraBufferCapacity = 1,
        onBufferOverflow = BufferOverflow.DROP_OLDEST
    )
    val events = _events.asSharedFlow()

    private var activeSession: Session? = null
    private var newApiRepository: NewApiRepository? = null

    init {
        viewModelScope.launch {
            loadExistingSession()
            _sessionState.collect { checkHistoryAvailability() }
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
                buildNewApiRepository(session)
                loadMobileConfig()
                refreshFromNewApi()
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
        val session = activeSession ?: return

        viewModelScope.launch {
            try {
                repository.fetchSummary(session, application)
                checkHistoryAvailability()
            } catch (authEx: UispRepository.AuthException) {
                sessionStore.clear()
                activeSession = null
                newApiRepository = null
                apiSummaryState.value = null
                apiIncidentsState.value = emptyList()
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

    fun refreshFromNewApi() {
        val repo = newApiRepository ?: return
        viewModelScope.launch {
            val devicesResult = repo.fetchDevices()
            val incidentsResult = repo.fetchIncidents()

            if (devicesResult is ApiResult.Success) {
                apiSummaryState.value = mapApiSummary(devicesResult.data)
            } else if (devicesResult is ApiResult.Error) {
                val diag = diagnosticFromApi(devicesResult.diagnostic)
                _events.emit(UiEvent.Error(diag))
            }

            if (incidentsResult is ApiResult.Success) {
                apiIncidentsState.value = incidentsResult.data
            } else if (incidentsResult is ApiResult.Error) {
                val diag = diagnosticFromApi(incidentsResult.diagnostic)
                _events.emit(UiEvent.Error(diag))
            }
        }
    }

    fun logout() {
        viewModelScope.launch {
            sessionStore.clear()
            activeSession = null
            newApiRepository = null
            apiSummaryState.value = null
            apiIncidentsState.value = emptyList()
            _mobileConfigState.value = MobileConfigUiState.Idle
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
            buildNewApiRepository(session)
            loadMobileConfig()
            refreshFromNewApi()
            refreshSummary()
        } else {
            _sessionState.value = SessionState.Unauthenticated(
                lastBackendUrl = sessionStore.lastBackendUrl(),
                lastUsername = sessionStore.lastUsername()
            )
        }
    }

    private fun mapApiSummary(dto: DeviceSummaryDto): DevicesSummary {
        val allDevices = dto.devices.map { device ->
            val role = device.role.lowercase(Locale.getDefault())
            val isGateway = role == "gateway"
            val isAp = role == "ap"
            val isBackbone = isGateway || isAp || role == "switch" || role == "router" || role == "ptp"
            DeviceStatus(
                id = device.id,
                name = device.name,
                role = role,
                isGateway = isGateway,
                isAp = isAp,
                isBackbone = isBackbone,
                online = device.online,
                latencyMs = device.latencyMs,
                status = if (device.online) "online" else "offline",
                ackUntilEpochMillis = device.ackUntil
            )
        }
        val gateways = allDevices.filter { it.isGateway }
        val aps = allDevices.filter { it.isAp }
        val switches = allDevices.filter { it.role == "switch" }
        val routers = allDevices.filter { it.role == "router" || it.role == "ptp" }
        val offlineGateways = gateways.filterNot { it.online }
        val offlineAps = aps.filterNot { it.online }
        val offlineBackbone = allDevices.filter { it.isBackbone && !it.online }
        val highLatencyCore = allDevices.filter { (it.isGateway || it.isAp) && (it.latencyMs ?: 0.0) > 200 }

        return DevicesSummary(
            allDevices = allDevices,
            lastUpdatedEpochMillis = System.currentTimeMillis(),
            gateways = gateways,
            aps = aps,
            switches = switches,
            routers = routers,
            offlineGateways = offlineGateways,
            offlineAps = offlineAps,
            offlineBackbone = offlineBackbone,
            highLatencyCore = highLatencyCore
        )
    }

    data class DashboardState(
        val isLoading: Boolean = false,
        val summary: DevicesSummary? = null,
        val errorMessage: String? = null,
        val source: DataSource = DataSource.LEGACY,
        val incidentsCount: Int? = null
    )

    enum class DataSource { LEGACY, API }

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

    sealed class MobileConfigUiState {
        object Idle : MobileConfigUiState()
        object Loading : MobileConfigUiState()
        data class Success(val config: MobileConfigDto) : MobileConfigUiState()
        data class Failure(val diagnostic: DiagnosticMessage) : MobileConfigUiState()
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

    private fun diagnosticFromApi(error: DiagnosticError): DiagnosticMessage =
        DiagnosticMessage(
            code = error.code,
            message = error.message,
            detail = error.detail,
            requestId = error.requestId,
            timestampMillis = error.timestampMillis
        )

    private fun buildNewApiRepository(session: Session, apiBaseOverride: String? = null) {
        val apiClient = ApiClient(
            baseUrl = apiBaseOverride?.takeIf { it.isNotBlank() } ?: session.uispBaseUrl,
            tokenProvider = { session.uispToken }
        )
        newApiRepository = NewApiRepository(apiClient)
    }

    private fun loadMobileConfig() {
        val repo = newApiRepository ?: return
        _mobileConfigState.value = MobileConfigUiState.Loading
        viewModelScope.launch {
            when (val result = repo.fetchMobileConfig()) {
                is ApiResult.Success -> {
                    _mobileConfigState.value = MobileConfigUiState.Success(result.data)
                    val apiBase = result.data.apiBaseUrl
                    if (!apiBase.isNullOrBlank() && activeSession != null) {
                        buildNewApiRepository(activeSession!!, apiBase)
                    }
                    _events.emit(
                        UiEvent.Message(
                            "Config loaded" + (result.data.environment?.let { " ($it)" } ?: "")
                        )
                    )
                    ensurePushRegistration()
                }
                is ApiResult.Error -> {
                    val diag = diagnosticFromApi(result.diagnostic)
                    _mobileConfigState.value = MobileConfigUiState.Failure(diag)
                    _events.emit(UiEvent.Error(diag))
                }
            }
        }
    }

    private fun ensurePushRegistration() {
        val repo = newApiRepository ?: return
        FirebaseMessaging.getInstance().token.addOnSuccessListener { token ->
            sessionStore.savePushToken(token)
            viewModelScope.launch {
                val result = repo.registerPush(
                    token = token,
                    platform = "android",
                    appVersion = BuildConfig.VERSION_NAME,
                    locale = Locale.getDefault()
                )
                when (result) {
                    is ApiResult.Success -> _events.emit(UiEvent.Message("Push registered"))
                    is ApiResult.Error -> _events.emit(UiEvent.Error(diagnosticFromApi(result.diagnostic)))
                }
            }
        }.addOnFailureListener { ex ->
            viewModelScope.launch {
                _events.emit(
                    UiEvent.Error(
                        diagnostic(
                            code = "push_token_failed",
                            message = "Unable to get push token",
                            detail = ex.message
                        )
                    )
                )
            }
        }
    }
}

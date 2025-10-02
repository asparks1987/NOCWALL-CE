package com.uisp.noc.wear

import android.app.Application
import android.net.Uri
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.google.android.gms.wearable.DataClient
import com.google.android.gms.wearable.DataEvent
import com.google.android.gms.wearable.DataEventBuffer
import com.google.android.gms.wearable.DataItem
import com.google.android.gms.wearable.DataMapItem
import com.google.android.gms.wearable.PutDataRequest
import com.google.android.gms.wearable.Wearable
import com.uisp.noc.wear.data.DevicesSummary
import com.uisp.noc.wear.data.WearConfig
import com.uisp.noc.wear.data.WearRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await

sealed interface WearUiState {
    data object Loading : WearUiState
    data class MissingConfig(val message: String) : WearUiState
    data class Error(val message: String, val lastSummary: DevicesSummary?) : WearUiState
    data class Success(val summary: DevicesSummary) : WearUiState
}

class MainViewModel(application: Application) : AndroidViewModel(application) {
    private val repository = WearRepository()
    private val dataClient = Wearable.getDataClient(application)
    private val messageClient = Wearable.getMessageClient(application)
    private val nodeClient = Wearable.getNodeClient(application)

    private val _uiState = MutableStateFlow<WearUiState>(WearUiState.Loading)
    val uiState: StateFlow<WearUiState> = _uiState

    private var lastSummary: DevicesSummary? = null
    private var currentConfig: WearConfig? = null

    private val dataListener = DataClient.OnDataChangedListener { buffer ->
        handleDataEvents(buffer)
    }

    init {
        dataClient.addListener(dataListener)
        loadInitialConfig()
    }

    override fun onCleared() {
        dataClient.removeListener(dataListener)
        super.onCleared()
    }

    fun refresh(force: Boolean = false) {
        val cfg = currentConfig
        if (cfg == null || !cfg.isValid()) {
            _uiState.value = WearUiState.MissingConfig("Awaiting configuration from phone")
            if (force) requestConfigFromPhone()
            return
        }
        refreshInternal(cfg, force)
    }

    fun requestConfigFromPhone() {
        viewModelScope.launch {
            try {
                val nodes = nodeClient.connectedNodes.await()
                if (nodes.isEmpty()) {
                    _uiState.value = WearUiState.MissingConfig("No paired phone detected")
                    return@launch
                }
                for (node in nodes) {
                    messageClient.sendMessage(node.id, PATH_REQUEST_CONFIG, ByteArray(0)).await()
                }
                if (currentConfig?.isValid() != true) {
                    _uiState.value = WearUiState.MissingConfig("Requested settings from phone…")
                }
            } catch (ex: Exception) {
                _uiState.value = WearUiState.MissingConfig("Unable to reach phone companion")
            }
        }
    }

    private fun loadInitialConfig() {
        viewModelScope.launch {
            try {
                val uri = Uri.Builder()
                    .scheme(PutDataRequest.WEAR_URI_SCHEME)
                    .path(PATH_CONFIG)
                    .build()
                val buffer = dataClient.getDataItems(uri).await()
                try {
                    if (buffer.count > 0) {
                        handleConfigUpdate(parseConfig(buffer[0]), force = true)
                    } else {
                        _uiState.value = WearUiState.MissingConfig("Open UISP NOC on your phone to sync settings")
                        requestConfigFromPhone()
                    }
                } finally {
                    buffer.release()
                }
            } catch (ex: Exception) {
                _uiState.value = WearUiState.MissingConfig("Waiting for configuration from phone")
                requestConfigFromPhone()
            }
        }
    }

    private fun handleDataEvents(events: DataEventBuffer) {
        try {
            for (event in events) {
                val path = event.dataItem.uri.path
                if (path == PATH_CONFIG) {
                    when (event.type) {
                        DataEvent.TYPE_CHANGED -> handleConfigUpdate(parseConfig(event.dataItem), force = true)
                        DataEvent.TYPE_DELETED -> {
                            currentConfig = null
                            _uiState.value = WearUiState.MissingConfig("Configuration removed. Open phone app.")
                        }
                    }
                }
            }
        } finally {
            events.release()
        }
    }

    private fun handleConfigUpdate(newConfig: WearConfig, force: Boolean) {
        currentConfig = newConfig
        if (!newConfig.isValid()) {
            _uiState.value = WearUiState.MissingConfig("Configure UISP URL and token on phone")
            return
        }
        refreshInternal(newConfig, force)
    }

    private fun parseConfig(dataItem: DataItem): WearConfig {
        val dataMap = DataMapItem.fromDataItem(dataItem).dataMap
        return WearConfig(
            baseUrl = dataMap.getString(KEY_BASE_URL) ?: "",
            apiToken = dataMap.getString(KEY_API_TOKEN) ?: ""
        )
    }

    private fun refreshInternal(cfg: WearConfig, force: Boolean) {
        if (!force && _uiState.value is WearUiState.Loading) return
        viewModelScope.launch {
            _uiState.value = WearUiState.Loading
            try {
                val summary = repository.fetchSummary(cfg)
                lastSummary = summary
                _uiState.value = WearUiState.Success(summary)
            } catch (ex: Exception) {
                _uiState.value = WearUiState.Error(
                    message = ex.message ?: "Unknown error",
                    lastSummary = lastSummary
                )
            }
        }
    }

    companion object {
        private const val PATH_CONFIG = "/uisp/config"
        private const val PATH_REQUEST_CONFIG = "/uisp/request-config"
        private const val KEY_BASE_URL = "base_url"
        private const val KEY_API_TOKEN = "api_token"
    }
}

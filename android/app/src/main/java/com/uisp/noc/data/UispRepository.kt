package com.uisp.noc.data

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import com.uisp.noc.MyFirebaseMessagingService
import com.uisp.noc.data.model.DeviceStatus
import com.uisp.noc.data.model.DevicesSummary
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.withContext
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.logging.HttpLoggingInterceptor
import org.json.JSONArray
import org.json.JSONException
import org.json.JSONObject
import java.io.IOException
import java.util.concurrent.TimeUnit

/**
 * Manages data operations for the UISP NOC application, handling authentication and
 * data fetching from a UISP backend using OkHttp.
 *
 * @property httpClient The OkHttpClient used for all network requests.
 */
open class UispRepository(
    private val httpClient: OkHttpClient = defaultClient(),
    private val acknowledgementStore: AcknowledgementStore = AcknowledgementStore.InMemory()
) {

    protected val _summaryFlow = MutableStateFlow<DevicesSummary?>(null)
    val summaryFlow: StateFlow<DevicesSummary?> = _summaryFlow.asStateFlow()

    private val deviceStatusHistory = mutableMapOf<String, MutableList<Long>>()
    private val FLAPPING_WINDOW_MS = 5 * 60 * 1000 // 5 minutes
    private val FLAPPING_THRESHOLD = 3 // 3 or more changes in the window

    //region Public API

    /**
     * Authenticates with the UISP backend using the provided credentials.
     */
    open suspend fun authenticate(
        uispUrl: String,
        displayName: String,
        apiToken: String
    ): Session = withContext(Dispatchers.IO) {
        val normalizedBackend = normalizeBackendUrl(uispUrl)
        val sanitizedToken = apiToken.trim()

        val statusRequest = Request.Builder()
            .url("$normalizedBackend/nms/heartbeat")
            .get()
            .header("Accept", "application/json")
            .header("x-auth-token", sanitizedToken)
            .header("User-Agent", USER_AGENT)
            .build()

        httpClient.newCall(statusRequest).execute().use { response ->
            if (response.code == 401) {
                throw AuthException("Invalid UISP token or insufficient permissions.")
            }
            if (!response.isSuccessful) {
                val message = response.body?.string().orEmpty()
                throw IOException("Unable to reach UISP server (HTTP ${response.code}). $message")
            }
        }

        val technicianName = displayName.ifBlank { "UISP technician" }
        return@withContext Session(
            backendUrl = normalizedBackend,
            username = technicianName,
            uispBaseUrl = normalizedBackend,
            uispToken = sanitizedToken
        )
    }

    /**
     * Fetches a summary of all network devices from the UISP API.
     */
    open suspend fun fetchSummary(session: Session, context: Context) {
        withContext(Dispatchers.IO) {
            val request = Request.Builder()
                .url(session.uispBaseUrl.trimEnd('/') + "/nms/api/v2.1/devices")
                .get()
                .header("Accept", "application/json")
                .header("x-auth-token", session.uispToken.trim())
                .header("User-Agent", USER_AGENT)
                .build()

            httpClient.newCall(request).execute().use { response ->
                val body = response.body?.string()
                    ?: throw IOException("Empty response from UISP devices endpoint")

                if (response.code == 401) {
                    throw AuthException("UISP token has expired or is invalid")
                }

                if (!response.isSuccessful) {
                    throw IOException("Failed to download UISP devices: HTTP ${response.code}")
                }

                val contentType = response.header("Content-Type")
                if (contentType == null || !contentType.startsWith("application/json")) {
                    throw AuthException("Login required. The UISP server returned an HTML page instead of JSON. Please check your session.")
                }

                val newSummary = applyAcknowledgements(parseDevices(body))
                val oldSummary = _summaryFlow.value
                _summaryFlow.value = newSummary

                if (oldSummary != null) {
                    detectAndNotifyStatusChanges(oldSummary, newSummary, context)
                }
            }
        }
    }

    private fun detectAndNotifyStatusChanges(
        oldSummary: DevicesSummary,
        newSummary: DevicesSummary,
        context: Context
    ) {
        val oldDevices = oldSummary.allDevices.associateBy { it.id }
        val newDevices = newSummary.allDevices.associateBy { it.id }

        for (newDevice in newDevices.values) {
            val oldDevice = oldDevices[newDevice.id]
            if (oldDevice != null && oldDevice.online != newDevice.online) {
                val isAcked = acknowledgementStore.ackUntil(newDevice.id)
                    ?.let { it > System.currentTimeMillis() } == true
                if (isAcked) continue

                if (newDevice.isBackbone) {
                    val sound = if (newDevice.isGateway) "buz" else "brrt"
                    val status = if (newDevice.online) "Online" else "Offline"
                    MyFirebaseMessagingService.sendNotification(
                        context,
                        "Device $status",
                        "${newDevice.name} (${newDevice.role}) is now $status.",
                        sound
                    )
                }

                if (newDevice.isBackbone) {
                    val deviceId = newDevice.id
                    val history = deviceStatusHistory.getOrPut(deviceId) { mutableListOf() }
                    val now = System.currentTimeMillis()
                    history.add(now)
                    history.removeAll { it < (now - FLAPPING_WINDOW_MS) }

                    if (history.size >= FLAPPING_THRESHOLD) {
                        val flappingSound = if (newDevice.isGateway) "buz" else "flrp"
                        MyFirebaseMessagingService.sendNotification(
                            context,
                            "Device Flapping",
                            "${newDevice.name} is flapping (changing status frequently).",
                            flappingSound
                        )
                        history.clear()
                    }
                }
            }
        }
    }

    open fun acknowledgeDevice(deviceId: String, durationMinutes: Int) {
        acknowledgementStore.acknowledge(deviceId, durationMinutes)
        _summaryFlow.value = _summaryFlow.value?.let { applyAcknowledgements(it) }
    }

    open fun clearAcknowledgement(deviceId: String) {
        acknowledgementStore.clear(deviceId)
        _summaryFlow.value = _summaryFlow.value?.let { applyAcknowledgements(it) }
    }

    private fun applyAcknowledgements(summary: DevicesSummary): DevicesSummary {
        val now = System.currentTimeMillis()
        val acknowledgements = acknowledgementStore.snapshot(now)

        fun DeviceStatus.applyAck(): DeviceStatus {
            val ackUntil = acknowledgements[id]?.takeIf { it > now }
            return copy(ackUntilEpochMillis = ackUntil)
        }

        fun List<DeviceStatus>.withAck(): List<DeviceStatus> = map { it.applyAck() }

        return summary.copy(
            allDevices = summary.allDevices.withAck(),
            gateways = summary.gateways.withAck(),
            aps = summary.aps.withAck(),
            switches = summary.switches.withAck(),
            routers = summary.routers.withAck(),
            offlineGateways = summary.offlineGateways.withAck(),
            offlineAps = summary.offlineAps.withAck(),
            offlineBackbone = summary.offlineBackbone.withAck(),
            highLatencyCore = summary.highLatencyCore.withAck()
        )
    }


    open fun triggerSimulatedGatewayOutage(context: Context) {
        val currentSummary = _summaryFlow.value
        if (currentSummary != null) {
            val fakeGateway = DeviceStatus(
                id = "fake-gateway-id",
                name = "Fake Gateway",
                role = "gateway",
                isGateway = true,
                isAp = false,
                isBackbone = true,
                online = false,
                latencyMs = null,
                status = "offline"
            )
            val newOfflineGateways = currentSummary.offlineGateways + fakeGateway
            val newSummary = currentSummary.copy(offlineGateways = newOfflineGateways)
            _summaryFlow.value = newSummary
            MyFirebaseMessagingService.sendNotification(
                context,
                "Gateway Offline",
                "Fake Gateway is offline.",
                "buz"
            )
        }
    }

    open fun clearSimulatedGatewayOutage() {
        val currentSummary = _summaryFlow.value
        if (currentSummary != null) {
            val newOfflineGateways = currentSummary.offlineGateways.filter { it.id != "fake-gateway-id" }
            val newSummary = currentSummary.copy(offlineGateways = newOfflineGateways)
            _summaryFlow.value = newSummary
        }
    }
    
    suspend fun isServerReachable(context: Context, session: Session?): Boolean {
        if (session == null) return false

        val connectivityManager =
            context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
        val activeNetwork = connectivityManager.activeNetwork ?: return false
        val networkCapabilities =
            connectivityManager.getNetworkCapabilities(activeNetwork) ?: return false

        val isWifi = networkCapabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI)
        if (!isWifi) return false

        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url(session.uispBaseUrl.trimEnd('/') + "/nms/heartbeat")
                    .get()
                    .header("Accept", "application/json")
                    .header("x-auth-token", session.uispToken.trim())
                    .header("User-Agent", USER_AGENT)
                    .build()
                httpClient.newCall(request).execute().use { response ->
                    response.isSuccessful
                }
            } catch (e: IOException) {
                false
            }
        }
    }

    //endregion

    //region Data Parsing and Transformation

    /**
     * Parses the raw JSON string of devices into a structured [DevicesSummary].
     */
    private fun parseDevices(payload: String): DevicesSummary {
        try {
            val root = JSONArray(payload)
            val allDevices = mutableListOf<DeviceStatus>()
            val gateways = mutableListOf<DeviceStatus>()
            val aps = mutableListOf<DeviceStatus>()
            val switches = mutableListOf<DeviceStatus>()
            val routers = mutableListOf<DeviceStatus>()
            val offlineGateways = mutableListOf<DeviceStatus>()
            val offlineAps = mutableListOf<DeviceStatus>()
            val offlineBackbone = mutableListOf<DeviceStatus>()
            val highLatencyCore = mutableListOf<DeviceStatus>()

            for (i in 0 until root.length()) {
                val device = root.optJSONObject(i) ?: continue
                val identification = device.optJSONObject("identification")
                val overview = device.optJSONObject("overview")

                val id = identification?.optString("id")
                    ?: identification?.optString("mac")
                    ?: device.optString("id", "unknown")
                val name = identification?.optString("name")?.takeIf { it.isNotBlank() } ?: id
                val roleRaw = identification?.optString("role")?.lowercase()?.trim() ?: "device"
                val role = normalizeRole(roleRaw)
                val online = isOnline(overview?.optString("status"))
                val latencyMs = extractLatency(overview)

                val isGateway = role == "gateway"
                val isAp = role == "ap"
                val isBackbone = role == "router" || role == "switch" || isGateway || isAp

                val status = DeviceStatus(
                    id = id,
                    name = name,
                    role = role,
                    isGateway = isGateway,
                    isAp = isAp,
                    isBackbone = isBackbone,
                    online = online,
                    latencyMs = latencyMs,
                    status = overview?.optString("status") ?: "unknown"
                )
                allDevices.add(status)

                when (role) {
                    "gateway" -> gateways.add(status)
                    "ap" -> aps.add(status)
                    "switch" -> switches.add(status)
                    "router" -> routers.add(status)
                }

                if (!online) {
                    when {
                        isGateway -> offlineGateways.add(status)
                        isAp -> offlineAps.add(status)
                        isBackbone -> offlineBackbone.add(status)
                    }
                }

                if (latencyMs != null && latencyMs > LATENCY_ALERT_THRESHOLD_MS && (isGateway || isAp)) {
                    highLatencyCore += status
                }
            }

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
        } catch (e: JSONException) {
            throw IOException("Failed to parse UISP devices response. Expected JSON array.", e)
        }
    }

    private fun extractLatency(overview: JSONObject?): Double? {
        if (overview == null) return null
        val keys = listOf("latency", "latencyMs", "ping", "rtt")
        for (key in keys) {
            val candidate = overview.optDouble(key, Double.NaN)
            if (!candidate.isNaN()) return candidate
        }
        return null
    }

    private fun isOnline(status: String?): Boolean {
        val normalized = status?.lowercase() ?: return false
        return normalized in ONLINE_STATUSES
    }

    private fun normalizeRole(role: String): String {
        return when (role) {
            "access-point", "accesspoint", "base-station", "basestation", "ap" -> "ap"
            else -> role
        }
    }

    //endregion

    //region URL Sanitization

    private fun normalizeBackendUrl(rawUrl: String): String {
        var trimmed = rawUrl.trim()
        if (!trimmed.contains("://")) {
            trimmed = "https://$trimmed"
        }
        return trimmed.removeSuffix("/")
    }

    private fun sanitizeBaseUrl(rawUrl: String): String {
        var trimmed = rawUrl.trim()
        if (!trimmed.contains("://")) {
            trimmed = "https://$trimmed"
        }
        return trimmed.removeSuffix("/")
    }

    private fun extractErrorMessage(body: String): String? =
        try {
            val json = JSONObject(body)
            json.optString("message").takeIf { it.isNotBlank() }
                ?: json.optString("error").takeIf { it.isNotBlank() }
        } catch (ignored: Exception) {
            null
        }

    //endregion

    //region Companion and Configuration

    class AuthException(message: String) : IOException(message)

    companion object {
        private const val USER_AGENT = "UISP-NOC-Android/1.0"
        private val ONLINE_STATUSES = setOf(
            "ok", "online", "active", "connected", "reachable", "enabled"
        )
        private const val LATENCY_ALERT_THRESHOLD_MS = 450.0

        /**
         * Creates a default OkHttpClient with pre-configured settings.
         */
        fun defaultClient(): OkHttpClient {
            val logging = HttpLoggingInterceptor().apply {
                level = HttpLoggingInterceptor.Level.BASIC
            }
            return OkHttpClient.Builder()
                .connectTimeout(10, TimeUnit.SECONDS)
                .readTimeout(10, TimeUnit.SECONDS)
                .writeTimeout(10, TimeUnit.SECONDS)
                .addInterceptor(logging)
                .build()
        }
    }
    //endregion
}

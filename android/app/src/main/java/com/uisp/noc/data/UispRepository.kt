package com.uisp.noc.data

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.FormBody
import okhttp3.JavaNetCookieJar
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.logging.HttpLoggingInterceptor
import org.json.JSONArray
import org.json.JSONException
import org.json.JSONObject
import java.io.IOException
import java.net.CookieManager
import java.net.CookiePolicy
import java.util.concurrent.TimeUnit
import com.uisp.noc.data.model.DeviceStatus
import com.uisp.noc.data.model.DevicesSummary

class UispRepository(
    private val httpClient: OkHttpClient = defaultClient()
) {

    suspend fun authenticate(
        backendUrl: String,
        username: String,
        password: String
    ): Session = withContext(Dispatchers.IO) {
        val normalizedBackend = normalizeBackendUrl(backendUrl)
        val cookieManager = CookieManager().apply {
            setCookiePolicy(CookiePolicy.ACCEPT_ALL)
        }
        val client = httpClient.newBuilder()
            .cookieJar(JavaNetCookieJar(cookieManager))
            .build()

        val loginBody = FormBody.Builder()
            .add("username", username)
            .add("password", password)
            .build()

        val loginRequest = Request.Builder()
            .url("$normalizedBackend/?action=login")
            .post(loginBody)
            .header("User-Agent", USER_AGENT)
            .build()

        client.newCall(loginRequest).execute().use { response ->
            // We intentionally ignore the body here; success is determined
            // by the ability to fetch the mobile config while authenticated.
            if (!response.isSuccessful && response.code != 302) {
                throw AuthException("Unable to authenticate with backend (HTTP ${response.code})")
            }
        }

        val configRequest = Request.Builder()
            .url("$normalizedBackend/?ajax=mobile_config")
            .get()
            .header("Accept", "application/json")
            .header("User-Agent", USER_AGENT)
            .build()

        client.newCall(configRequest).execute().use { response ->
            val body = response.body?.string()
                ?: throw IOException("Empty response from mobile_config endpoint")

            if (response.code == 401) {
                throw AuthException("Invalid username or password")
            }

            if (!response.isSuccessful) {
                val message = extractErrorMessage(body)
                throw IOException("Failed to obtain UISP token: HTTP ${response.code} $message")
            }

            val payload = try {
                JSONObject(body)
            } catch (ex: JSONException) {
                throw IOException("Unexpected response from mobile_config endpoint", ex)
            }

            val baseUrl = payload.optString("uisp_base_url").takeIf { it.isNotBlank() }
                ?: throw IOException("UISP base URL missing from backend response")
            val token = payload.optString("uisp_token").takeIf { it.isNotBlank() }
                ?: throw IOException("UISP token missing from backend response")

            Session(
                backendUrl = normalizedBackend,
                username = username,
                uispBaseUrl = sanitizeBaseUrl(baseUrl),
                uispToken = token
            )
        }
    }

    suspend fun fetchSummary(session: Session): DevicesSummary = withContext(Dispatchers.IO) {
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

            parseDevices(body)
        }
    }

    private fun parseDevices(payload: String): DevicesSummary {
        val root = JSONArray(payload)
        val offlineGateways = mutableListOf<DeviceStatus>()
        val offlineBackbone = mutableListOf<DeviceStatus>()
        val offlineCpes = mutableListOf<DeviceStatus>()
        val highLatency = mutableListOf<DeviceStatus>()

        var totalGateways = 0
        var totalBackbone = 0
        var totalCpes = 0

        for (i in 0 until root.length()) {
            val device = root.optJSONObject(i) ?: continue
            val identification = device.optJSONObject("identification")
            val overview = device.optJSONObject("overview")

            val id = identification?.optString("id")
                ?: identification?.optString("mac")
                ?: device.optString("id", "unknown")
            val name = identification?.optString("name")?.takeIf { it.isNotBlank() } ?: id
            val role = identification?.optString("role")?.lowercase()?.trim() ?: "device"
            val online = isOnline(overview?.optString("status"))
            val latencyMs = extractLatency(overview)

            val isGateway = role == "gateway"
            val isBackbone = role == "router" || role == "switch" || isGateway
            val status = DeviceStatus(
                id = id,
                name = name,
                role = role,
                isGateway = isGateway,
                isBackbone = isBackbone,
                online = online,
                latencyMs = latencyMs
            )

            when {
                isGateway -> {
                    totalGateways += 1
                    if (!online) offlineGateways += status
                }
                isBackbone -> {
                    totalBackbone += 1
                    if (!online) offlineBackbone += status
                }
                else -> {
                    totalCpes += 1
                    if (!online) offlineCpes += status
                }
            }

            if (latencyMs != null && latencyMs > LATENCY_ALERT_THRESHOLD_MS) {
                highLatency += status
            }
        }

        return DevicesSummary(
            lastUpdatedEpochMillis = System.currentTimeMillis(),
            totalGateways = totalGateways,
            offlineGateways = offlineGateways,
            totalBackbone = totalBackbone,
            offlineBackbone = offlineBackbone,
            totalCpes = totalCpes,
            offlineCpes = offlineCpes,
            highLatencyGateways = highLatency
        )
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

    private fun normalizeBackendUrl(rawUrl: String): String {
        var trimmed = rawUrl.trim()
        if (!trimmed.contains("://")) {
            trimmed = "https://$trimmed"
        }
        // Remove trailing slash but keep path segments intact.
        trimmed = trimmed.removeSuffix("/")
        return trimmed
    }

    private fun sanitizeBaseUrl(rawUrl: String): String {
        var trimmed = rawUrl.trim()
        if (trimmed.contains("://").not()) {
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

    class AuthException(message: String) : IOException(message)

    companion object {
        private const val USER_AGENT = "UISP-NOC-Android/1.0"
        private val ONLINE_STATUSES = setOf(
            "ok", "online", "active", "connected", "reachable", "enabled"
        )
        private const val LATENCY_ALERT_THRESHOLD_MS = 450.0

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
}

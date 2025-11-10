package com.uisp.noc.data

import com.uisp.noc.data.model.DeviceStatus
import com.uisp.noc.data.model.DevicesSummary
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

/**
 * Manages data operations for the UISP NOC application, handling authentication and
 * data fetching from a UISP backend using OkHttp.
 *
 * @property httpClient The OkHttpClient used for all network requests.
 */
open class UispRepository(private val httpClient: OkHttpClient = defaultClient()) {

    //region Public API

    /**
     * Authenticates with the UISP backend using the provided credentials.
     */
    open suspend fun authenticate(
        backendUrl: String,
        username: String,
        password: String
    ): Session = withContext(Dispatchers.IO) {
        val normalizedBackend = normalizeBackendUrl(backendUrl)

        // Create a dedicated client with a cookie jar to handle the session
        val cookieManager = CookieManager().apply {
            setCookiePolicy(CookiePolicy.ACCEPT_ALL)
        }
        val authClient = httpClient.newBuilder()
            .cookieJar(JavaNetCookieJar(cookieManager))
            .build()

        // Step 1: Perform form-based login to establish a session cookie.
        val loginBody = FormBody.Builder()
            .add("username", username)
            .add("password", password)
            .build()

        val loginRequest = Request.Builder()
            .url("$normalizedBackend/?action=login")
            .post(loginBody)
            .header("User-Agent", USER_AGENT)
            .build()

        authClient.newCall(loginRequest).execute().use { response ->
            // A successful login redirects (302), so we treat that as success.
            if (!response.isSuccessful && !response.isRedirect) {
                val message = response.body?.string().orEmpty()
                throw AuthException("Unable to authenticate with backend (HTTP ${response.code}). $message")
            }
        }

        // Step 2: Fetch the mobile config to get the API token. The cookie is sent automatically.
        val configRequest = Request.Builder()
            .url("$normalizedBackend/?ajax=mobile_config")
            .get()
            .header("Accept", "application/json")
            .header("User-Agent", USER_AGENT)
            .build()

        authClient.newCall(configRequest).execute().use { response ->
            val body = response.body?.string()
                ?: throw IOException("Empty response from mobile_config endpoint")

            if (response.code == 401) {
                throw AuthException("Invalid username or password. The session may have expired.")
            }

            if (!response.isSuccessful) {
                val message = extractErrorMessage(body)
                throw IOException("Failed to obtain UISP token: HTTP ${response.code} ${message ?: ""}")
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

    /**
     * Fetches a summary of all network devices from the UISP API.
     */
    open suspend fun fetchSummary(session: Session): DevicesSummary = withContext(Dispatchers.IO) {
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

    //endregion

    //region Data Parsing and Transformation

    /**
     * Parses the raw JSON string of devices into a structured [DevicesSummary].
     */
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

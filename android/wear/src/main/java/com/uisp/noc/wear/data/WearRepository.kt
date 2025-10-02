package com.uisp.noc.wear.data

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.Call
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.logging.HttpLoggingInterceptor
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.util.concurrent.TimeUnit

class WearRepository(
    private val client: OkHttpClient = defaultClient()
) {
    suspend fun fetchSummary(config: WearConfig): DevicesSummary = withContext(Dispatchers.IO) {
        val url = config.normalizedBaseUrl() + "/nms/api/v2.1/devices"
        val request = Request.Builder()
            .url(url)
            .header("accept", "application/json")
            .header("x-auth-token", config.apiToken.trim())
            .build()

        val response = executeRequest(client, request)
        val bodyString = response.body?.string()
            ?: throw IOException("Empty response body")

        if (!response.isSuccessful) {
            throw IOException("HTTP " + response.code + ": " + bodyString)
        }

        parseDevices(bodyString)
    }

    private fun executeRequest(client: OkHttpClient, request: Request): Response {
        val call: Call = client.newCall(request)
        return call.execute()
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
            val name = identification?.optString("name")?.takeIf { it.isNotBlank() }
                ?: id
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
        if (status == null) return false
        val normalized = status.lowercase()
        return normalized in ONLINE_STATUSES
    }

    companion object {
        private val ONLINE_STATUSES = setOf(
            "ok", "online", "active", "connected", "reachable", "enabled"
        )
        private const val LATENCY_ALERT_THRESHOLD_MS = 450.0

        private fun defaultClient(): OkHttpClient {
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

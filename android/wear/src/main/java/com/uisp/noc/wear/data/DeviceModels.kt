package com.uisp.noc.wear.data

data class DeviceStatus(
    val id: String,
    val name: String,
    val role: String,
    val isGateway: Boolean,
    val isBackbone: Boolean,
    val online: Boolean,
    val latencyMs: Double?
)

data class DevicesSummary(
    val lastUpdatedEpochMillis: Long,
    val totalGateways: Int,
    val offlineGateways: List<DeviceStatus>,
    val totalBackbone: Int,
    val offlineBackbone: List<DeviceStatus>,
    val totalCpes: Int,
    val offlineCpes: List<DeviceStatus>,
    val highLatencyGateways: List<DeviceStatus>
) {
    val totalDevices: Int = totalGateways + totalBackbone + totalCpes
    val totalOffline: Int =
        offlineGateways.size + offlineBackbone.size + offlineCpes.size
    val healthPercent: Int = if (totalDevices == 0) 100 else {
        (((totalDevices - totalOffline).toDouble() / totalDevices.toDouble()) * 100).toInt()
    }
}

data class WearConfig(
    val baseUrl: String,
    val apiToken: String
) {
    fun isValid(): Boolean = baseUrl.isNotBlank() && apiToken.isNotBlank()

    fun normalizedBaseUrl(): String {
        var trimmed = baseUrl.trim()
        if (!trimmed.startsWith("http://") && !trimmed.startsWith("https://")) {
            trimmed = "https://"
        }
        return trimmed.trimEnd('/')
    }
}

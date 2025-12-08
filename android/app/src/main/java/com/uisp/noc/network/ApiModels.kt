package com.uisp.noc.network

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class MobileConfigDto(
    @SerialName("uisp_base_url") val uispBaseUrl: String,
    @SerialName("api_base_url") val apiBaseUrl: String,
    @SerialName("feature_flags") val featureFlags: Map<String, Boolean> = emptyMap(),
    @SerialName("push_register_url") val pushRegisterUrl: String? = null,
    @SerialName("environment") val environment: String? = null,
    @SerialName("version") val version: String? = null,
    @SerialName("banner") val banner: String? = null
)

@Serializable
data class PushRegisterRequest(
    @SerialName("token") val token: String,
    @SerialName("platform") val platform: String,
    @SerialName("app_version") val appVersion: String,
    @SerialName("locale") val locale: String
)

@Serializable
data class PushRegisterResponse(
    @SerialName("request_id") val requestId: String? = null,
    @SerialName("message") val message: String? = null
)

@Serializable
data class DeviceDto(
    val id: String,
    val name: String,
    val role: String,
    @SerialName("site_id") val siteId: String? = null,
    val online: Boolean,
    @SerialName("latency_ms") val latencyMs: Double? = null,
    @SerialName("ack_until") val ackUntil: Long? = null,
    @SerialName("suppressed_reason") val suppressedReason: String? = null
)

@Serializable
data class DeviceSummaryDto(
    @SerialName("last_updated") val lastUpdated: Long,
    @SerialName("devices") val devices: List<DeviceDto> = emptyList()
)

@Serializable
data class IncidentDto(
    val id: String,
    @SerialName("device_id") val deviceId: String,
    val type: String,
    val severity: String,
    @SerialName("started_at") val startedAt: String,
    @SerialName("resolved_at") val resolvedAt: String? = null,
    @SerialName("ack_until") val ackUntil: String? = null,
    @SerialName("suppressed") val suppressed: Boolean = false,
    @SerialName("suppression_reason") val suppressionReason: String? = null
)

@Serializable
data class AlertDeliveryDto(
    @SerialName("incident_id") val incidentId: String,
    val channel: String,
    val status: String,
    val error: String? = null,
    @SerialName("request_id") val requestId: String? = null
)

package com.uisp.noc.data.model

import kotlinx.serialization.Serializable

@Serializable
data class DeviceStatus(
    val id: String,
    val name: String,
    val role: String,
    val isGateway: Boolean,
    val isBackbone: Boolean,
    val online: Boolean,
    val latencyMs: Double?
)

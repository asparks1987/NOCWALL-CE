package com.uisp.noc.data.model

import kotlinx.serialization.Serializable

/**
 * Represents the real-time status of a single network device.
 *
 * @property id The unique identifier for the device (e.g., MAC address or a UISP ID).
 * @property name The user-defined name of the device. Defaults to the ID if not set.
 * @property role The functional role of the device in the network (e.g., "gateway", "router").
 * @property isGateway True if the device is a primary internet gateway.
 * @property isBackbone True if the device is part of the core network backbone (e.g., router, switch).
 * @property online True if the device is currently connected and responsive.
 * @property latencyMs The device's network latency in milliseconds, if available.
 */
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

package com.uisp.noc.data.model

import kotlinx.serialization.Serializable

/**
 * A summary of the network's device statuses, compiled from a UISP backend.
 *
 * @property lastUpdatedEpochMillis The timestamp (in milliseconds) when this summary was generated.
 * @property totalGateways The total number of gateway devices in the network.
 * @property offlineGateways A list of gateway devices that are currently offline.
 * @property totalBackbone The total number of backbone devices (routers, switches) in the network.
 * @property offlineBackbone A list of backbone devices that are currently offline.
 * @property totalCpes The total number of Customer-Premises Equipment (CPE) devices.
 * @property offlineCpes A list of CPE devices that are currently offline.
 * @property highLatencyGateways A list of devices experiencing high network latency.
 */
@Serializable
data class DevicesSummary(
    val lastUpdatedEpochMillis: Long,
    val totalGateways: Int,
    val offlineGateways: List<DeviceStatus>,
    val totalBackbone: Int,
    val offlineBackbone: List<DeviceStatus>,
    val totalCpes: Int,
    val offlineCpes: List<DeviceStatus>,
    val highLatencyGateways: List<DeviceStatus>
)

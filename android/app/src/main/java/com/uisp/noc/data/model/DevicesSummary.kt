package com.uisp.noc.data.model

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

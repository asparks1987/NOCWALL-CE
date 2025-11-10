package com.uisp.noc.data.model

data class DevicesSummary(
    val lastUpdatedEpochMillis: Long = 0,
    val gateways: List<DeviceStatus> = emptyList(),
    val switches: List<DeviceStatus> = emptyList(),
    val routers: List<DeviceStatus> = emptyList(),
    val offlineGateways: List<DeviceStatus> = emptyList(),
    val offlineBackbone: List<DeviceStatus> = emptyList(),
    val offlineCpes: List<DeviceStatus> = emptyList(),
    val highLatencyGateways: List<DeviceStatus> = emptyList()
)

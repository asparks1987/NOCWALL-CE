package com.uisp.noc.data.model

data class DevicesSummary(
    val allDevices: List<DeviceStatus> = emptyList(),
    val lastUpdatedEpochMillis: Long = 0,
    val gateways: List<DeviceStatus> = emptyList(),
    val aps: List<DeviceStatus> = emptyList(),
    val switches: List<DeviceStatus> = emptyList(),
    val routers: List<DeviceStatus> = emptyList(),
    val offlineGateways: List<DeviceStatus> = emptyList(),
    val offlineAps: List<DeviceStatus> = emptyList(),
    val offlineBackbone: List<DeviceStatus> = emptyList(),
    val highLatencyCore: List<DeviceStatus> = emptyList()
)

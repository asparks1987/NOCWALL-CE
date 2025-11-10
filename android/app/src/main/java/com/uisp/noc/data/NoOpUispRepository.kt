package com.uisp.noc.data

import com.uisp.noc.data.model.DeviceStatus
import com.uisp.noc.data.model.DevicesSummary

/**
 * A no-op implementation of [UispRepository] that does nothing.
 * This is used for placeholder ViewModels during asynchronous initialization.
 */
class NoOpUispRepository : UispRepository() {
    override suspend fun authenticate(
        backendUrl: String,
        username: String,
        password: String
    ): Session {
        // Do nothing, return a dummy session
        return Session("", "", "", "")
    }

    override suspend fun fetchSummary(session: Session): DevicesSummary {
        // Do nothing, return empty summary
        return DevicesSummary(
            lastUpdatedEpochMillis = 0,
            totalGateways = 0,
            offlineGateways = emptyList<DeviceStatus>(),
            totalBackbone = 0,
            offlineBackbone = emptyList<DeviceStatus>(),
            totalCpes = 0,
            offlineCpes = emptyList<DeviceStatus>(),
            highLatencyGateways = emptyList<DeviceStatus>()
        )
    }
}
package com.uisp.noc.data

import android.content.Context
import com.uisp.noc.data.model.DevicesSummary

/**
 * A no-op implementation of [UispRepository] that does nothing.
 * This is used for placeholder ViewModels during asynchronous initialization.
 */
class NoOpUispRepository : UispRepository() {
    override suspend fun authenticate(
        uispUrl: String,
        displayName: String,
        apiToken: String
    ): Session {
        // Do nothing, return a dummy session
        return Session("", "", "", "")
    }

    override suspend fun fetchSummary(session: Session, context: Context) {
        // Do nothing, just emit an empty summary
        _summaryFlow.value = DevicesSummary(
            lastUpdatedEpochMillis = 0,
            gateways = emptyList(),
            aps = emptyList(),
            switches = emptyList(),
            routers = emptyList(),
            offlineGateways = emptyList(),
            offlineAps = emptyList(),
            offlineBackbone = emptyList(),
            highLatencyCore = emptyList()
        )
    }

    override fun acknowledgeDevice(deviceId: String, durationMinutes: Int) {
        // no-op
    }

    override fun clearAcknowledgement(deviceId: String) {
        // no-op
    }
}

package com.uisp.noc

import android.content.Context
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.uisp.noc.data.SessionStore
import com.uisp.noc.data.model.DeviceStatus

class GatewayStatusWorker(
    private val appContext: Context,
    workerParams: WorkerParameters
) : CoroutineWorker(appContext, workerParams) {

    // In-memory store for offline gateways. A persistent store would be better for a real app.
    private val offlineGateways = mutableMapOf<String, OfflineGatewayInfo>()

    override suspend fun doWork(): Result {
        val sessionStore = SessionStore.getInstance(appContext)
        val session = sessionStore.load() ?: return Result.success()
        val repository = Injector.getRepository()

        try {
            repository.fetchSummary(session, appContext)
            val summary = repository.summaryFlow.value ?: return Result.failure()
            val now = System.currentTimeMillis()

            // Update the offline gateways map
            summary.offlineGateways.forEach { gateway ->
                if (!offlineGateways.containsKey(gateway.id)) {
                    offlineGateways[gateway.id] = OfflineGatewayInfo(gateway, now)
                }
            }

            // Check for gateways that are back online
            val onlineGateways = summary.offlineGateways.map { it.id }.toSet()
            offlineGateways.keys.removeIf { !onlineGateways.contains(it) }

            // Check for notifications
            offlineGateways.values.forEach { info ->
                val offlineDuration = now - info.firstSeenOffline
                val minutesOffline = offlineDuration / 1000 / 60

                if (minutesOffline >= 1 && !info.notifiedInitial) {
                    NotificationHelper.showOfflineNotification(appContext, info.gateway.name)
                    info.notifiedInitial = true
                    info.lastNotified = now
                } else if (info.notifiedInitial) {
                    val minutesSinceLastNotification = (now - info.lastNotified) / 1000 / 60
                    if (minutesSinceLastNotification >= 15) {
                        NotificationHelper.showOfflineReminderNotification(appContext, info.gateway.name)
                        info.lastNotified = now
                    }
                }
            }
        } catch (e: Exception) {
            // Silently fail to avoid crashing the worker
            return Result.failure()
        }

        return Result.success()
    }

    private data class OfflineGatewayInfo(
        val gateway: DeviceStatus,
        val firstSeenOffline: Long,
        var notifiedInitial: Boolean = false,
        var lastNotified: Long = 0
    )
}

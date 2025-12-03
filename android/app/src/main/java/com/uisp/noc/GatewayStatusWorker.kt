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

    companion object {
        // In-memory store for offline gateways. A persistent store would be better for a real app.
        private val offlineGateways = mutableMapOf<String, OfflineGatewayInfo>()
    }

    override suspend fun doWork(): Result {
        val sessionStore = SessionStore.getInstance(appContext)
        val session = sessionStore.load() ?: return Result.success()
        val repository = Injector.getRepository(appContext)

        try {
            repository.fetchSummary(session, appContext)
            val summary = repository.summaryFlow.value ?: return Result.failure()
            val now = System.currentTimeMillis()

            val currentOfflineIds = summary.offlineGateways.map { it.id }.toSet()

            // 1. Check for gateways that came back online
            val newlyOnlineGateways = offlineGateways.filter { (id, _) -> id !in currentOfflineIds }
            newlyOnlineGateways.forEach { (id, info) ->
                MyFirebaseMessagingService.sendNotification(
                    appContext,
                    title = "${info.gateway.name} is online",
                    messageBody = "The device is now connected.",
                    sound = "default_online"
                )
                offlineGateways.remove(id)
            }

            // 2. Add newly offline gateways to tracking
            summary.offlineGateways.forEach { gateway ->
                if (!offlineGateways.containsKey(gateway.id)) {
                    offlineGateways[gateway.id] = OfflineGatewayInfo(gateway, now)
                }
            }

            // 3. Handle notifications for gateways that are still offline
            offlineGateways.values.forEach { info ->
                val minutesOffline = (now - info.firstSeenOffline) / 1000 / 60

                if (minutesOffline >= 1 && !info.notifiedInitial) {
                    MyFirebaseMessagingService.sendNotification(
                        appContext,
                        title = "${info.gateway.name} is offline",
                        messageBody = "The device has been offline for 1 minute.",
                        sound = "buz"
                    )
                    info.notifiedInitial = true
                    info.lastNotified = now
                } else if (info.notifiedInitial) {
                    val minutesSinceLastNotification = (now - info.lastNotified) / 1000 / 60
                    if (minutesSinceLastNotification >= 15) {
                        MyFirebaseMessagingService.sendNotification(
                            appContext,
                            title = "${info.gateway.name} is still offline",
                            messageBody = "The device has been offline for over 15 minutes.",
                            sound = "buz"
                        )
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

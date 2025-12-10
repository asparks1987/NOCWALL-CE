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
        // In-memory stores. In a production app this should be persisted.
        private val offlineGateways = mutableMapOf<String, OfflineGatewayInfo>()
        private val offlineAps = mutableMapOf<String, OfflineGatewayInfo>()
        private val offlineBackbone = mutableMapOf<String, OfflineGatewayInfo>()

        private const val GATEWAY_INITIAL_MINUTES = 1L    // gateways alert quickly (desktop parity)
        private const val GATEWAY_REPEAT_MINUTES = 15L
        private const val AP_INITIAL_MINUTES = 15L        // APs must be down 15m before alerting (desktop parity)
        private const val AP_REPEAT_MINUTES = 15L
        private const val BACKBONE_INITIAL_MINUTES = 15L  // treat backbone like APs for noise control
        private const val BACKBONE_REPEAT_MINUTES = 15L
    }

    override suspend fun doWork(): Result {
        val sessionStore = SessionStore.getInstance(appContext)
        val session = sessionStore.load() ?: return Result.success()
        val repository = Injector.getRepository(appContext)

        try {
            repository.fetchSummary(session, appContext)
            val summary = repository.summaryFlow.value ?: return Result.failure()
            val now = System.currentTimeMillis()

            // 1) Handle devices that came back online
            handleRestoredDevices(
                currentOffline = summary.offlineGateways,
                tracked = offlineGateways,
                now = now,
                onlineMessage = "The device is now connected."
            )
            handleRestoredDevices(
                currentOffline = summary.offlineAps,
                tracked = offlineAps,
                now = now,
                onlineMessage = "The access point is now connected."
            )
            handleRestoredDevices(
                currentOffline = summary.offlineBackbone,
                tracked = offlineBackbone,
                now = now,
                onlineMessage = "The backbone device is now connected."
            )

            // 2) Track newly offline devices
            trackOffline(summary.offlineGateways, offlineGateways, now)
            trackOffline(summary.offlineAps, offlineAps, now)
            trackOffline(summary.offlineBackbone, offlineBackbone, now)

            // 3) Notify for ongoing outages with desktop parity timing
            notifyOngoing(
                map = offlineGateways,
                now = now,
                initialThresholdMinutes = GATEWAY_INITIAL_MINUTES,
                repeatThresholdMinutes = GATEWAY_REPEAT_MINUTES,
                initialMessage = { "${it.name} is offline" },
                repeatMessage = { "${it.name} is still offline" }
            )
            notifyOngoing(
                map = offlineAps,
                now = now,
                initialThresholdMinutes = AP_INITIAL_MINUTES,
                repeatThresholdMinutes = AP_REPEAT_MINUTES,
                initialMessage = { "${it.name} is offline" },
                repeatMessage = { "${it.name} is still offline" }
            )
            notifyOngoing(
                map = offlineBackbone,
                now = now,
                initialThresholdMinutes = BACKBONE_INITIAL_MINUTES,
                repeatThresholdMinutes = BACKBONE_REPEAT_MINUTES,
                initialMessage = { "${it.name} is offline" },
                repeatMessage = { "${it.name} is still offline" }
            )
        } catch (e: Exception) {
            // Silently fail to avoid crashing the worker
            return Result.failure()
        }

        return Result.success()
    }

    private fun handleRestoredDevices(
        currentOffline: List<DeviceStatus>,
        tracked: MutableMap<String, OfflineGatewayInfo>,
        now: Long,
        onlineMessage: String
    ) {
        val currentOfflineIds = currentOffline.map { it.id }.toSet()
        val restored = tracked.filter { (id, _) -> id !in currentOfflineIds }
        restored.forEach { (id, info) ->
            MyFirebaseMessagingService.sendNotification(
                appContext,
                title = "${info.device.name} is online",
                messageBody = onlineMessage,
                sound = "default_online"
            )
            tracked.remove(id)
        }
    }

    private fun trackOffline(
        offlineList: List<DeviceStatus>,
        tracked: MutableMap<String, OfflineGatewayInfo>,
        now: Long
    ) {
        offlineList.forEach { device ->
            if (!tracked.containsKey(device.id)) {
                tracked[device.id] = OfflineGatewayInfo(device, now)
            }
        }
    }

    private fun notifyOngoing(
        map: MutableMap<String, OfflineGatewayInfo>,
        now: Long,
        initialThresholdMinutes: Long,
        repeatThresholdMinutes: Long,
        initialMessage: (DeviceStatus) -> String,
        repeatMessage: (DeviceStatus) -> String
    ) {
        val nowMinutes = now / 60000
        map.values.forEach { info ->
            // Respect ACKs if provided by backend
            val ackUntil = info.device.ackUntilEpochMillis
            if (ackUntil != null && ackUntil > now) {
                return@forEach
            }

            val minutesOffline = nowMinutes - (info.firstSeenOffline / 60000)
            if (!info.notifiedInitial && minutesOffline >= initialThresholdMinutes) {
                MyFirebaseMessagingService.sendNotification(
                    appContext,
                    title = initialMessage(info.device),
                    messageBody = buildMessage(minutesOffline, initialThresholdMinutes),
                    sound = "buz"
                )
                info.notifiedInitial = true
                info.lastNotified = now
            } else if (info.notifiedInitial) {
                val minutesSinceLast = (now - info.lastNotified) / 60000
                if (minutesSinceLast >= repeatThresholdMinutes) {
                    MyFirebaseMessagingService.sendNotification(
                        appContext,
                        title = repeatMessage(info.device),
                        messageBody = buildMessage(minutesOffline, repeatThresholdMinutes),
                        sound = "buz"
                    )
                    info.lastNotified = now
                }
            }
        }
    }

    private fun buildMessage(minutesOffline: Long, thresholdMinutes: Long): String {
        return if (minutesOffline >= thresholdMinutes) {
            "The device has been offline for over ${thresholdMinutes} minute${if (thresholdMinutes == 1L) "" else "s"}."
        } else {
            "The device is offline."
        }
    }

    private data class OfflineGatewayInfo(
        val device: DeviceStatus,
        val firstSeenOffline: Long,
        var notifiedInitial: Boolean = false,
        var lastNotified: Long = 0
    )
}

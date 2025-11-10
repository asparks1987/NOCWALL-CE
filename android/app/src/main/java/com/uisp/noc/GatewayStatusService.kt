package com.uisp.noc

import android.app.Service
import android.content.Intent
import android.os.IBinder
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import java.util.concurrent.TimeUnit

class GatewayStatusService : Service() {

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            stopForeground(true)
            stopSelf()
            WorkManager.getInstance(this).cancelUniqueWork("gateway-status-worker")
            return START_NOT_STICKY
        }

        val notification = NotificationHelper.createForegroundNotification(this)
        startForeground(NOTIFICATION_ID, notification)

        val workRequest = PeriodicWorkRequestBuilder<GatewayStatusWorker>(15, TimeUnit.MINUTES)
            .build()

        WorkManager.getInstance(this).enqueueUniquePeriodicWork(
            "gateway-status-worker",
            ExistingPeriodicWorkPolicy.KEEP,
            workRequest
        )

        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? {
        return null
    }

    companion object {
        const val ACTION_STOP = "com.uisp.noc.ACTION_STOP"
        private const val NOTIFICATION_ID = 1
    }
}

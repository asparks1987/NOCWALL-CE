package com.uisp.noc

import android.app.Service
import android.content.Intent
import android.os.IBinder
import android.util.Log
import androidx.core.app.NotificationManagerCompat
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import com.uisp.noc.data.SessionStore
import com.uisp.noc.data.UispRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.io.IOException
import java.util.concurrent.TimeUnit

class GatewayStatusService : Service() {

    private val job = SupervisorJob()
    private val scope = CoroutineScope(Dispatchers.IO + job)

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            stopForeground(true)
            stopSelf()
            return START_NOT_STICKY
        }

        // Start the foreground service with a placeholder notification
        val placeholderNotification = NotificationHelper.createForegroundNotification(
            this, 0, 0, 0, 0, 0, 0
        )
        startForeground(NOTIFICATION_ID, placeholderNotification)

        // Launch a coroutine to update the notification periodically
        scope.launch {
            delay(5000) // Add a 5-second delay to avoid race conditions on login
            while (true) {
                updateNotification()
                delay(TimeUnit.MINUTES.toMillis(1))
            }
        }

        // Schedule the worker for notifications
        val workRequest = PeriodicWorkRequestBuilder<GatewayStatusWorker>(15, TimeUnit.MINUTES)
            .build()

        WorkManager.getInstance(this).enqueueUniquePeriodicWork(
            "gateway-status-worker",
            ExistingPeriodicWorkPolicy.KEEP,
            workRequest
        )

        return START_STICKY
    }

    private suspend fun updateNotification() {
        val sessionStore = SessionStore.getInstance(this)
        val session = sessionStore.load() ?: return
        val repository = Injector.getRepository(this)

        try {
            repository.fetchSummary(session, this)
            val summary = repository.summaryFlow.value ?: return
            val notification = NotificationHelper.createForegroundNotification(
                this,
                summary.gateways.count { it.online },
                summary.gateways.count { !it.online },
                summary.switches.count { it.online },
                summary.switches.count { !it.online },
                summary.routers.count { it.online },
                summary.routers.count { !it.online }
            )
            with(NotificationManagerCompat.from(this)) {
                notify(NOTIFICATION_ID, notification)
            }
        } catch (e: UispRepository.AuthException) {
            Log.e(TAG, "Authentication error", e)
            NotificationHelper.sendLoginRequiredNotification(this)
            stopSelf() // Stop the service as we can't continue without a valid session
        } catch (e: IOException) {
            Log.e(TAG, "Network error while updating notification", e)
            // Optionally, update the notification to show a disconnected/error state
        } catch (e: Exception) {
            Log.e(TAG, "Failed to update notification", e)
        }
    }


    override fun onBind(intent: Intent?): IBinder? {
        return null
    }

    override fun onDestroy() {
        super.onDestroy()
        job.cancel()
        WorkManager.getInstance(this).cancelUniqueWork("gateway-status-worker")
    }

    companion object {
        const val ACTION_STOP = "com.uisp.noc.ACTION_STOP"
        private const val NOTIFICATION_ID = 1
        private const val TAG = "GatewayStatusService"
    }
}

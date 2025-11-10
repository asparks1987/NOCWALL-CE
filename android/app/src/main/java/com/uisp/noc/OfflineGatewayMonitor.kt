package com.uisp.noc

import android.content.Context
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import java.util.concurrent.TimeUnit

object OfflineGatewayMonitor {

    private const val WORK_TAG = "gateway-status-worker"

    fun start(context: Context) {
        val workRequest = PeriodicWorkRequestBuilder<GatewayStatusWorker>(1, TimeUnit.MINUTES)
            .build()

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            WORK_TAG,
            androidx.work.ExistingPeriodicWorkPolicy.REPLACE,
            workRequest
        )
    }

    fun stop(context: Context) {
        WorkManager.getInstance(context).cancelUniqueWork(WORK_TAG)
    }
}

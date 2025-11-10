package com.uisp.noc

import android.content.Context
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.uisp.noc.data.UispRepository

class GatewayStatusWorker(
    appContext: Context,
    workerParams: WorkerParameters
) : CoroutineWorker(appContext, workerParams) {

    override suspend fun doWork(): Result {
        // I will add the logic to check for offline gateways here.
        return Result.success()
    }
}

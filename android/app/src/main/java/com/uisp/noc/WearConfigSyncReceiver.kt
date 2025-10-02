package com.uisp.noc

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log

class WearConfigSyncReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent?) {
        val action = intent?.action ?: return
        if (action in INTERESTING_ACTIONS) {
            val pendingResult = goAsync()
            try {
                WearSyncManager.syncStoredConfig(context.applicationContext)
            } catch (ex: Exception) {
                Log.w("WearSyncReceiver", "Failed to sync config for action ", ex)
            } finally {
                pendingResult.finish()
            }
        }
    }

    companion object {
        private val INTERESTING_ACTIONS = setOf(
            Intent.ACTION_BOOT_COMPLETED,
            Intent.ACTION_MY_PACKAGE_REPLACED,
            Intent.ACTION_TIME_CHANGED,
            Intent.ACTION_TIMEZONE_CHANGED
        )
    }
}

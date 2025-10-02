package com.uisp.noc

import android.content.Context
import android.util.Log
import com.google.android.gms.wearable.PutDataMapRequest
import com.google.android.gms.wearable.Wearable

object WearSyncManager {
    private const val PATH_CONFIG = "/uisp/config"
    private const val KEY_BASE_URL = "base_url"
    private const val KEY_API_TOKEN = "api_token"
    private const val KEY_UPDATED_AT = "updated_at"
    private const val PREFS_NAME = "uisp_prefs"

    fun syncStoredConfig(context: Context) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val baseUrl = prefs.getString("dashboard_url", "") ?: ""
        val token = prefs.getString("api_token", "") ?: ""
        syncConfig(context, baseUrl, token)
    }

    fun syncConfig(context: Context, baseUrl: String?, apiToken: String?) {
        try {
            val request = PutDataMapRequest.create(PATH_CONFIG).apply {
                dataMap.putString(KEY_BASE_URL, baseUrl ?: "")
                dataMap.putString(KEY_API_TOKEN, apiToken ?: "")
                dataMap.putLong(KEY_UPDATED_AT, System.currentTimeMillis())
            }.asPutDataRequest().setUrgent()
            Wearable.getDataClient(context).putDataItem(request)
        } catch (ex: Exception) {
            Log.w("WearSyncManager", "Failed to sync config to wear", ex)
        }
    }

    fun handleRequestedSync(context: Context) {
        syncStoredConfig(context)
    }

    const val PATH_REQUEST_CONFIG = "/uisp/request-config"
}

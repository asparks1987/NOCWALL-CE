package com.uisp.noc.data

import android.content.Context
import android.content.SharedPreferences
import com.uisp.noc.WearSyncManager

class SessionStore private constructor(context: Context) {

    private val prefs: SharedPreferences =
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    private val appContext = context.applicationContext

    fun load(): Session? {
        val backendUrl = prefs.getString(KEY_BACKEND_URL, null)?.takeIf { it.isNotBlank() }
        val username = prefs.getString(KEY_USERNAME, null)?.takeIf { it.isNotBlank() }
        val uispBaseUrl = prefs.getString(KEY_UISP_BASE_URL, null)?.takeIf { it.isNotBlank() }
        val token = prefs.getString(KEY_API_TOKEN, null)?.takeIf { it.isNotBlank() }
        return if (backendUrl != null && username != null && uispBaseUrl != null && token != null) {
            Session(
                backendUrl = backendUrl,
                username = username,
                uispBaseUrl = uispBaseUrl,
                uispToken = token,
                authenticatedAtMillis = prefs.getLong(KEY_AUTH_AT, System.currentTimeMillis())
            )
        } else {
            null
        }
    }

    fun save(session: Session) {
        prefs.edit()
            .putString(KEY_BACKEND_URL, session.backendUrl)
            .putString(KEY_USERNAME, session.username)
            .putString(KEY_UISP_BASE_URL, session.uispBaseUrl)
            .putString(KEY_API_TOKEN, session.uispToken)
            .putLong(KEY_AUTH_AT, session.authenticatedAtMillis)
            .apply()

        // Maintain backwards compatibility for wear sync consumers.
        WearSyncManager.syncConfig(appContext, session.uispBaseUrl, session.uispToken)
    }

    fun clear(preserveConnectionInfo: Boolean = true) {
        val editor = prefs.edit()
        if (!preserveConnectionInfo) {
            editor.remove(KEY_BACKEND_URL)
            editor.remove(KEY_USERNAME)
        }
        editor.remove(KEY_UISP_BASE_URL)
        editor.remove(KEY_API_TOKEN)
        editor.remove(KEY_AUTH_AT)
        editor.remove(KEY_PUSH_TOKEN)
        editor.apply()

        WearSyncManager.syncConfig(appContext, null, null)
    }

    fun lastBackendUrl(): String? = prefs.getString(KEY_BACKEND_URL, null)

    fun lastUsername(): String? = prefs.getString(KEY_USERNAME, null)

    fun savePushToken(token: String) {
        prefs.edit().putString(KEY_PUSH_TOKEN, token).apply()
    }

    fun pushToken(): String? = prefs.getString(KEY_PUSH_TOKEN, null)

    companion object {
        @Volatile
        private var INSTANCE: SessionStore? = null

        fun getInstance(context: Context): SessionStore {
            return INSTANCE ?: synchronized(this) {
                INSTANCE ?: SessionStore(context.applicationContext).also {
                    INSTANCE = it
                }
            }
        }

        private const val PREFS_NAME = "uisp_prefs"
        private const val KEY_BACKEND_URL = "backend_url"
        private const val KEY_USERNAME = "username"
        private const val KEY_UISP_BASE_URL = "dashboard_url"
        private const val KEY_API_TOKEN = "api_token"
        private const val KEY_AUTH_AT = "authenticated_at"
        private const val KEY_PUSH_TOKEN = "push_token"
    }
}

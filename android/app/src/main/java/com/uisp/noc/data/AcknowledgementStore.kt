package com.uisp.noc.data

import android.content.Context
import android.content.SharedPreferences
import java.util.concurrent.TimeUnit

interface AcknowledgementStore {
    fun acknowledge(deviceId: String, durationMinutes: Int)
    fun clear(deviceId: String)
    fun ackUntil(deviceId: String, nowMillis: Long = System.currentTimeMillis()): Long?
    fun snapshot(nowMillis: Long = System.currentTimeMillis()): Map<String, Long>

    class InMemory : AcknowledgementStore {
        private val backing: MutableMap<String, Long> = mutableMapOf()

        override fun acknowledge(deviceId: String, durationMinutes: Int) {
            val expiresAt = System.currentTimeMillis() + TimeUnit.MINUTES.toMillis(durationMinutes.toLong())
            backing[deviceId] = expiresAt
        }

        override fun clear(deviceId: String) {
            backing.remove(deviceId)
        }

        override fun ackUntil(deviceId: String, nowMillis: Long): Long? {
            val expires = backing[deviceId]
            if (expires != null && expires > nowMillis) {
                return expires
            }
            if (expires != null && expires <= nowMillis) {
                backing.remove(deviceId)
            }
            return null
        }

        override fun snapshot(nowMillis: Long): Map<String, Long> {
            return backing.filterValues { it > nowMillis }
        }
    }

    class Persistent private constructor(context: Context) : AcknowledgementStore {
        private val prefs: SharedPreferences =
            context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

        override fun acknowledge(deviceId: String, durationMinutes: Int) {
            val updated = loadRaw().toMutableMap()
            val expiresAt = System.currentTimeMillis() + TimeUnit.MINUTES.toMillis(durationMinutes.toLong())
            updated[deviceId] = expiresAt
            saveRaw(updated)
        }

        override fun clear(deviceId: String) {
            val updated = loadRaw().toMutableMap()
            updated.remove(deviceId)
            saveRaw(updated)
        }

        override fun ackUntil(deviceId: String, nowMillis: Long): Long? {
            val raw = loadRaw()
            val expires = raw[deviceId]
            if (expires != null && expires > nowMillis) {
                return expires
            }
            if (expires != null && expires <= nowMillis) {
                clear(deviceId)
            }
            return null
        }

        override fun snapshot(nowMillis: Long): Map<String, Long> {
            val raw = loadRaw()
            val filtered = raw.filterValues { it > nowMillis }
            if (filtered.size != raw.size) {
                saveRaw(filtered)
            }
            return filtered
        }

        private fun loadRaw(): Map<String, Long> {
            val stored = prefs.getString(PREF_KEY_ACKS, "") ?: ""
            if (stored.isBlank()) return emptyMap()
            return stored.split(ENTRY_SEPARATOR)
                .mapNotNull { entry ->
                    val parts = entry.split(KEY_VALUE_SEPARATOR)
                    if (parts.size == 2) {
                        val key = parts[0]
                        val value = parts[1].toLongOrNull()
                        if (value != null) key to value else null
                    } else {
                        null
                    }
                }
                .toMap()
        }

        private fun saveRaw(map: Map<String, Long>) {
            if (map.isEmpty()) {
                prefs.edit().remove(PREF_KEY_ACKS).apply()
                return
            }
            val serialized = map.entries.joinToString(ENTRY_SEPARATOR.toString()) { (key, value) ->
                "$key$KEY_VALUE_SEPARATOR$value"
            }
            prefs.edit().putString(PREF_KEY_ACKS, serialized).apply()
        }

        companion object {
            fun getInstance(context: Context): Persistent {
                return Persistent(context.applicationContext)
            }

            private const val PREFS_NAME = "uisp_ack_prefs"
            private const val PREF_KEY_ACKS = "acks"
            private const val ENTRY_SEPARATOR = ';'
            private const val KEY_VALUE_SEPARATOR = '|'
        }
    }
}

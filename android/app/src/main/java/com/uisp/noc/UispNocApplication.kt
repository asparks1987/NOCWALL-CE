package com.uisp.noc

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.media.AudioAttributes
import android.net.Uri
import android.os.Build

class UispNocApplication : Application() {

    override fun onCreate() {
        super.onCreate()
        createNotificationChannels()
    }

    private fun createNotificationChannels() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val notificationManager = getSystemService(NotificationManager::class.java)
            val sounds = listOf("buz", "brrt", "flrp")
            sounds.forEach { sound ->
                val soundUri = Uri.parse(
                    "android.resource://" + packageName + "/" + when (sound) {
                        "brrt" -> R.raw.brrt
                        "flrp" -> R.raw.flrp
                        else -> R.raw.buz
                    }
                )

                val channel = NotificationChannel(
                    "uisp_alerts_channel_$sound",
                    "UISP Alerts ($sound)",
                    NotificationManager.IMPORTANCE_HIGH
                ).apply {
                    description = "Channel for UISP alert notifications with $sound sound"
                    val audioAttributes = AudioAttributes.Builder()
                        .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                        .setUsage(AudioAttributes.USAGE_NOTIFICATION)
                        .build()
                    setSound(soundUri, audioAttributes)
                }
                notificationManager.createNotificationChannel(channel)
            }
        }
    }
}

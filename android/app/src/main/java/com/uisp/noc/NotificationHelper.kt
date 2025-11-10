package com.uisp.noc

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.media.AudioAttributes
import android.net.Uri
import android.os.Build
import androidx.core.app.NotificationCompat

object NotificationHelper {

    private const val CHANNEL_ID = "gateway-status"
    private const val CHANNEL_NAME = "Gateway Status"

    fun createNotificationChannel(context: Context) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val soundUri = Uri.parse("android.resource://" + context.packageName + "/" + R.raw.buz)
            val audioAttributes = AudioAttributes.Builder()
                .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                .setUsage(AudioAttributes.USAGE_NOTIFICATION)
                .build()

            val channel = NotificationChannel(
                CHANNEL_ID,
                CHANNEL_NAME,
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Notifications for offline gateways"
                setSound(soundUri, audioAttributes)
            }
            val notificationManager =
                context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            notificationManager.createNotificationChannel(channel)
        }
    }

    fun showOfflineNotification(context: Context, gatewayName: String) {
        showNotification(context, gatewayName, "$gatewayName has been offline for 1 minute.")
    }

    fun showOfflineReminderNotification(context: Context, gatewayName: String) {
        showNotification(context, gatewayName, "$gatewayName is still offline.")
    }

    private fun showNotification(context: Context, gatewayName: String, message: String) {
        val soundUri = Uri.parse("android.resource://" + context.packageName + "/" + R.raw.buz)
        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_launcher)
            .setContentTitle("Gateway Offline")
            .setContentText(message)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setSound(soundUri)
            .build()

        val notificationManager =
            context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.notify(gatewayName.hashCode(), notification)
    }
}

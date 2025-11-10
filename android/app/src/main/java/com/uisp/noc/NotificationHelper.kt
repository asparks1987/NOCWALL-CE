package com.uisp.noc

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat

object NotificationHelper {

    private const val CHANNEL_ID = "gateway_status"
    private const val FOREGROUND_CHANNEL_ID = "foreground_service"

    fun createNotificationChannel(context: Context) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val name = "Gateway Status"
            val descriptionText = "Notifications for gateway status changes"
            val importance = NotificationManager.IMPORTANCE_DEFAULT
            val channel = NotificationChannel(CHANNEL_ID, name, importance).apply {
                description = descriptionText
            }

            val foregroundName = "Background Service"
            val foregroundDescription = "Persistent notification to keep the app running"
            val foregroundImportance = NotificationManager.IMPORTANCE_LOW
            val foregroundChannel = NotificationChannel(FOREGROUND_CHANNEL_ID, foregroundName, foregroundImportance).apply {
                description = foregroundDescription
            }

            val notificationManager: NotificationManager =
                context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            notificationManager.createNotificationChannel(channel)
            notificationManager.createNotificationChannel(foregroundChannel)
        }
    }

    fun createForegroundNotification(
        context: Context,
        onlineGateways: Int,
        offlineGateways: Int,
        onlineSwitches: Int,
        offlineSwitches: Int,
        onlineRouters: Int,
        offlineRouters: Int
    ): Notification {
        val stopSelfIntent = Intent(context, GatewayStatusService::class.java).apply {
            action = GatewayStatusService.ACTION_STOP
        }
        val pendingIntent = PendingIntent.getService(
            context, 0, stopSelfIntent, PendingIntent.FLAG_CANCEL_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val contentText = "Gateways: $onlineGateways online, $offlineGateways offline\n" +
                "Switches: $onlineSwitches online, $offlineSwitches offline\n" +
                "Routers: $onlineRouters online, $offlineRouters offline"

        return NotificationCompat.Builder(context, FOREGROUND_CHANNEL_ID)
            .setContentTitle("UISP NOC is running")
            .setContentText(contentText)
            .setStyle(NotificationCompat.BigTextStyle().bigText(contentText))
            .setSmallIcon(R.drawable.ic_launcher) // Replace with a real icon
            .addAction(R.drawable.ic_launcher, "Stop", pendingIntent) // Replace with a real icon
            .build()
    }

    fun showOfflineNotification(context: Context, gatewayName: String) {
        val builder = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_launcher) // Replace with a real icon
            .setContentTitle("Gateway Offline")
            .setContentText("$gatewayName has been offline for over 1 minute.")
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)

        with(NotificationManagerCompat.from(context)) {
            // notificationId is a unique int for each notification that you must define
            notify(gatewayName.hashCode(), builder.build())
        }
    }

    fun showOfflineReminderNotification(context: Context, gatewayName: String) {
        val builder = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_launcher) // Replace with a real icon
            .setContentTitle("Gateway Still Offline")
            .setContentText("$gatewayName is still offline.")
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)

        with(NotificationManagerCompat.from(context)) {
            notify(gatewayName.hashCode(), builder.build())
        }
    }
}

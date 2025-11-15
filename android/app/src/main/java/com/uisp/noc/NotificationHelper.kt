package com.uisp.noc

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.Typeface
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.graphics.drawable.IconCompat

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

        val totalOnlineBackbone = onlineGateways + onlineSwitches + onlineRouters
        val totalOfflineBackbone = offlineGateways + offlineSwitches + offlineRouters

        val contentText = "Backbone: $totalOnlineBackbone online, $totalOfflineBackbone offline\n" +
                "Gateways: $onlineGateways online, $offlineGateways offline\n" +
                "Switches: $onlineSwitches online, $offlineSwitches offline\n" +
                "Routers: $onlineRouters online, $offlineRouters offline"

        val statusIcon = createStatusIcon(context, totalOnlineBackbone, totalOfflineBackbone)

        return NotificationCompat.Builder(context, FOREGROUND_CHANNEL_ID)
            .setContentTitle("UISP NOC is running")
            .setContentText(contentText)
            .setStyle(NotificationCompat.BigTextStyle().bigText(contentText))
            .setSmallIcon(statusIcon)
            .addAction(R.drawable.ic_launcher, "Stop", pendingIntent) // Replace with a real icon
            .build()
    }

    private fun createStatusIcon(context: Context, onlineCount: Int, offlineCount: Int): IconCompat {
        val bitmap = Bitmap.createBitmap(128, 128, Bitmap.Config.ARGB_8888)
        val canvas = Canvas(bitmap)
        val paint = Paint(Paint.ANTI_ALIAS_FLAG)

        // Text color should be white for status bar icons
        paint.color = Color.WHITE
        paint.textSize = 80f // Make it larger to be more visible
        paint.textAlign = Paint.Align.CENTER
        paint.typeface = Typeface.create(Typeface.DEFAULT, Typeface.BOLD)

        val text = if (offlineCount > 0) {
            paint.style = Paint.Style.STROKE
            paint.strokeWidth = 4f
            paint.color = Color.RED
            "$offlineCount"
        } else {
            "$onlineCount"
        }

        val xPos = canvas.width / 2f
        val yPos = (canvas.height / 2f) - ((paint.descent() + paint.ascent()) / 2f)

        canvas.drawText(text, xPos, yPos, paint)

        if(offlineCount == 0){
            paint.style = Paint.Style.FILL
            paint.color = Color.WHITE
            canvas.drawText(text, xPos, yPos, paint)
        }


        return IconCompat.createWithBitmap(bitmap)
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

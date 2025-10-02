package com.uisp.noc

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.media.AudioAttributes
import android.net.Uri
import android.os.Build
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage

class MyFirebaseMessagingService : FirebaseMessagingService() {

    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        super.onMessageReceived(remoteMessage)

        // Get notification data
        val title = remoteMessage.notification?.title ?: "New Alert"
        val body = remoteMessage.notification?.body ?: "You have a new alert."

        // Show the notification
        sendNotification(title, body)
    }

    private fun sendNotification(title: String, messageBody: String) {
        val channelId = "uisp_alerts_channel"
        // IMPORTANT: The sound file must be placed in `res/raw/alert_sound.mp3`
        val soundUri = Uri.parse("android.resource://" + applicationContext.packageName + "/" + R.raw.alert_sound)

        val notificationBuilder = NotificationCompat.Builder(this, channelId)
            .setSmallIcon(R.drawable.ic_launcher) // Make sure you have this drawable
            .setContentTitle(title)
            .setContentText(messageBody)
            .setAutoCancel(true)
            .setSound(soundUri)
            .setPriority(NotificationCompat.PRIORITY_HIGH)

        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

        // Since android Oreo notification channel is needed.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                channelId,
                "UISP Alerts",
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Channel for UISP alert notifications"
                val audioAttributes = AudioAttributes.Builder()
                    .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                    .setUsage(AudioAttributes.USAGE_NOTIFICATION)
                    .build()
                setSound(soundUri, audioAttributes)
            }
            notificationManager.createNotificationChannel(channel)
        }

        notificationManager.notify(0 /* ID of notification */, notificationBuilder.build())
    }

    override fun onNewToken(token: String) {
        super.onNewToken(token)
        // You would typically send this token to your server to be able to send messages to this device
        // Log.d("FCM_TOKEN", "Refreshed token: $token")
    }
}

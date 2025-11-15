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
        val title: String
        val body: String
        val sound: String

        // Check if the message contains a data payload.
        if (remoteMessage.data.isNotEmpty()) {
            title = remoteMessage.data["title"] ?: "New Alert"
            body = remoteMessage.data["body"] ?: "You have a new alert."
            sound = remoteMessage.data["sound"] ?: "buz"
        } else {
            // Check if the message contains a notification payload.
            title = remoteMessage.notification?.title ?: "New Alert"
            body = remoteMessage.notification?.body ?: "You have a new alert."
            sound = remoteMessage.notification?.sound ?: "buz"
        }


        // Show the notification
        sendNotification(this, title, body, sound)
    }

    override fun onNewToken(token: String) {
        super.onNewToken(token)
        // You would typically send this token to your server to be able to send messages to this device
        // Log.d("FCM_TOKEN", "Refreshed token: $token")
    }

    companion object {
        fun sendNotification(context: Context, title: String, messageBody: String, sound: String = "buz") {
            val channelId = "uisp_alerts_channel_$sound"
            val soundUri =
                Uri.parse("android.resource://" + context.packageName + "/" + when (sound) {
                    "brrt" -> R.raw.brrt
                    "flrp" -> R.raw.flrp
                    else -> R.raw.buz
                })

            val notificationBuilder = NotificationCompat.Builder(context, channelId)
                .setSmallIcon(R.drawable.ic_launcher) // Make sure you have this drawable
                .setContentTitle(title)
                .setContentText(messageBody)
                .setAutoCancel(true)
                .setSound(soundUri)
                .setPriority(NotificationCompat.PRIORITY_HIGH)

            val notificationManager =
                context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

            notificationManager.notify(System.currentTimeMillis().toInt(), notificationBuilder.build())
        }
    }
}

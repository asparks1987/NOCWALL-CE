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

        val title: String
        val body: String
        var sound: String

        if (remoteMessage.data.isNotEmpty()) {
            title = remoteMessage.data["title"] ?: "New Alert"
            body = remoteMessage.data["body"] ?: "You have a new alert."
            sound = remoteMessage.data["sound"] ?: "buz"
        } else {
            title = remoteMessage.notification?.title ?: "New Alert"
            body = remoteMessage.notification?.body ?: "You have a new alert."
            sound = remoteMessage.notification?.sound ?: "buz"
        }

        if (title.contains("online", ignoreCase = true)) {
            sound = "default_online" // Use a new unique identifier for the sound type
        }

        sendNotification(this, title, body, sound)
    }

    override fun onNewToken(token: String) {
        super.onNewToken(token)
        // Persist token so UI can register with backend when available
        SessionStore.getInstance(applicationContext).savePushToken(token)
    }

    companion object {
        fun sendNotification(context: Context, title: String, messageBody: String, sound: String = "buz") {
            val notificationManager =
                context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

            // Use a unique channel ID for each sound type to avoid conflicts with stale channels
            val channelId = "uisp_alerts_channel_$sound"

            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                // Determine channel name based on sound
                val channelName = when (sound) {
                    "default_online" -> "UISP Alerts (Online)"
                    "buz" -> "UISP Alerts (Offline)"
                    else -> "UISP Alerts ($sound)"
                }

                val channel = NotificationChannel(channelId, channelName, NotificationManager.IMPORTANCE_HIGH)

                if (sound != "default_online") {
                    val soundResId = when (sound) {
                        "brrt" -> R.raw.brrt
                        "flrp" -> R.raw.flrp
                        else -> R.raw.buz
                    }
                    val soundUri = Uri.parse("android.resource://${context.packageName}/$soundResId")
                    val audioAttributes = AudioAttributes.Builder()
                        .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                        .setUsage(AudioAttributes.USAGE_NOTIFICATION)
                        .build()
                    channel.setSound(soundUri, audioAttributes)
                }
                // For the "default_online" channel, we don't set a sound URI, so the system default is used.
                notificationManager.createNotificationChannel(channel)
            }

            val notificationBuilder = NotificationCompat.Builder(context, channelId)
                .setSmallIcon(R.drawable.ic_launcher)
                .setContentTitle(title)
                .setContentText(messageBody)
                .setAutoCancel(true)
                .setPriority(NotificationCompat.PRIORITY_HIGH)

            // For older Android versions, set the sound directly on the builder
            if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
                if (sound == "default_online") {
                    notificationBuilder.setDefaults(NotificationCompat.DEFAULT_SOUND)
                } else {
                    val soundResId = when (sound) {
                        "brrt" -> R.raw.brrt
                        "flrp" -> R.raw.flrp
                        else -> R.raw.buz
                    }
                    val soundUri = Uri.parse("android.resource://${context.packageName}/$soundResId")
                    notificationBuilder.setSound(soundUri)
                }
            }

            notificationManager.notify(System.currentTimeMillis().toInt(), notificationBuilder.build())
        }
    }
}

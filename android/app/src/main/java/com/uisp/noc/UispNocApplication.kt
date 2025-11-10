package com.uisp.noc

import android.app.Application

class UispNocApplication : Application() {

    override fun onCreate() {
        super.onCreate()
        NotificationHelper.createNotificationChannel(this)
        OfflineGatewayMonitor.start(this)
    }
}

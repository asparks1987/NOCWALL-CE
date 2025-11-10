package com.uisp.noc

import android.app.Application

class UispNocApplication : Application() {

    override fun onCreate() {
        super.onCreate()
        // Initialization that needs to happen on app startup can go here.
        // We have moved the notification channel creation and worker scheduling
        // to MainActivity to ensure they happen at the right time.
    }
}

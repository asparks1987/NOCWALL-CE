package com.uisp.noc

import com.google.android.gms.wearable.MessageEvent
import com.google.android.gms.wearable.WearableListenerService

class WearConfigListenerService : WearableListenerService() {
    override fun onMessageReceived(messageEvent: MessageEvent) {
        if (messageEvent.path == WearSyncManager.PATH_REQUEST_CONFIG) {
            WearSyncManager.handleRequestedSync(this)
        } else {
            super.onMessageReceived(messageEvent)
        }
    }
}

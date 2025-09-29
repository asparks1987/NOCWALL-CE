Android wrapper app (WebView)

This directory contains a minimal Android app that wraps the UISP NOC dashboard in a WebView. It's intended as a quick mobile wrapper to view the dashboard from an Android device or emulator.

Quick start (Android Studio)
- Open this folder (`android`) in Android Studio.
- Let Gradle sync. You may need to install the Kotlin and Android Gradle plugins as prompted.
- Run on an emulator or device.

Notes
- Default URL loaded by the app is http://10.0.2.2/ which maps to the host machine when using the Android emulator. Change the default in `MainActivity.kt` or pass a URL via an intent extra named "url".
- The app enables JavaScript and DOM storage in the WebView.
- This is intentionally minimal for quick testing; for production you should add:
  - TLS/HTTPS enforcement
  - Content-Security-Policy or other hardening
  - Error pages and offline handling
  - WebView process isolation and bridging carefully if needed

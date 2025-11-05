# Android Wrapper (WebView)

This module provides a minimal Android application that wraps the UISP NOC dashboard inside a WebView. It is intended for technicians who want a dedicated launcher on handheld devices without installing a full browser bookmark or PWA.

---

## Project Highlights

* Kotlin + AndroidX single-activity project (`MainActivity.kt`).
* WebView with JavaScript, DOM storage, zoom controls, and media playback enabled to support the siren and charts.
* Default URL of `http://10.0.2.2/` for emulator testing (maps to the host machine running Docker).
* Accepts an explicit URL via an Intent extra named `url` for deep linking.
* Calls `https://<noc-host>/?ajax=mobile_config` to retrieve the UISP base URL and token when hosted alongside the server (requires the same network and configured token on the server).

---

## Getting Started (Android Studio)

1. Open the `android/` directory in Android Studio.
2. Allow Gradle to sync; install the Kotlin and Android Gradle plugins if prompted.
3. Update the default dashboard URL if needed:
   * Edit `app/src/main/java/com/example/uispnoc/MainActivity.kt` and change `DEFAULT_URL`.
   * Or launch the app with `adb shell am start -n com.example.uispnoc/.MainActivity --es url https://noc.example.com`.
4. Run on an emulator or physical device running Android 8.0+.

The WebView caches cookies/local storage, so the UISP NOC session persists between launches. To reset, clear the app data from Android settings.

---

## Production Hardening Tips

* Enforce HTTPS-only URLs before shipping to devices.
* Add a splash/loading indicator and error screens for offline scenarios.
* Restrict WebView debugging (`setWebContentsDebuggingEnabled(false)`) in production builds.
* Consider implementing biometrics or device management if tablets/phones are shared among technicians.

---

## Building Release APKs

```bash
./gradlew assembleRelease
```

Configure signing configs (`app/build.gradle`) or use Android Studio’s **Build > Generate Signed Bundle / APK** workflow. The wrapper does not include Play Store metadata; add icons, branding, and store assets if you plan to publish it publicly.

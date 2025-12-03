# Android Companion App (WebView)

This module provides a minimal Android application that wraps the UISP NOC dashboard inside a WebView. It is intended for technicians who want a dedicated launcher on handheld devices without installing a full browser bookmark or PWA. Think of it as the officially blessed companion: it mirrors the siren/vibration from Gotify alerts, stores sessions between shifts, and drops techs directly into gateway/AP outage grids the moment they unlock their phone.

---

## Project Highlights

* Kotlin + AndroidX single-activity project (`MainActivity.kt`).
* WebView with JavaScript, DOM storage, zoom controls, and media playback enabled to support the siren and charts (limited to gateway/AP alerts).
* Default URL of `http://10.0.2.2/` for emulator testing (maps to the host machine running Docker).
* Accepts an explicit URL via an Intent extra named `url` for deep linking.
* Mirrors the desktop layout (Gateways, APs, Routers/Switches); stations/CPEs are not shown, and siren triggers only for gateway/AP outages with latencies streaming in as the dashboard polls.
* Connects directly to your UISP instance using its base URL and an API token; no desktop companion required.
* Allows acknowledging offline gateways/APs from the mobile dashboard so noisy alerts can be muted on the go.
* Launches full-screen with no browser chrome so the gateway/AP/routers-switches grids feel like a native console.
* Mirrors the desktop siren/vibration cues so Gotify-driven alerts stay loud even when the device is pocketed.

---

## Getting Started (Android Studio)

1. Open the `android/` directory in Android Studio.
2. Allow Gradle to sync; install the Kotlin and Android Gradle plugins if prompted.
3. Update the default dashboard URL if needed:
   * Edit `app/src/main/java/com/example/uispnoc/MainActivity.kt` and change `DEFAULT_URL`.
   * Or launch the app with `adb shell am start -n com.example.uispnoc/.MainActivity --es url https://noc.example.com`.
4. Run on an emulator or physical device running Android 8.0+.

The WebView caches cookies/local storage, so the UISP NOC session persists between launches. To reset, clear the app data from Android settings.

## Field Workflow Ideas

* Pair the APK with Gotify push notifications so tapping an alert jumps straight back into the outage dashboard.
* Ship one build per customer/site by pre-setting `DEFAULT_URL` and branding assets, then push through your MDM.
* Use Android's `Work Profile` to keep technician personal data separate while still letting them silence/acknowledge alerts from the home screen widget of your launcher choice.

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



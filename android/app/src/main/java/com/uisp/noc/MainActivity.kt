package com.uisp.noc

import android.annotation.SuppressLint
import android.content.ClipData
import android.content.ClipboardManager
import android.os.Bundle
import android.view.View
import android.widget.Button
import android.widget.EditText
import android.webkit.WebChromeClient
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.activity.OnBackPressedCallback
import androidx.appcompat.app.AppCompatActivity
import android.widget.Toast

class MainActivity : AppCompatActivity() {
    private lateinit var webView: WebView

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        webView = findViewById(R.id.webview)
        val settings: WebSettings = webView.settings
        settings.javaScriptEnabled = true
        settings.domStorageEnabled = true
        settings.useWideViewPort = true
        settings.loadWithOverviewMode = true

        webView.webViewClient = WebViewClient()
        webView.webChromeClient = WebChromeClient()

        // Default URL for now (user-editable). The user asked for 192.168.1.5:1200
        val defaultUrl = "http://192.168.1.5:1200/"

        val prefs = getSharedPreferences("uisp_prefs", MODE_PRIVATE)
        val prompt = findViewById<View>(R.id.url_prompt)
        val urlInput = findViewById<EditText>(R.id.url_input)
        val tokenInput = findViewById<EditText>(R.id.token_input)
        val btnLoad = findViewById<Button>(R.id.btn_load)
        val btnPaste = findViewById<Button>(R.id.btn_paste_token)
        val btnSettings = findViewById<Button>(R.id.btn_open_settings)

        val savedUrl = prefs.getString("dashboard_url", null)
        val savedToken = prefs.getString("api_token", "")

        fun hidePrompt() {
            prompt.visibility = View.GONE
            btnSettings.visibility = View.VISIBLE
        }

        fun showPrompt(prefillDefaults: Boolean) {
            if (prefillDefaults) {
                val latestUrl = prefs.getString("dashboard_url", null) ?: defaultUrl
                val latestToken = prefs.getString("api_token", "") ?: ""
                urlInput.setText(latestUrl)
                tokenInput.setText(latestToken)
            }
            prompt.visibility = View.VISIBLE
            btnSettings.visibility = View.GONE
        }

        fun persistAndSync(url: String, token: String) {
            prefs.edit()
                .putString("dashboard_url", url)
                .putString("api_token", token)
                .apply()
            WearSyncManager.syncStoredConfig(this)
        }

        btnSettings.setOnClickListener {
            showPrompt(prefillDefaults = true)
        }

        btnPaste.setOnClickListener {
            val clipboard = getSystemService(CLIPBOARD_SERVICE) as? ClipboardManager
            val clipText = clipboard?.primaryClip?.takeIf { it.itemCount > 0 }
                ?.getItemAt(0)
                ?.coerceToText(this)
                ?.toString()
                ?.trim()
            if (!clipText.isNullOrEmpty()) {
                tokenInput.setText(clipText)
                tokenInput.setSelection(clipText.length)
                Toast.makeText(this, "Token pasted from clipboard", Toast.LENGTH_SHORT).show()
            } else {
                Toast.makeText(this, "Clipboard is empty", Toast.LENGTH_SHORT).show()
            }
        }

        btnLoad.setOnClickListener {
            var url = urlInput.text.toString().trim()
            val token = tokenInput.text.toString().trim()
            if (!url.startsWith("http://") && !url.startsWith("https://")) {
                url = "http://$url"
            }
            persistAndSync(url, token)
            hidePrompt()
            webView.loadUrl(url)
        }

        if (savedUrl.isNullOrBlank()) {
            showPrompt(prefillDefaults = true)
        } else {
            hidePrompt()
            urlInput.setText(savedUrl)
            tokenInput.setText(savedToken ?: "")
            webView.loadUrl(savedUrl)
            WearSyncManager.syncStoredConfig(this)
        }
        onBackPressedDispatcher.addCallback(this, object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                if (webView.canGoBack()) {
                    webView.goBack()
                } else {
                    isEnabled = false
                    onBackPressedDispatcher.onBackPressed()
                }
            }
        })

        if (!savedUrl.isNullOrBlank()) {
            btnSettings.visibility = View.VISIBLE
        }
    }

    override fun onResume() {
        super.onResume()
        WearSyncManager.syncStoredConfig(this)
    }
}

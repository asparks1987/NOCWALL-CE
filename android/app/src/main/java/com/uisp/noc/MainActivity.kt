package com.uisp.noc

import android.annotation.SuppressLint
import android.os.Bundle
import android.webkit.WebChromeClient
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.appcompat.app.AppCompatActivity

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
        val savedUrl = prefs.getString("dashboard_url", null)
        if (savedUrl != null) {
            webView.loadUrl(savedUrl)
            findViewById<android.view.View>(R.id.url_prompt).visibility = android.view.View.GONE
        } else {
            // Show prompt overlay
            val prompt = findViewById<android.view.View>(R.id.url_prompt)
            val input = findViewById<android.widget.EditText>(R.id.url_input)
            val btn = findViewById<android.widget.Button>(R.id.btn_load)
            input.setText(defaultUrl)
            prompt.visibility = android.view.View.VISIBLE
            btn.setOnClickListener {
                var u = input.text.toString().trim()
                if (!u.startsWith("http://") && !u.startsWith("https://")) {
                    u = "http://$u"
                }
                prefs.edit().putString("dashboard_url", u).apply()
                prompt.visibility = android.view.View.GONE
                webView.loadUrl(u)
            }
        }
    }

    override fun onBackPressed() {
        if (webView.canGoBack()) webView.goBack() else super.onBackPressed()
    }
}

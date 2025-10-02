package com.uisp.noc.wear

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.uisp.noc.wear.ui.WearDashboardScreen
import com.uisp.noc.wear.ui.theme.UISPNOCWearTheme

class MainActivity : ComponentActivity() {
    private val viewModel: MainViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            UISPNOCWearTheme {
                val uiState by viewModel.uiState.collectAsStateWithLifecycle()

                WearDashboardScreen(
                    uiState = uiState,
                    onRefresh = { viewModel.refresh(force = true) },
                    onRequestConfig = { viewModel.requestConfigFromPhone() }
                )
            }
        }
    }
}

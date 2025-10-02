package com.uisp.noc.wear.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.wear.compose.foundation.lazy.ScalingLazyColumn
import androidx.wear.compose.foundation.lazy.ScalingLazyListState
import androidx.wear.compose.foundation.lazy.items
import androidx.wear.compose.foundation.lazy.rememberScalingLazyListState
import androidx.wear.compose.material.Card
import androidx.wear.compose.material.Chip
import androidx.wear.compose.material.ChipDefaults
import androidx.wear.compose.material.MaterialTheme
import androidx.wear.compose.material.PositionIndicator
import androidx.wear.compose.material.Scaffold
import androidx.wear.compose.material.Text
import androidx.wear.compose.material.TimeText
import androidx.wear.compose.material.Vignette
import androidx.wear.compose.material.VignettePosition
import com.uisp.noc.wear.WearUiState
import com.uisp.noc.wear.data.DeviceStatus
import com.uisp.noc.wear.data.DevicesSummary
import java.lang.StringBuilder
import java.text.DecimalFormat
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

@Composable
fun WearDashboardScreen(
    uiState: WearUiState,
    onRefresh: () -> Unit,
    onRequestConfig: () -> Unit
) {
    val listState = rememberScalingLazyListState()

    Scaffold(
        timeText = { TimeText() },
        vignette = { Vignette(vignettePosition = VignettePosition.TopAndBottom) },
        positionIndicator = { PositionIndicator(scalingLazyListState = listState) }
    ) {
        when (uiState) {
            is WearUiState.Loading -> LoadingState(onRefresh)
            is WearUiState.MissingConfig -> MissingConfigState(uiState.message, onRequestConfig)
            is WearUiState.Error -> ErrorState(uiState.message, uiState.lastSummary, onRefresh, onRequestConfig, listState)
            is WearUiState.Success -> SuccessState(uiState.summary, listState, onRefresh, onRequestConfig)
        }
    }
}

@Composable
private fun LoadingState(onRefresh: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 12.dp, vertical = 24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Text(text = "Loading…", fontSize = 14.sp)
        Spacer(modifier = Modifier.height(12.dp))
        Chip(onClick = onRefresh, label = { Text(text = "Refresh") })
    }
}

@Composable
private fun MissingConfigState(message: String, onRequestConfig: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 12.dp, vertical = 24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Text(text = "Waiting for phone", fontSize = 16.sp, fontWeight = FontWeight.SemiBold)
        Spacer(modifier = Modifier.height(6.dp))
        Text(text = message, fontSize = 12.sp)
        Spacer(modifier = Modifier.height(12.dp))
        Chip(onClick = onRequestConfig, label = { Text("Request Sync") })
    }
}

@Composable
private fun ErrorState(
    message: String,
    lastSummary: DevicesSummary?,
    onRefresh: () -> Unit,
    onRequestConfig: () -> Unit,
    listState: ScalingLazyListState
) {
    ScalingLazyColumn(
        state = listState,
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 12.dp, vertical = 6.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        item {
            Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.fillMaxWidth()) {
                Text(text = "Unable to refresh", fontSize = 14.sp, fontWeight = FontWeight.SemiBold)
                Spacer(modifier = Modifier.height(4.dp))
                Text(text = message, fontSize = 12.sp)
            }
        }
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Chip(onClick = onRefresh, label = { Text("Retry") })
                Chip(onClick = onRequestConfig, label = { Text("Sync Phone") })
            }
        }
        lastSummary?.let { summary ->
            item { SummaryCard(summary) }
        }
    }
}

@Composable
private fun SuccessState(
    summary: DevicesSummary,
    listState: ScalingLazyListState,
    onRefresh: () -> Unit,
    onRequestConfig: () -> Unit
) {
    ScalingLazyColumn(
        state = listState,
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 12.dp, vertical = 6.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Chip(onClick = onRefresh, label = { Text("Refresh") })
                Chip(onClick = onRequestConfig, label = { Text("Sync Phone") })
            }
        }
        item { SummaryCard(summary) }
        if (summary.highLatencyGateways.isNotEmpty()) {
            item { SectionHeader(text = "Latency Alerts") }
            items(summary.highLatencyGateways) { DeviceChip(it, highlightLatency = true) }
        }
        if (summary.offlineGateways.isNotEmpty()) {
            item { SectionHeader(text = "Gateways Offline") }
            items(summary.offlineGateways) { DeviceChip(it) }
        }
        if (summary.offlineBackbone.isNotEmpty()) {
            item { SectionHeader(text = "Backbone Offline") }
            items(summary.offlineBackbone) { DeviceChip(it) }
        }
        if (summary.offlineCpes.isNotEmpty()) {
            item { SectionHeader(text = "CPEs Offline") }
            items(summary.offlineCpes.take(8)) { DeviceChip(it) }
        }
        if (summary.offlineGateways.isEmpty() && summary.offlineBackbone.isEmpty() && summary.offlineCpes.isEmpty()) {
            item {
                Text(
                    text = "All monitored devices are online.",
                    fontSize = 14.sp,
                    modifier = Modifier.padding(vertical = 8.dp)
                )
            }
        }
    }
}

@Composable
private fun SummaryCard(summary: DevicesSummary) {
    Card(onClick = { }) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(text = "Network Status", fontSize = 14.sp, fontWeight = FontWeight.SemiBold)
            Spacer(modifier = Modifier.height(6.dp))
            Row(horizontalArrangement = Arrangement.SpaceBetween, modifier = Modifier.fillMaxWidth()) {
                Column {
                    Text(text = "Gateways", fontSize = 12.sp)
                    Text(text = formatCount(summary.totalGateways, summary.offlineGateways.size))
                }
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(text = "Backbone", fontSize = 12.sp)
                    Text(text = formatCount(summary.totalBackbone, summary.offlineBackbone.size))
                }
                Column(horizontalAlignment = Alignment.End) {
                    Text(text = "CPEs", fontSize = 12.sp)
                    Text(text = formatCount(summary.totalCpes, summary.offlineCpes.size))
                }
            }
            Spacer(modifier = Modifier.height(6.dp))
            Text(text = "Health: " + summary.healthPercent + "%", fontSize = 12.sp)
            Text(
                text = "Updated " + formatTimestamp(summary.lastUpdatedEpochMillis),
                fontSize = 10.sp
            )
        }
    }
}

@Composable
private fun SectionHeader(text: String) {
    Text(text = text, fontSize = 12.sp, fontWeight = FontWeight.Bold)
}

@Composable
private fun DeviceChip(device: DeviceStatus, highlightLatency: Boolean = false) {
    val latencyText = device.latencyMs?.let { DecimalFormat("#.0").format(it) + " ms" }
    Chip(
        onClick = {},
        label = {
            Column {
                Text(text = device.name, maxLines = 1)
                val subtitle = StringBuilder().apply {
                    append(device.role.replaceFirstChar { ch -> ch.titlecase(Locale.getDefault()) })
                    if (!device.online) {
                        append(" • offline")
                    }
                    if (latencyText != null) {
                        append(" • ")
                        append(latencyText)
                    }
                }.toString()
                Text(text = subtitle, fontSize = 10.sp)
            }
        },
        colors = if (highlightLatency) {
            ChipDefaults.primaryChipColors(backgroundColor = MaterialTheme.colors.secondary)
        } else {
            ChipDefaults.primaryChipColors()
        }
    )
}

private fun formatCount(total: Int, offline: Int): String = total.toString() + " (" + offline + " offline)"

private fun formatTimestamp(epochMillis: Long): String {
    val formatter = SimpleDateFormat("HH:mm", Locale.getDefault())
    return formatter.format(Date(epochMillis))
}

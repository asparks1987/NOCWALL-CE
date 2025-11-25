package com.uisp.noc.ui

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.annotation.ColorRes
import androidx.core.content.ContextCompat
import androidx.core.view.isVisible
import androidx.fragment.app.Fragment
import androidx.fragment.app.activityViewModels
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import com.google.android.material.card.MaterialCardView
import com.uisp.noc.R
import com.uisp.noc.data.Session
import com.uisp.noc.data.model.DeviceStatus
import kotlinx.coroutines.launch
import java.text.DateFormat
import java.util.Date
import kotlin.math.roundToInt

class DashboardFragment : Fragment() {

    private val viewModel: MainViewModel by activityViewModels()

    private lateinit var swipeRefresh: SwipeRefreshLayout
    private lateinit var connectionSummary: TextView
    private lateinit var lastUpdated: TextView
    private lateinit var overviewGatewaysValue: TextView
    private lateinit var overviewGatewaysLabel: TextView
    private lateinit var overviewApsValue: TextView
    private lateinit var overviewApsLabel: TextView
    private lateinit var overviewBackboneValue: TextView
    private lateinit var overviewBackboneLabel: TextView
    private lateinit var listOfflineGateways: LinearLayout
    private lateinit var listOfflineAps: LinearLayout
    private lateinit var listOfflineBackbone: LinearLayout
    private lateinit var listLatency: LinearLayout
    private lateinit var errorCard: MaterialCardView
    private lateinit var errorText: TextView
    private lateinit var testOutageButton: Button
    private lateinit var clearOutageButton: Button
    private lateinit var historyButton: Button

    private var layoutInflaterRef: LayoutInflater? = null
    private var currentSession: Session? = null

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        return inflater.inflate(R.layout.fragment_dashboard, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        layoutInflaterRef = layoutInflater
        bindViews(view)
        swipeRefresh.setOnRefreshListener {
            viewModel.refreshSummary()
        }
        testOutageButton.setOnClickListener {
            viewModel.simulateGatewayOutage()
        }
        clearOutageButton.setOnClickListener {
            viewModel.clearSimulatedGatewayOutage()
        }
        historyButton.setOnClickListener {
            Toast.makeText(requireContext(), "History coming soon!", Toast.LENGTH_SHORT).show()
        }
        collectState()
    }

    private fun bindViews(root: View) {
        swipeRefresh = root.findViewById(R.id.swipe_refresh)
        val accentColor = ContextCompat.getColor(root.context, R.color.uisp_accent_primary)
        val surfaceColor = ContextCompat.getColor(root.context, R.color.uisp_surface_variant)
        swipeRefresh.setColorSchemeColors(accentColor)
        swipeRefresh.setProgressBackgroundColorSchemeColor(surfaceColor)

        connectionSummary = root.findViewById(R.id.text_connection_summary)
        lastUpdated = root.findViewById(R.id.text_last_updated)
        testOutageButton = root.findViewById(R.id.button_test_outage)
        clearOutageButton = root.findViewById(R.id.button_clear_outage)
        historyButton = root.findViewById(R.id.button_history)

        val gatewaysInclude = root.findViewById<View>(R.id.overview_gateways)
        overviewGatewaysValue = gatewaysInclude.findViewById(R.id.text_value)
        overviewGatewaysLabel = gatewaysInclude.findViewById(R.id.text_label)

        val apInclude = root.findViewById<View>(R.id.overview_aps)
        overviewApsValue = apInclude.findViewById(R.id.text_value)
        overviewApsLabel = apInclude.findViewById(R.id.text_label)

        val backboneInclude = root.findViewById<View>(R.id.overview_backbone)
        overviewBackboneValue = backboneInclude.findViewById(R.id.text_value)
        overviewBackboneLabel = backboneInclude.findViewById(R.id.text_label)

        listOfflineGateways = root.findViewById(R.id.list_offline_gateways)
        listOfflineAps = root.findViewById(R.id.list_offline_aps)
        listOfflineBackbone = root.findViewById(R.id.list_offline_backbone)
        listLatency = root.findViewById(R.id.list_latency)

        errorCard = root.findViewById(R.id.card_error)
        errorText = root.findViewById(R.id.text_error)
    }

    private fun collectState() {
        viewLifecycleOwner.lifecycleScope.launch {
            viewLifecycleOwner.repeatOnLifecycle(Lifecycle.State.STARTED) {
                launch {
                    viewModel.dashboardState.collect { state ->
                        swipeRefresh.isRefreshing = state.isLoading
                        renderSummary(state)
                    }
                }
                launch {
                    viewModel.sessionState.collect { state ->
                        when (state) {
                            is MainViewModel.SessionState.Authenticated -> {
                                currentSession = state.session
                                val host = state.session.uispBaseUrl.substringAfter("://")
                                connectionSummary.text =
                                    "Connected to $host\nSigned in as ${state.session.username}"
                                setConnectionSummaryColor(R.color.uisp_text_primary)
                            }
                            is MainViewModel.SessionState.Unauthenticated -> {
                                currentSession = null
                                connectionSummary.text = "Not connected"
                                setConnectionSummaryColor(R.color.uisp_status_warn_fg)
                            }
                            is MainViewModel.SessionState.Loading -> {
                                connectionSummary.text = "Connecting..."
                                setConnectionSummaryColor(R.color.uisp_text_secondary)
                            }
                        }
                    }
                }
                launch {
                    viewModel.isHistoryAvailable.collect { isAvailable ->
                        historyButton.isVisible = isAvailable
                    }
                }
            }
        }
    }

    private fun renderSummary(state: MainViewModel.DashboardState) {
        val summary = state.summary
        if (summary == null) {
            lastUpdated.text = "Waiting for data..."
            overviewGatewaysValue.text = "-"
            overviewGatewaysLabel.text = "Gateways"
            overviewApsValue.text = "-"
            overviewApsLabel.text = "APs"
            overviewBackboneValue.text = "-"
            overviewBackboneLabel.text = "Routers/Switches"
            setOverviewState(overviewGatewaysValue, overviewGatewaysLabel, hasIssue = false)
            setOverviewState(overviewApsValue, overviewApsLabel, hasIssue = false)
            setOverviewState(overviewBackboneValue, overviewBackboneLabel, hasIssue = false)
            updateDeviceList(
                listOfflineGateways,
                emptyList(),
                "No gateway data yet."
            )
            updateDeviceList(
                listOfflineAps,
                emptyList(),
                "No AP data yet."
            )
            updateDeviceList(
                listOfflineBackbone,
                emptyList(),
                "No backbone data yet."
            )
            updateDeviceList(
                listLatency,
                emptyList(),
                "No latency data yet."
            )
        } else {
            val formatter = DateFormat.getDateTimeInstance(DateFormat.SHORT, DateFormat.SHORT)
            lastUpdated.text = "Last updated ${formatter.format(Date(summary.lastUpdatedEpochMillis))}"

            overviewGatewaysValue.text = summary.offlineGateways.size.takeIf { it > 0 }
                ?.let { "$it / ${summary.gateways.size}" }
                ?: summary.gateways.size.toString()
            overviewGatewaysLabel.text =
                if (summary.offlineGateways.isEmpty()) "Gateways online" else "Gateways offline"
            val gatewaysIssue = summary.offlineGateways.isNotEmpty()
            setOverviewState(overviewGatewaysValue, overviewGatewaysLabel, gatewaysIssue)

            overviewApsValue.text = summary.offlineAps.size.takeIf { it > 0 }
                ?.let { "$it / ${summary.aps.size}" }
                ?: summary.aps.size.toString()
            overviewApsLabel.text =
                if (summary.offlineAps.isEmpty()) "APs online" else "APs offline"
            val apsIssue = summary.offlineAps.isNotEmpty()
            setOverviewState(overviewApsValue, overviewApsLabel, apsIssue)

            val backboneDevices = summary.routers + summary.switches
            val offlineBackbone = summary.offlineBackbone
            overviewBackboneValue.text = offlineBackbone.size.takeIf { it > 0 }
                ?.let { "$it / ${backboneDevices.size}" }
                ?: backboneDevices.size.toString()
            overviewBackboneLabel.text =
                if (offlineBackbone.isEmpty()) "Routers/Switches online" else "Routers/Switches offline"
            val backboneIssue = offlineBackbone.isNotEmpty()
            setOverviewState(overviewBackboneValue, overviewBackboneLabel, backboneIssue)

            updateDeviceList(
                listOfflineGateways,
                summary.offlineGateways,
                "All gateways online."
            )
            updateDeviceList(
                listOfflineAps,
                summary.offlineAps,
                "All APs online."
            )
            updateDeviceList(
                listOfflineBackbone,
                summary.offlineBackbone,
                "All backbone devices online."
            )
            updateDeviceList(
                listLatency,
                summary.highLatencyCore,
                "No high latency core devices detected.",
                showLatency = true
            )
        }

        errorCard.isVisible = state.errorMessage != null
        errorText.text = state.errorMessage ?: ""
    }

    private fun updateDeviceList(
        container: LinearLayout,
        items: List<DeviceStatus>,
        emptyMessage: String,
        showLatency: Boolean = false
    ) {
        container.removeAllViews()
        if (items.isEmpty()) {
            val tv = TextView(container.context).apply {
                text = emptyMessage
                setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_BodyMedium)
                setTextColor(ContextCompat.getColor(context, R.color.uisp_text_secondary))
            }
            container.addView(tv)
            return
        }

        val inflater = layoutInflaterRef ?: LayoutInflater.from(container.context)
        items.forEach { device ->
            val row = inflater.inflate(R.layout.item_device_status, container, false)
            val nameView = row.findViewById<TextView>(R.id.text_name)
            val detailsView = row.findViewById<TextView>(R.id.text_details)
            val statusChip = row.findViewById<TextView>(R.id.text_status_chip)

            nameView.text = device.name
            val roleLabel = device.role.replaceFirstChar { if (it.isLowerCase()) it.titlecase() else it.toString() }
            val detailsText = buildString {
                append(roleLabel)
                if (showLatency) {
                    val latency = device.latencyMs?.roundToInt()
                    append(" - ")
                    append(latency?.let { "$it ms" } ?: "latency unknown")
                } else {
                    append(" - offline")
                }
            }
            detailsView.text = detailsText

            if (showLatency) {
                val latencyLabel = device.latencyMs?.roundToInt()?.let { "$it ms" } ?: "Latency"
                statusChip.text = latencyLabel
                statusChip.setTextColorRes(R.color.uisp_status_warn_fg)
                statusChip.setBackgroundResource(R.drawable.bg_status_chip_warn)
            } else if (device.online) {
                statusChip.text = "Online"
                statusChip.setTextColorRes(R.color.uisp_status_good_fg)
                statusChip.setBackgroundResource(R.drawable.bg_status_chip_good)
            } else {
                statusChip.text = "Offline"
                statusChip.setTextColorRes(R.color.uisp_status_bad_fg)
                statusChip.setBackgroundResource(R.drawable.bg_status_chip_bad)
            }

            container.addView(row)
        }
    }

    private fun setConnectionSummaryColor(@ColorRes colorRes: Int) {
        connectionSummary.setTextColor(ContextCompat.getColor(connectionSummary.context, colorRes))
    }

    private fun setOverviewState(
        valueView: TextView,
        labelView: TextView,
        hasIssue: Boolean
    ) {
        val valueColor = if (hasIssue) R.color.uisp_status_bad_fg else R.color.uisp_text_primary
        val labelColor = if (hasIssue) R.color.uisp_status_bad_fg else R.color.uisp_text_secondary
        valueView.setTextColorRes(valueColor)
        labelView.setTextColorRes(labelColor)
    }

    private fun TextView.setTextColorRes(@ColorRes colorRes: Int) {
        setTextColor(ContextCompat.getColor(context, colorRes))
    }
}

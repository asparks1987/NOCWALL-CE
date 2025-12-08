package com.uisp.noc.data

import com.uisp.noc.network.ApiClient
import com.uisp.noc.network.ApiResult
import com.uisp.noc.network.MobileConfigDto
import com.uisp.noc.network.DeviceSummaryDto
import com.uisp.noc.network.IncidentDto
import com.uisp.noc.network.PushRegisterResponse
import java.util.Locale

/**
 * Placeholder repository for the new API described in docs/PROJECT_PLAN.md.
 * Not yet wired into the UI; provides a seam to migrate away from the legacy WebView flow.
 */
class NewApiRepository(
    private val apiClient: ApiClient
) {

    suspend fun fetchMobileConfig(): ApiResult<MobileConfigDto> =
        apiClient.fetchMobileConfig()

    suspend fun registerPush(
        token: String,
        platform: String,
        appVersion: String,
        locale: Locale
    ): ApiResult<PushRegisterResponse> =
        apiClient.registerPush(token, platform, appVersion, locale)

    suspend fun fetchDevices(): ApiResult<DeviceSummaryDto> =
        apiClient.fetchDevices()

    suspend fun fetchIncidents(): ApiResult<List<IncidentDto>> =
        apiClient.fetchIncidents()
}

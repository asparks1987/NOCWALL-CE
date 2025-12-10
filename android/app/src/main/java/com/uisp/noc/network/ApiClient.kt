package com.uisp.noc.network

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.OkHttpClient
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.logging.HttpLoggingInterceptor
import java.io.IOException
import java.util.Locale
import java.util.UUID

/**
 * Early API client scaffold for the new UISP NOC backend (see docs/PROJECT_PLAN.md).
 * Uses OkHttp + Kotlin serialization; designed to surface verbose diagnostics for the UI banner.
 */
class ApiClient(
    baseUrl: String,
    private val tokenProvider: () -> String?,
    private val client: OkHttpClient = defaultClient(),
    private val json: Json = Json { ignoreUnknownKeys = true }
    ) {

    private val apiBase = baseUrl.trimEnd('/')

    suspend fun fetchMobileConfig(): ApiResult<MobileConfigDto> {
        val url = "$apiBase/nms/api/v2.1/mobile/config"
        val request = Request.Builder()
            .url(url)
            .get()
            .headers(commonHeaders())
            .build()
        return execute(request, "mobile_config_failed") { body ->
            json.decodeFromString<MobileConfigDto>(body)
        }
    }

    suspend fun registerPush(token: String, platform: String, appVersion: String, locale: Locale): ApiResult<PushRegisterResponse> {
        val url = "$apiBase/nms/api/v2.1/push/register"
        val payload = PushRegisterRequest(
            token = token,
            platform = platform,
            appVersion = appVersion,
            locale = locale.toLanguageTag()
        )
        val body = json.encodeToString(PushRegisterRequest.serializer(), payload)
            .toRequestBody(JSON_MEDIA_TYPE)
        val request = Request.Builder()
            .url(url)
            .post(body)
            .headers(commonHeaders())
            .build()
        return execute(request, "push_register_failed") { raw ->
            json.decodeFromString<PushRegisterResponse>(raw)
        }
    }

    suspend fun fetchDevices(): ApiResult<DeviceSummaryDto> {
        val url = "$apiBase/nms/api/v2.1/devices"
        val request = Request.Builder()
            .url(url)
            .get()
            .headers(commonHeaders())
            .build()
        return execute(request, "devices_fetch_failed") { raw ->
            json.decodeFromString<DeviceSummaryDto>(raw)
        }
    }

    suspend fun fetchIncidents(): ApiResult<List<IncidentDto>> {
        val url = "$apiBase/nms/api/v2.1/incidents"
        val request = Request.Builder()
            .url(url)
            .get()
            .headers(commonHeaders())
            .build()
        return execute(request, "incidents_fetch_failed") { raw ->
            json.decodeFromString<List<IncidentDto>>(raw)
        }
    }

    private fun commonHeaders() = okhttp3.Headers.Builder()
        .add("Accept", "application/json")
        .apply {
            tokenProvider()?.takeIf { it.isNotBlank() }?.let { add("Authorization", "Bearer $it") }
            add("X-Request-ID", UUID.randomUUID().toString())
        }
        .build()

    private suspend fun <T> execute(
        request: Request,
        defaultErrorCode: String,
        parser: (String) -> T
    ): ApiResult<T> {
        // Validate URL early
        if (request.url.toString().toHttpUrlOrNull() == null) {
            return ApiResult.Error(
                networkDiagnostic(
                    code = "invalid_url",
                    message = "Request URL is invalid",
                    detail = request.url.toString()
                )
            )
        }

        return try {
            val response = withContext(Dispatchers.IO) {
                client.newCall(request).execute()
            }
            response.use {
                val reqId = it.header("X-Request-ID")
                val bodyStr = it.body?.string().orEmpty()
                if (!it.isSuccessful) {
                    val diag = it.asDiagnostic(
                        body = bodyStr,
                        codeFallback = defaultErrorCode,
                        messageFallback = "HTTP ${it.code} from server"
                    )
                    return ApiResult.Error(diag)
                }
                val parsed = parser(bodyStr)
                ApiResult.Success(parsed, requestId = reqId)
            }
        } catch (ex: SerializationException) {
            ApiResult.Error(
                networkDiagnostic(
                    code = "parse_failed",
                    message = "Failed to parse server response",
                    detail = ex.message
                )
            )
        } catch (ex: IOException) {
            ApiResult.Error(
                networkDiagnostic(
                    code = "network_error",
                    message = "Network error while contacting server",
                    detail = ex.message
                )
            )
        }
    }

    companion object {
        private val JSON_MEDIA_TYPE = "application/json; charset=utf-8".toMediaType()

        fun defaultClient(): OkHttpClient {
            val logging = HttpLoggingInterceptor().apply {
                level = HttpLoggingInterceptor.Level.BASIC
            }
            return OkHttpClient.Builder()
                .addInterceptor(logging)
                .build()
        }
    }
}

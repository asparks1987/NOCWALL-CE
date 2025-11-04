package com.uisp.noc.data

import com.uisp.noc.data.model.DeviceStatus
import com.uisp.noc.data.model.DevicesSummary
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.plugins.cookies.*
import io.ktor.client.plugins.logging.*
import io.ktor.client.request.*
import io.ktor.client.request.forms.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.io.IOException

/**
 * Manages data operations for the UISP NOC application, handling authentication and
 * data fetching from a UISP backend.
 *
 * This repository abstracts the underlying network calls and data parsing, providing a
 * clean interface for the application to interact with the UISP API.
 *
 * @property httpClient The Ktor HttpClient used for all network requests.
 */
class UispRepository(private val httpClient: HttpClient = defaultClient()) {

    //region Public API

    /**
     * Authenticates with the UISP backend using the provided credentials.
     *
     * This method performs a multi-step authentication process:
     * 1. It sends a login request to the backend's form-based login endpoint.
     * 2. On success, it fetches a mobile-specific configuration which contains the final API token.
     *
     * @param backendUrl The base URL of the UISP instance (e.g., "https://uisp.example.com").
     * @param username The username for authentication.
     * @param password The password for authentication.
     * @return A [Session] object containing the necessary base URL and token for subsequent API calls.
     * @throws AuthException if the credentials are invalid or authentication fails.
     * @throws IOException for network errors or if the server response is malformed.
     */
    suspend fun authenticate(
        backendUrl: String,
        username: String,
        password: String
    ): Session = withContext(Dispatchers.IO) {
        val normalizedBackend = normalizeBackendUrl(backendUrl)

        // Step 1: Perform form-based login to establish a session cookie.
        // We set expectSuccess = false because a successful login results in a 302 redirect,
        // which Ktor would otherwise treat as an error.
        httpClient.submitForm(
            url = "$normalizedBackend/?action=login",
            formParameters = parameters {
                append("username", username)
                append("password", password)
            }
        ) { expectSuccess = false }

        // Step 2: Fetch the mobile config to get the API token. This call uses the default validator.
        val config = httpClient.get("$normalizedBackend/?ajax=mobile_config").body<MobileConfigResponse>()

        Session(
            backendUrl = normalizedBackend,
            username = username,
            uispBaseUrl = sanitizeBaseUrl(config.uispBaseUrl),
            uispToken = config.uispToken
        )
    }

    /**
     * Fetches a summary of all network devices from the UISP API.
     *
     * @param session The active, authenticated [Session].
     * @return A [DevicesSummary] object containing aggregated data about the network status.
     * @throws AuthException if the session token is expired or invalid.
     * @throws IOException for other network-related failures.
     */
    suspend fun fetchSummary(session: Session): DevicesSummary = withContext(Dispatchers.IO) {
        val devices = httpClient.get(session.uispBaseUrl.trimEnd('/') + "/nms/api/v2.1/devices") {
            header("x-auth-token", session.uispToken.trim())
        }.body<List<ApiDevice>>()

        parseDevices(devices)
    }

    //endregion

    //region Data Parsing and Transformation

    /**
     * Parses the raw list of devices from the API into a structured [DevicesSummary].
     *
     * @param apiDevices The list of [ApiDevice] objects returned from the UISP API.
     * @return A [DevicesSummary] object.
     */
    private fun parseDevices(apiDevices: List<ApiDevice>): DevicesSummary {
        val deviceStatuses = apiDevices.mapNotNull { device ->
            val id = device.identification.id ?: device.identification.mac ?: return@mapNotNull null
            val role = device.identification.role?.lowercase()?.trim() ?: "device"
            val isGateway = role == "gateway"
            val isBackbone = role in setOf("router", "switch") || isGateway

            DeviceStatus(
                id = id,
                name = device.identification.name ?: id,
                role = role,
                isGateway = isGateway,
                isBackbone = isBackbone,
                online = device.overview.status?.lowercase() in ONLINE_STATUSES,
                latencyMs = device.overview.latency
            )
        }

        val gateways = deviceStatuses.filter { it.isGateway }
        val backbones = deviceStatuses.filter { it.isBackbone && !it.isGateway }
        val cpes = deviceStatuses.filter { !it.isBackbone && !it.isGateway }

        return DevicesSummary(
            lastUpdatedEpochMillis = System.currentTimeMillis(),
            totalGateways = gateways.size,
            offlineGateways = gateways.filter { !it.online },
            totalBackbone = backbones.size,
            offlineBackbone = backbones.filter { !it.online },
            totalCpes = cpes.size,
            offlineCpes = cpes.filter { !it.online },
            highLatencyGateways = deviceStatuses.filter { (it.latencyMs ?: 0.0) > LATENCY_ALERT_THRESHOLD_MS }
        )
    }

    //endregion

    //region URL Sanitization

    /** Normalizes a backend URL to ensure it has a scheme and no trailing slash. */
    private fun normalizeBackendUrl(rawUrl: String): String {
        var trimmed = rawUrl.trim()
        if (!trimmed.contains("://")) {
            trimmed = "https://$trimmed"
        }
        return trimmed.removeSuffix("/")
    }

    /** Sanitizes the base API URL to ensure it has a scheme and no trailing slash. */
    private fun sanitizeBaseUrl(rawUrl: String): String {
        var trimmed = rawUrl.trim()
        if (!trimmed.contains("://")) {
            trimmed = "https://$trimmed"
        }
        return trimmed.removeSuffix("/")
    }

    //endregion

    //region Companion and Configuration

    /**
     * Represents an authentication-related error.
     * @param message A descriptive error message.
     */
    class AuthException(message: String) : IOException(message)

    companion object {
        private const val USER_AGENT = "UISP-NOC-Android/1.0"
        private val ONLINE_STATUSES = setOf("ok", "online", "active", "connected", "reachable", "enabled")
        private const val LATENCY_ALERT_THRESHOLD_MS = 450.0

        /**
         * Creates a default Ktor HttpClient with pre-configured settings for logging,
         * error handling, and JSON serialization.
         */
        fun defaultClient(): HttpClient {
            return HttpClient(CIO) {
                // CIO is a Coroutine-based I/O engine.
                engine {
                    requestTimeout = 10_000 // 10 seconds
                }

                // Configure logging for requests and responses.
                install(Logging) {
                    logger = Logger.DEFAULT
                    level = LogLevel.HEADERS
                }

                // Automatically handle cookies to maintain session.
                install(HttpCookies) {
                    storage = AcceptAllCookiesStorage()
                }

                // Configure JSON serialization.
                install(ContentNegotiation) {
                    json(Json { isLenient = true; ignoreUnknownKeys = true })
                }

                // Centralize response validation and error handling.
                install(HttpResponseValidator) {
                    handleResponseExceptionWithRequest { exception, _ ->
                        val clientException = exception as? ResponseException ?: return@handleResponseExceptionWithRequest
                        val response = clientException.response
                        val errorBody = try {
                            response.bodyAsText()
                        } catch (e: Exception) {
                            "(Could not read response body)"
                        }

                        when (response.status) {
                            HttpStatusCode.Unauthorized, HttpStatusCode.Forbidden -> {
                                throw AuthException("Authentication failed: ${response.status.value}. $errorBody")
                            }
                            else -> {
                                throw IOException("Request failed: ${response.status.value}. $errorBody")
                            }
                        }
                    }
                }

                // Apply a default User-Agent header to all requests.
                install(DefaultRequest) {
                    header(HttpHeaders.UserAgent, USER_AGENT)
                }
            }
        }
    }

    //endregion

    //region API Data Models

    /** Represents the JSON response from the /ajax=mobile_config endpoint. */
    @Serializable
    private data class MobileConfigResponse(
        @SerialName("uisp_base_url") val uispBaseUrl: String,
        @SerialName("uisp_token") val uispToken: String
    )

    /** Represents a single device object in the /devices API response. */
    @Serializable
    private data class ApiDevice(
        val identification: ApiIdentification,
        val overview: ApiOverview
    )

    /** Represents the "identification" block for a device. */
    @Serializable
    private data class ApiIdentification(
        val id: String? = null,
        val mac: String? = null,
        val name: String? = null,
        val role: String? = null
    )

    /** Represents the "overview" block for a device. */
    @Serializable
    private data class ApiOverview(
        val status: String? = null,
        val latency: Double? = null
    )

    //endregion
}

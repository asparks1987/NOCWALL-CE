package com.uisp.noc.network

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.Response
import java.util.UUID

private val json = Json { ignoreUnknownKeys = true }

fun Response.asDiagnostic(codeFallback: String, messageFallback: String): DiagnosticError {
    val rid = header("X-Request-ID") ?: UUID.randomUUID().toString()
    val bodyStr = body?.string().orEmpty()
    val detail = extractDetail(bodyStr)
    return DiagnosticError(
        code = codeFallback,
        message = messageFallback,
        detail = detail,
        httpStatus = code,
        requestId = rid
    )
}

fun networkDiagnostic(code: String, message: String, detail: String? = null): DiagnosticError =
    DiagnosticError(
        code = code,
        message = message,
        detail = detail,
        httpStatus = null,
        requestId = UUID.randomUUID().toString()
    )

private fun extractDetail(body: String): String? {
    return runCatching {
        val node = json.parseToJsonElement(body) as? JsonObject ?: return null
        listOf("detail", "message", "error").firstNotNullOfOrNull { key ->
            node[key]?.jsonPrimitive?.contentOrNull?.takeIf { it.isNotBlank() }
        }
    }.getOrNull()
}

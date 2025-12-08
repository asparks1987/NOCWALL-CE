package com.uisp.noc.network

/**
 * Basic API result wrapper with diagnostic-rich errors for banners/support.
 */
sealed class ApiResult<out T> {
    data class Success<T>(
        val data: T,
        val requestId: String?
    ) : ApiResult<T>()

    data class Error(
        val diagnostic: DiagnosticError
    ) : ApiResult<Nothing>()
}

data class DiagnosticError(
    val code: String,
    val message: String,
    val detail: String? = null,
    val httpStatus: Int? = null,
    val requestId: String,
    val timestampMillis: Long = System.currentTimeMillis()
)

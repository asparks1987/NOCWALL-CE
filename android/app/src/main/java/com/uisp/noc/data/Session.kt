package com.uisp.noc.data

data class Session(
    val backendUrl: String,
    val username: String,
    val uispBaseUrl: String,
    val uispToken: String,
    val authenticatedAtMillis: Long = System.currentTimeMillis()
)

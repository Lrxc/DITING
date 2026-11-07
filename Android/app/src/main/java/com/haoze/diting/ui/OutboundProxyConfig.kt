package com.haoze.diting.ui

import android.content.Context
import org.json.JSONObject

enum class OutboundProxyProtocol(val storageValue: String, val displayName: String) {
    SOCKS5("socks5", "SOCKS5"),
    HTTP("http", "HTTP CONNECT");

    companion object {
        fun fromStorageValue(value: String?): OutboundProxyProtocol =
            entries.firstOrNull { it.storageValue == value } ?: SOCKS5
    }
}

/**
 * Outbound proxy settings.
 *
 * [proxyAppPackages] is the set of applications whose traffic is routed
 * through the proxy. Applications that are not selected keep dialing
 * directly; DNS resolution is always direct.
 */
data class OutboundProxyConfig(
    val enabled: Boolean = false,
    val protocol: OutboundProxyProtocol = OutboundProxyProtocol.SOCKS5,
    val host: String = "127.0.0.1",
    val port: Int = 7890,
    val username: String = "",
    val password: String = "",
    val proxyAppPackages: List<String> = emptyList()
) {
    fun validationError(context: Context): String? {
        if (host != "127.0.0.1" && host != "::1") return "代理地址仅支持 127.0.0.1 或 ::1"
        if (port !in 1..65535) return "代理端口必须在 1 到 65535 之间"
        if (proxyAppPackages.isEmpty()) return "请至少选择一个走代理的应用"
        if (context.packageName in proxyAppPackages) return "不能选择 DITING 自身"
        val missing = proxyAppPackages.firstOrNull { packageName ->
            runCatching { context.packageManager.getApplicationInfo(packageName, 0) }.isFailure
        }
        if (missing != null) return "所选应用中有已卸载的应用"
        if (username.toByteArray().size > 255 || password.toByteArray().size > 255) return "代理账号或密码过长"
        return null
    }

    fun toNativeJson(): String = JSONObject()
        .put("enabled", enabled)
        .put("protocol", protocol.storageValue)
        .put("host", host)
        .put("port", port)
        .put("username", username)
        .put("password", password)
        .toString()
}

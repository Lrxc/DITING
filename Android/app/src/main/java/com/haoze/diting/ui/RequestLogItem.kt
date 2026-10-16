package com.haoze.diting.ui

import com.haoze.diting.core.log.LogResult
import com.haoze.diting.data.RequestSource
import com.haoze.diting.data.RequestStatus
import com.haoze.diting.data.entity.DnsLogEntity
import com.haoze.diting.data.entity.HttpRequestLogEntity
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

data class RequestLogItem(
    val key: String,
    val timestamp: Long,
    val source: RequestSource,
    val status: RequestStatus,
    val title: String,
    val subtitle: String,
    val detail: String?,
    val domain: String?,
    val packageName: String? = null,
    /** Resolved application label for [packageName], when it could be resolved. */
    val appName: String? = null,
    /** DNS record type, transport protocol, or HTTP protocol, depending on [source]. */
    val requestType: String? = null,
    /** Rule that blocked this request; null when nothing was blocked or no rule was recorded. */
    val matchedRule: String? = null,
    /**
     * Chain hop (CNAME/SVCB target) that actually matched the blocking rule. Set only when the
     * block came from response-chain filtering rather than the queried domain itself.
     */
    val blockedChainTarget: String? = null,
    /** True for an app-scoped blocking rule, false for a global one, null when unreported. */
    val ruleIsAppSpecific: Boolean? = null,
    /**
     * Rule-source token the engine reports in parentheses: `sub_6` for a subscription,
     * `important` / `security` / `filter_list` for a built-in list, or a rule pattern such as
     * `.*bytedecko.*`. Suffixes this app appends itself ("app rule" / "global rule") are excluded.
     */
    val blockedRuleSource: String? = null,
    /** Handling time in milliseconds, when the log recorded one. */
    val elapsedMs: Long? = null,
    /** Subscription that supplied the blocking rule, when one was recorded. */
    val blockSubscriptionId: Long? = null,
    /** Name of [blockSubscriptionId], resolved by the ViewModel. */
    val subscriptionName: String? = null,
    val cached: Boolean = false
)

private val requestTimeFormatter = ThreadLocal.withInitial {
    SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault())
}

private fun formatRequestTime(timestamp: Long): String =
    requestTimeFormatter.get()?.format(Date(timestamp)).orEmpty()

private val directOutcomes = setOf("passthrough", "bypassed", "unsupported_protocol", "resource_bypass")

fun dnsRequestItem(log: DnsLogEntity): RequestLogItem {
    val message = log.message.orEmpty()
    val isBlocked = log.result == LogResult.BLOCKED.value ||
        message.contains("blocked_by=firewall:", ignoreCase = true)
    val isConnection = log.queryType == 0 ||
        message.contains("blocked_by=connection", ignoreCase = true) ||
        message.contains("blocked_by=firewall:", ignoreCase = true) ||
        log.queryName.startsWith("TCP ", ignoreCase = true) ||
        log.queryName.startsWith("UDP ", ignoreCase = true)
    val rewritten = log.result == LogResult.REWRITTEN.value ||
        message.contains("matched rewrite rule", ignoreCase = true) ||
        message.contains("blocked_by=rewrite=", ignoreCase = true) ||
        message.contains("复写") || message.contains("覆写")
    val appInfo = log.packageName?.let { " · $it" } ?: ""
    val blockedRule = extractBlockedRule(message)
    val elapsedMs = extractElapsedMs(message)

    return if (isConnection) {
        val proto = if (log.queryName.startsWith("UDP", ignoreCase = true)) "UDP" else "TCP"
        val status = if (isBlocked) RequestStatus.BLOCKED else RequestStatus.BYPASSED
        val tag = if (isBlocked) "${proto}拦截" else "${proto}直连"
        RequestLogItem(
            key = "conn-${log.id}",
            timestamp = log.timestamp,
            source = RequestSource.DNS,
            status = status,
            title = log.queryName,
            subtitle = "${formatRequestTime(log.timestamp)} · $tag$appInfo",
            detail = log.message,
            domain = null,
            packageName = log.packageName,
            requestType = proto,
            matchedRule = if (status == RequestStatus.BLOCKED) blockedRule else null,
            blockedChainTarget = extractChainTarget(blockedRule),
            ruleIsAppSpecific = extractAppSpecific(blockedRule),
            blockedRuleSource = extractRuleSource(blockedRule),
            elapsedMs = elapsedMs,
            blockSubscriptionId = log.blockSubscriptionId ?: extractSubscriptionId(blockedRule),
            cached = false
        )
    } else {
        val status = when {
            rewritten -> RequestStatus.REWRITTEN
            log.result == LogResult.PASSED.value -> RequestStatus.PASSED
            log.result == LogResult.BLOCKED.value -> RequestStatus.BLOCKED
            else -> RequestStatus.ERROR
        }
        RequestLogItem(
            key = "dns-${log.id}",
            timestamp = log.timestamp,
            source = RequestSource.DNS,
            status = status,
            title = log.queryName,
            subtitle = "${formatRequestTime(log.timestamp)} · DNS · ${dnsRequestType(log.queryType)}${if (log.cached) " · 命中缓存" else ""}$appInfo",
            detail = log.message,
            domain = log.queryName,
            packageName = log.packageName,
            requestType = dnsRequestType(log.queryType),
            matchedRule = if (status == RequestStatus.BLOCKED) blockedRule else null,
            blockedChainTarget = extractChainTarget(blockedRule),
            ruleIsAppSpecific = extractAppSpecific(blockedRule),
            blockedRuleSource = extractRuleSource(blockedRule),
            elapsedMs = elapsedMs,
            blockSubscriptionId = log.blockSubscriptionId ?: extractSubscriptionId(blockedRule),
            cached = log.cached
        )
    }
}

fun httpRequestItem(log: HttpRequestLogEntity): RequestLogItem {
    val isBypassed = log.outcome in directOutcomes
    val isError = log.outcome in setOf("decryption_failed", "handshake_failed", "upstream_failed", "error")
    val detail = when {
        isBypassed -> {
            when (log.matchedRule) {
                "passthrough" -> "旁路直连 · 安全白名单 / 证书固定自动旁路"
                "unsupported_protocol" -> "旁路直连 · 不支持的协议直接转发"
                "resource_bypass" -> "旁路直连 · 静态资源旁路"
                null, "" -> "旁路直连 · 未解密直接转发"
                else -> "旁路原因 · ${log.matchedRule}"
            }
        }
        isError -> {
            when (log.matchedRule) {
                "client_tls" -> "握手失败 · 客户端拒绝证书 (可能存在证书绑定/Pinning) 未能建立连接"
                "upstream_tls" -> "上游异常 · 与目标服务器 TLS 握手失败"
                "upstream_dial" -> "建连异常 · 无法连接至上游服务器"
                "upstream_read_failed" -> "传输异常 · 读取上游响应失败"
                "upstream_write_failed" -> "传输异常 · 发送请求至上游失败"
                "client_write_failed" -> "传输异常 · 写回响应至客户端失败"
                "peek_flow_failed" -> "传输异常 · 客户端未发送有效数据或连接超时"
                null, "" -> "请求异常 · 处理过程出错"
                else -> "异常原因 · ${log.matchedRule}"
            }
        }
        else -> {
            log.matchedRule?.takeIf { it.isNotBlank() }?.let { "匹配规则 · $it" }
        }
    }

    val status = when (log.outcome) {
        "allowed" -> RequestStatus.PASSED
        "rewritten" -> RequestStatus.REWRITTEN
        "blocked", "invalid" -> RequestStatus.BLOCKED
        "passthrough", "bypassed", "unsupported_protocol", "resource_bypass" -> RequestStatus.BYPASSED
        else -> RequestStatus.ERROR
    }

    return RequestLogItem(
        key = "https-${log.id}",
        timestamp = log.timestamp,
        source = RequestSource.HTTPS,
        status = status,
        title = log.authority ?: "未取得 authority",
        subtitle = "${formatRequestTime(log.timestamp)} · ${log.protocol} · ${log.packageName}",
        detail = detail,
        domain = log.authority,
        packageName = log.packageName,
        requestType = log.protocol,
        matchedRule = if (status == RequestStatus.BLOCKED) log.matchedRule?.takeIf { it.isNotBlank() } else null,
        blockedChainTarget = extractChainTarget(log.matchedRule),
        ruleIsAppSpecific = extractAppSpecific(log.matchedRule),
        blockSubscriptionId = log.blockSubscriptionId
    )
}

private fun dnsRequestType(type: Int) = when (type) {
    1 -> "A"
    28 -> "AAAA"
    5 -> "CNAME"
    15 -> "MX"
    16 -> "TXT"
    2 -> "NS"
    else -> "TYPE_$type"
}

/** Pulls the rule name out of a DNS log message such as `app=x, blocked_by=<rule>, elapsed=3ms`. */
private fun extractBlockedRule(message: String): String? {
    val marker = "blocked_by="
    val index = message.indexOf(marker, ignoreCase = true)
    if (index < 0) return null
    return message.substring(index + marker.length)
        .substringBefore(',')
        .trim()
        .takeIf { it.isNotBlank() }
}

/** Pulls the elapsed milliseconds out of a DNS log message, when present. */
private fun extractElapsedMs(message: String): Long? {
    val marker = "elapsed="
    val index = message.indexOf(marker, ignoreCase = true)
    if (index < 0) return null
    return message.substring(index + marker.length)
        .substringBefore(',')
        .trim()
        .removeSuffix("ms")
        .trim()
        .toLongOrNull()
}

/**
 * Pulls the CNAME/SVCB chain target out of a blocking reason such as
 * `cname:oth.eve.mdt.qq.com (sub_6) (global rule)`. The rule fired on the chain target, so that
 * is the domain worth showing; returns null for a direct block on the queried domain.
 */
private fun extractChainTarget(reason: String?): String? {
    val prefix = when {
        reason == null -> return null
        reason.startsWith("cname:", ignoreCase = true) -> "cname:"
        reason.startsWith("svcb:", ignoreCase = true) -> "svcb:"
        else -> return null
    }
    return reason.removePrefix(prefix).substringBefore(' ').trim().takeIf { it.isNotEmpty() }
}

/** `... (app rule)` means app-scoped, `... (global rule)` means global, otherwise unknown. */
private fun extractAppSpecific(reason: String?): Boolean? = when {
    reason == null -> null
    reason.contains("(app rule)", ignoreCase = true) -> true
    reason.contains("(global rule)", ignoreCase = true) -> false
    else -> null
}

/**
 * Subscription id embedded in an engine reason such as `cname:host (sub_6) (global rule)`.
 *
 * This is a fallback for rows whose persisted `blockSubscriptionId` is empty — records written
 * before the tunnel learned to parse the token out of the reason. Without it the UI cannot name
 * the subscription even though the engine reported it.
 */
private val SUBSCRIPTION_ID_REGEX = Regex("""(?:^|[^\w])sub_(\d+)""")

private fun extractSubscriptionId(reason: String?): Long? =
    reason?.let { SUBSCRIPTION_ID_REGEX.find(it)?.groupValues?.get(1)?.toLongOrNull() }

/**
 * Rule-source token embedded in an engine reason, e.g. `sub_6` in `cname:host (sub_6)`, or the
 * rule pattern itself in `cname:host (.*bytedecko.*)`. The first parenthesised group that is not
 * one of the app's own suffixes wins.
 */
private val RULE_SOURCE_REGEX = Regex("""\(([^()]+)\)""")

private fun extractRuleSource(reason: String?): String? = reason?.let {
    RULE_SOURCE_REGEX.findAll(it)
        .map { match -> match.groupValues[1].trim() }
        .firstOrNull { token ->
            token.isNotBlank() &&
                !token.equals("app rule", ignoreCase = true) &&
                !token.equals("global rule", ignoreCase = true)
        }
}

private const val SYSTEM_UID_PREFIX = "uid:"
private const val UNKNOWN_OWNER = "unknown"
private const val ROOT_PROXY_OWNER = "RootProxy"

/**
 * Owner markers the tunnel writes when a request cannot be attributed to an installed package.
 * They are not package names, so the UI must label them explicitly instead of resolving them.
 */
sealed interface RequestOwner {
    /** Kernel/root traffic known only by UID, written as `uid:<n>`. */
    data class Uid(val uid: Int) : RequestOwner

    /** The tunnel could not resolve the owner at all, written as `unknown`. */
    data object Unknown : RequestOwner

    /** Server mode: the query came from this device rather than a LAN client (`RootProxy`). */
    data object RootProxy : RequestOwner
}

/**
 * Decodes those markers so the list and the detail sheet can show a name and a placeholder icon
 * instead of a blank row. Returns null for real package names and for missing packages.
 */
fun RequestLogItem.requestOwner(): RequestOwner? {
    val pkg = packageName ?: return null
    return when {
        pkg.startsWith(SYSTEM_UID_PREFIX) ->
            pkg.removePrefix(SYSTEM_UID_PREFIX).toIntOrNull()?.let { RequestOwner.Uid(it) }

        pkg.equals(UNKNOWN_OWNER, ignoreCase = true) -> RequestOwner.Unknown
        pkg.equals(ROOT_PROXY_OWNER, ignoreCase = true) -> RequestOwner.RootProxy
        else -> null
    }
}

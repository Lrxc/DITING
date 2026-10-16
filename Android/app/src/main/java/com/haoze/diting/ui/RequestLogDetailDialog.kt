package com.haoze.diting.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.haoze.diting.ui.components.AppAlertDialog
import com.haoze.diting.ui.components.AppDialogButton
import com.haoze.diting.ui.components.AppIconImage
import com.haoze.diting.ui.components.SystemAppPlaceholder
import com.haoze.diting.util.formatMs
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

private val detailTimeFormatter = ThreadLocal.withInitial {
    SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault())
}

/**
 * Read-only detail sheet for a single request log entry. Every field occupies its own line so
 * the list rows can stay terse.
 */
@Composable
fun RequestLogDetailDialog(
    item: RequestLogItem,
    onDismiss: () -> Unit,
    onDomainAction: (() -> Unit)? = null
) {
    val domainButton: (@Composable () -> Unit)? = onDomainAction?.let { action ->
        { AppDialogButton(label = "域名操作", onClick = action) }
    }
    val owner = if (item.appName == null) item.requestOwner() else null
    val appLabel = item.appName ?: requestOwnerLabel(owner)
    // Shared with the list rows so both stay on the same palette.
    val statusColor = requestStatusColor(item.status)
    val highlighted = FontWeight.SemiBold

    AppAlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Column {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    if (owner != null) {
                        SystemAppPlaceholder(size = 32.dp)
                    } else {
                        AppIconImage(packageName = item.packageName, size = 32.dp)
                    }
                    Text(
                        text = appLabel ?: localizedText("请求详情"),
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f)
                    )
                }
                HorizontalDivider(
                    modifier = Modifier.padding(top = 12.dp),
                    color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f)
                )
            }
        },
        text = {
            Column(
                modifier = Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                DetailLine(
                    label = localizedText("域名"),
                    value = item.title,
                    valueColor = MaterialTheme.colorScheme.primary,
                    valueWeight = highlighted
                )
                val subscriptionColor = MaterialTheme.colorScheme.tertiary
                val ruleColor = MaterialTheme.colorScheme.error
                val scopeColor = MaterialTheme.colorScheme.onSurfaceVariant
                DetailLine(
                    label = localizedText("规则"),
                    // Up to three colour-coded lines: the subscribing source (tertiary), the rule
                    // itself (error), and which list it came from (muted). `block.trie` (the
                    // blacklist) reports the rule pattern, so a parenthesised non-token is the rule
                    // and counts as 黑名单.
                    value = buildAnnotatedString {
                        val source = item.blockedRuleSource
                        val builtInTokens = setOf("important", "security", "filter_list")
                        val isBlacklistPattern = source != null &&
                            !source.startsWith("sub_") &&
                            source !in builtInTokens

                        val subscriptionLine = item.subscriptionName
                            ?: item.blockSubscriptionId?.let {
                                "${localizedText("订阅")} #$it · ${localizedText("未找到该订阅")}"
                            }
                        val scope = when {
                            isBlacklistPattern -> localizedText("黑名单")
                            source == "important" -> localizedText("重要规则")
                            source == "security" -> localizedText("安全规则")
                            source == "filter_list" -> localizedText("内置过滤")
                            item.ruleIsAppSpecific == true -> localizedText("应用规则")
                            item.ruleIsAppSpecific == false -> localizedText("全局规则")
                            else -> null
                        }

                        val lines = mutableListOf<Pair<String, Color>>()
                        subscriptionLine?.let { lines += it to subscriptionColor }
                        if (isBlacklistPattern && source != null) lines += source to ruleColor
                        // Scope is parenthesised so it reads as metadata, not as another rule line.
                        scope?.let { lines += "($it)" to scopeColor }
                        if (lines.isEmpty()) {
                            // Last resort: the chain hop that matched, then the raw engine reason.
                            val fallback = item.blockedChainTarget
                                ?: item.matchedRule.orEmpty().ifBlank { localizedText("未记录") }
                            lines += fallback to scopeColor
                        }

                        lines.forEachIndexed { index, (text, color) ->
                            if (index > 0) append('\n')
                            withStyle(SpanStyle(color = color)) { append(text) }
                        }
                    },
                    valueWeight = highlighted
                )
                DetailLine(
                    label = localizedText("状态"),
                    value = localizedText(item.status.label),
                    valueColor = statusColor,
                    valueWeight = highlighted
                )
                DetailLine(localizedText("来源"), localizedText(item.source.label))
                DetailLine(localizedText("类型"), item.requestType.orEmpty())
                DetailLine(
                    label = localizedText("耗时"),
                    value = formatMs((item.elapsedMs ?: 0L).toDouble())
                )
                DetailLine(
                    label = localizedText("缓存"),
                    value = localizedText(if (item.cached) "命中" else "未命中"),
                    valueColor = if (item.cached) {
                        MaterialTheme.colorScheme.tertiary
                    } else {
                        MaterialTheme.colorScheme.onSurfaceVariant
                    },
                    valueWeight = highlighted
                )
                DetailLine(localizedText("时间"), detailTimeFormatter.get()?.format(Date(item.timestamp)).orEmpty())
                DetailLine(localizedText("详情"), item.detail.orEmpty())
            }
        },
        confirmButton = { AppDialogButton(label = "关闭", onClick = onDismiss) },
        dismissButton = domainButton
    )
}

@Composable
private fun DetailLine(
    label: String,
    value: CharSequence,
    valueColor: Color? = null,
    valueWeight: FontWeight = FontWeight.Normal
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.Top
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.width(56.dp)
        )
        if (value is AnnotatedString) {
            // Rich value: colours come from the spans, so no single valueColor applies.
            Text(
                text = value,
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = valueWeight,
                modifier = Modifier.weight(1f)
            )
        } else {
            Text(
                text = value.toString(),
                style = MaterialTheme.typography.bodyMedium,
                color = valueColor ?: MaterialTheme.colorScheme.onSurface,
                fontWeight = valueWeight,
                modifier = Modifier.weight(1f)
            )
        }
    }
}

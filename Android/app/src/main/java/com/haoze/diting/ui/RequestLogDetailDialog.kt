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
import androidx.compose.ui.text.font.FontWeight
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
    val systemUid = if (item.appName == null) item.systemUid() else null
    val appLabel = item.appName
        ?: systemUid?.let { localizedText("系统") + " (uid $it)" }

    AppAlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Column {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    if (systemUid != null) {
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
                DetailLine(localizedText("域名"), item.title)
                DetailLine(localizedText("规则"), item.subscriptionName.orEmpty())
                DetailLine(localizedText("状态"), localizedText(item.status.label))
                DetailLine(localizedText("来源"), localizedText(item.source.label))
                DetailLine(localizedText("类型"), item.requestType.orEmpty())
                DetailLine(localizedText("耗时"), item.elapsedMs?.let { formatMs(it.toDouble()) }.orEmpty())
                DetailLine(localizedText("缓存"), if (item.cached) localizedText("命中缓存") else "")
                DetailLine(localizedText("时间"), detailTimeFormatter.get()?.format(Date(item.timestamp)).orEmpty())
                DetailLine(localizedText("详情"), item.detail.orEmpty())
            }
        },
        confirmButton = { AppDialogButton(label = "关闭", onClick = onDismiss) },
        dismissButton = domainButton
    )
}

@Composable
private fun DetailLine(label: String, value: String) {
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
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface,
            modifier = Modifier.weight(1f)
        )
    }
}

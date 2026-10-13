package com.haoze.diting.ui.dashboard

import com.haoze.diting.util.formatMs
import com.haoze.diting.util.formatPercent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.haoze.diting.ui.ModernLogDashboardUiState
import com.haoze.diting.ui.localizedText
import com.haoze.diting.ui.components.SettingsCardMargin

@Composable
fun OffModeDashboard(
    state: ModernLogDashboardUiState,
    onNavigateToDnsCache: () -> Unit,
    onNavigateToRaceStats: () -> Unit,
    onNavigateToBootstrapStats: () -> Unit,
    onNavigateToTrafficStats: (() -> Unit)? = null,
    contentBottomPadding: Dp = 0.dp
) {
    val race = state.race
    val bootstrap = state.bootstrap
    val raceRate = if (race.requests > 0) race.successes.toDouble() / race.requests else 0.0

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = SettingsCardMargin, top = 12.dp, end = SettingsCardMargin, bottom = 28.dp + contentBottomPadding),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        item {
            DashboardHero(
                title = localizedText("服务状态与解析概览"),
                description = localizedText("当前处于隐私模式，不记录请求明细。DNS 缓存、并发竞速与网络流量监控仍正常运行。")
            )
        }
        item {
            DashboardCard {
                SectionTitle(
                    title = localizedText("隐私模式保护中"),
                    trailing = { PillLabel(localizedText("日志已关闭")) }
                )
                Text(
                    text = localizedText("重新开启日志后，仪表盘将从新产生的请求开始统计。当前各类解析服务与缓存热点均正常工作。"),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
        if (onNavigateToTrafficStats != null) {
            item {
                TrafficMonitorDashboardCard(onClick = onNavigateToTrafficStats)
            }
        }
        item {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                MetricCard(
                    label = localizedText("有效缓存"),
                    value = formatCount(state.totalCacheEntries),
                    valueColor = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.weight(1f)
                )
                MetricCard(
                    label = localizedText("累计缓存命中"),
                    value = formatCount(state.totalCacheHits),
                    valueColor = MaterialTheme.colorScheme.tertiary,
                    modifier = Modifier.weight(1f)
                )
            }
        }
        item {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                MetricCard(
                    label = localizedText("并发竞速请求"),
                    value = formatCount(race.requests),
                    valueColor = MaterialTheme.colorScheme.secondary,
                    modifier = Modifier.weight(1f)
                )
                MetricCard(
                    label = localizedText("Bootstrap 尝试"),
                    value = formatCount(bootstrap.attempts),
                    valueColor = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.weight(1f)
                )
            }
        }
        item {
            DashboardCard {
                SectionTitle(
                    title = localizedText("解析效能概况"),
                    trailing = { PillLabel(localizedText("实时统计")) }
                )
                RateRow(
                    label = localizedText("竞速成功率"),
                    value = localizedText("${formatPercent(raceRate)} · ${formatCount(race.requests)} 次 · ${formatMs(race.avgElapsedMs)}"),
                    progress = raceRate.toFloat()
                )
                Spacer(modifier = Modifier.height(14.dp))
                RateRow(
                    label = localizedText("Bootstrap 成功率"),
                    value = localizedText("${formatPercent(bootstrap.successRate)} · ${formatCount(bootstrap.attempts)} 次 · ${formatMs(bootstrap.avgElapsedMs)}"),
                    progress = bootstrap.successRate.toFloat()
                )
            }
        }
        item {
            DashboardListCard(
                title = localizedText("缓存热点"),
                emptyText = localizedText("暂无有效缓存"),
                rowCount = state.cacheEntries.size,
                trailing = { PillLink(text = localizedText("详情"), onClick = onNavigateToDnsCache) }
            ) { index ->
                CacheEntryRow(state.cacheEntries[index])
            }
        }
        item {
            DashboardListCard(
                title = localizedText("竞速胜出"),
                emptyText = localizedText("暂无竞速数据"),
                rowCount = race.winners.size,
                trailing = { PillLink(text = localizedText("详情"), onClick = onNavigateToRaceStats) }
            ) { index ->
                val item = race.winners[index]
                DashboardListRow(
                    title = item.name,
                    subtitle = localizedText("平均胜出耗时 ${formatMs(item.avgElapsedMs)}"),
                    tag = localizedText("${formatCount(item.wins)} 次"),
                    tagColor = MaterialTheme.colorScheme.secondary
                )
            }
        }
        item {
            DashboardListCard(
                title = localizedText("Bootstrap DNS"),
                emptyText = localizedText("暂无 Bootstrap 数据"),
                rowCount = bootstrap.ips.size,
                trailing = { PillLink(text = localizedText("详情"), onClick = onNavigateToBootstrapStats) }
            ) { index ->
                val item = bootstrap.ips[index]
                DashboardListRow(
                    title = item.name,
                    subtitle = localizedText("${item.ip} · 成功 ${formatPercent(item.successRate)} · ${formatMs(item.avgElapsedMs)}"),
                    tag = localizedText("${formatCount(item.attempts)} 次"),
                    tagColor = MaterialTheme.colorScheme.primary
                )
            }
        }
    }
}

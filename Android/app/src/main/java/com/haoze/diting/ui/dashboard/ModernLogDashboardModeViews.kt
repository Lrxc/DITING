package com.haoze.diting.ui.dashboard

import com.haoze.diting.util.formatMs
import com.haoze.diting.util.formatPercent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.haoze.diting.ui.ModernLogDashboardUiState
import com.haoze.diting.ui.localizedText
import com.haoze.diting.data.RequestStatus
import com.haoze.diting.ui.components.SettingsCardMargin

/** Dashboard preview lists show only the leading entry; the "详情" pages hold the full lists. */
private const val DASHBOARD_LIST_LIMIT = 3

@Composable
fun AllModeDashboard(
    state: ModernLogDashboardUiState,
    onNavigateToDnsLogs: () -> Unit,
    onNavigateToDnsCache: () -> Unit,
    onNavigateToRaceStats: () -> Unit,
    onNavigateToBootstrapStats: () -> Unit,
    onNavigateToSubscriptionInterceptionStats: () -> Unit,
    onNavigateToTrafficStats: (() -> Unit)? = null,
    onNavigateToRequestLogs: ((RequestStatus?) -> Unit)? = null,
    contentBottomPadding: Dp = 0.dp
) {
    val stats = state.dailyStats
    val race = state.race
    val bootstrap = state.bootstrap
    val raceRate = if (race.requests > 0) race.successes.toDouble() / race.requests else 0.0
    val blockRate = if (stats.total > 0) stats.blocked.toDouble() / stats.total else 0.0
    val cacheRate = if (stats.total > 0) stats.cached.toDouble() / stats.total else 0.0

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = SettingsCardMargin, top = 12.dp, end = SettingsCardMargin, bottom = 28.dp + contentBottomPadding),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
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
                    label = localizedText("今日请求"),
                    value = formatCount(stats.total),
                    valueColor = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.weight(1f),
                    onClick = onNavigateToDnsLogs
                )
                MetricCard(
                    label = localizedText("过滤"),
                    value = formatCount(stats.blocked),
                    valueColor = MaterialTheme.colorScheme.error,
                    modifier = Modifier.weight(1f),
                    onClick = { onNavigateToRequestLogs?.invoke(RequestStatus.BLOCKED) }
                )
            }
        }
        item {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                MetricCard(
                    label = localizedText("订阅规则"),
                    value = formatCount(state.subscriptions.totalRuleCount),
                    valueColor = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.weight(1f),
                    onClick = onNavigateToSubscriptionInterceptionStats
                )
                MetricCard(
                    label = localizedText("DNS 缓存"),
                    value = formatCount(state.totalCacheEntries),
                    valueColor = MaterialTheme.colorScheme.tertiary,
                    modifier = Modifier.weight(1f),
                    onClick = onNavigateToDnsCache
                )
            }
        }
        item {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                MetricCard(
                    label = localizedText("竞速胜出"),
                    value = formatCount(race.successes),
                    valueColor = MaterialTheme.colorScheme.secondary,
                    modifier = Modifier.weight(1f),
                    onClick = onNavigateToRaceStats
                )
                MetricCard(
                    label = localizedText("Bootstrap DNS"),
                    value = formatCount(bootstrap.attempts),
                    valueColor = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.weight(1f),
                    onClick = onNavigateToBootstrapStats
                )
            }
        }
        item {
            DashboardCard {
                SectionTitle(title = localizedText("今日流量结构"), trailing = { PillLabel(localizedText("实时汇总")) })
                state.error?.let { message ->
                    ErrorBanner(message = message)
                    Spacer(modifier = Modifier.height(12.dp))
                }
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(IntrinsicSize.Min),
                    horizontalArrangement = Arrangement.spacedBy(24.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    ResultDonut(
                        stats = stats,
                        modifier = Modifier.size(128.dp)
                    )
                    Column(
                        modifier = Modifier
                            .weight(1f)
                            .fillMaxHeight()
                            .padding(end = 8.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp, Alignment.CenterVertically)
                    ) {
                        LegendRow(localizedText("通过"), formatCount(stats.passed), MaterialTheme.colorScheme.primary)
                        LegendRow(localizedText("过滤"), formatCount(stats.blocked), MaterialTheme.colorScheme.error)
                        LegendRow(localizedText("失败"), formatCount(stats.error), MaterialTheme.colorScheme.error.copy(alpha = 0.7f))
                        LegendRow(localizedText("旁路"), formatCount(stats.bypassed), MaterialTheme.colorScheme.secondary)
                        LegendRow(localizedText("缓存"), formatCount(stats.cached), MaterialTheme.colorScheme.tertiary)
                    }
                }
                Spacer(modifier = Modifier.height(14.dp))
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
                Spacer(modifier = Modifier.height(14.dp))
                RateRow(
                    label = localizedText("拦截率"),
                    value = localizedText("${formatPercent(blockRate)} · ${formatCount(stats.blocked)} 次"),
                    progress = blockRate.toFloat()
                )
                Spacer(modifier = Modifier.height(14.dp))
                RateRow(
                    label = localizedText("缓存率"),
                    value = localizedText("${formatPercent(cacheRate)} · ${formatCount(stats.cached)} 次"),
                    progress = cacheRate.toFloat()
                )
            }
        }
    }
}

@Composable
fun FilteredModeDashboard(
    state: ModernLogDashboardUiState,
    onNavigateToDnsLogs: () -> Unit,
    onNavigateToDnsCache: () -> Unit,
    onNavigateToRaceStats: () -> Unit,
    onNavigateToBootstrapStats: () -> Unit,
    onNavigateToSubscriptionInterceptionStats: () -> Unit,
    onNavigateToTrafficStats: (() -> Unit)? = null,
    contentBottomPadding: Dp = 0.dp
) {
    val stats = state.dailyStats
    val race = state.race
    val bootstrap = state.bootstrap
    val raceRate = if (race.requests > 0) race.successes.toDouble() / race.requests else 0.0
    val blockRate = if (stats.total > 0) stats.blocked.toDouble() / stats.total else 0.0

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = SettingsCardMargin, top = 12.dp, end = SettingsCardMargin, bottom = 28.dp + contentBottomPadding),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        item {
            DashboardHero(
                title = localizedText("重点请求仪表盘"),
                description = localizedText("DNS 与 HTTPS 拦截、错误、旁路记录及解析状态概览")
            )
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
                    label = localizedText("重点记录"),
                    value = formatCount(stats.total),
                    valueColor = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.weight(1f)
                )
                MetricCard(
                    label = localizedText("过滤拦截"),
                    value = formatCount(stats.blocked),
                    valueColor = MaterialTheme.colorScheme.error,
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
                    label = localizedText("异常失败"),
                    value = formatCount(stats.error),
                    valueColor = MaterialTheme.colorScheme.error.copy(alpha = 0.7f),
                    modifier = Modifier.weight(1f)
                )
                MetricCard(
                    label = localizedText("旁路直连"),
                    value = formatCount(stats.bypassed),
                    valueColor = MaterialTheme.colorScheme.secondary,
                    modifier = Modifier.weight(1f)
                )
            }
        }
        item {
            DashboardCard {
                SectionTitle(title = localizedText("重点流量结构"), trailing = { PillLabel(localizedText("实时汇总")) })
                state.error?.let { message ->
                    ErrorBanner(message = message)
                    Spacer(modifier = Modifier.height(12.dp))
                }
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(IntrinsicSize.Min),
                    horizontalArrangement = Arrangement.spacedBy(24.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    ResultDonut(
                        stats = stats,
                        modifier = Modifier.size(128.dp)
                    )
                    Column(
                        modifier = Modifier
                            .weight(1f)
                            .fillMaxHeight()
                            .padding(end = 8.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterVertically)
                    ) {
                        LegendRow(localizedText("过滤"), formatCount(stats.blocked), MaterialTheme.colorScheme.error)
                        LegendRow(localizedText("失败"), formatCount(stats.error), MaterialTheme.colorScheme.error.copy(alpha = 0.7f))
                        LegendRow(localizedText("旁路"), formatCount(stats.bypassed), MaterialTheme.colorScheme.secondary)
                    }
                }
                Spacer(modifier = Modifier.height(14.dp))
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
                Spacer(modifier = Modifier.height(14.dp))
                RateRow(
                    label = localizedText("拦截占比"),
                    value = localizedText("${formatPercent(blockRate)} · ${formatCount(stats.blocked)} 次"),
                    progress = blockRate.toFloat()
                )
            }
        }
        item {
            DashboardCard {
                SectionTitle(
                    title = localizedText("最近拦截与错误"),
                    trailing = { PillLink(text = localizedText("查看全部"), onClick = onNavigateToDnsLogs) }
                )
                if (state.recentLogs.isEmpty()) {
                    EmptyText(localizedText("暂无拦截或错误记录"))
                } else {
                    state.recentLogs.forEachIndexed { index, item ->
                        if (index > 0) ListDivider()
                        RequestLogRow(item)
                    }
                }
            }
        }
        item {
            DashboardListCard(
                title = localizedText("竞速胜出"),
                emptyText = localizedText("暂无竞速数据"),
                rowCount = minOf(race.winners.size, DASHBOARD_LIST_LIMIT),
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
                rowCount = minOf(bootstrap.ips.size, DASHBOARD_LIST_LIMIT),
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


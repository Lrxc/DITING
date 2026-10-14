package com.haoze.diting.ui.traffic

enum class TrafficTimeRange(val displayName: String) {
    SESSION("本次会话"),
    TODAY("今日"),
    THIS_WEEK("本周"),
    THIS_MONTH("本月"),
    ALL_HISTORY("全部历史")
}

enum class TrafficSortMode(val displayName: String) {
    TOTAL_TRAFFIC("按总流量"),
    REALTIME_SPEED("按实时网速"),
    BLOCKED("按拦截数量")
}

data class AppTrafficUiItem(
    val packageName: String,
    val appName: String,
    val isSystemApp: Boolean,
    val txBytes: Long,
    val rxBytes: Long,
    val totalBytes: Long,
    val percentage: Float, // 0.0f - 1.0f
    val currentTxSpeedBps: Long,
    val currentRxSpeedBps: Long,
    val currentTotalSpeedBps: Long,
    /** Requests blocked by filtering rules during the selected time range (DNS + HTTPS). */
    val blockedCount: Int = 0,
    /** All requests recorded for this app during the selected time range (DNS + HTTPS). */
    val totalRequestCount: Int = 0
)

data class AppDetailTrafficState(
    val packageName: String,
    val appName: String,
    val isSystemApp: Boolean,
    val todayTxBytes: Long,
    val todayRxBytes: Long,
    val sessionTxBytes: Long,
    val sessionRxBytes: Long,
    val totalHistoryTxBytes: Long,
    val totalHistoryRxBytes: Long
)

data class AppTrafficStatsUiState(
    val isRunning: Boolean = false,
    val isGoTunnelActive: Boolean = false,
    val selectedTimeRange: TrafficTimeRange = TrafficTimeRange.TODAY,
    val selectedSortMode: TrafficSortMode = TrafficSortMode.BLOCKED,
    val searchQuery: String = "",
    val hideSystemApps: Boolean = false,
    val totalTxSpeedBps: Long = 0L,
    val totalRxSpeedBps: Long = 0L,
    val currentPeriodTxBytes: Long = 0L,
    val currentPeriodRxBytes: Long = 0L,
    val currentPeriodTotalBytes: Long = 0L,
    val sessionStartTimeMs: Long = 0L,
    val appItems: List<AppTrafficUiItem> = emptyList(),
    val isLoading: Boolean = false,
    val selectedAppDetail: AppDetailTrafficState? = null
)

package com.haoze.diting.normal.ui

import com.haoze.diting.core.log.HttpRequestOutcome
import com.haoze.diting.core.log.LogResult
import com.haoze.diting.data.AppDatabase
import com.haoze.diting.ui.traffic.AppBlockedRecord
import kotlinx.coroutines.CancellationException

/** Upper bound on how many blocked entries the app detail sheet lists. */
private const val BLOCKED_RECORD_LIMIT = 200

private const val SOURCE_DNS = "DNS"
private const val SOURCE_HTTPS = "HTTPS"

/**
 * Loads the requests blocked for [packageName] at or after [since], merging the DNS and HTTPS
 * logs and ordering them newest first.
 */
internal suspend fun loadAppBlockedRecords(
    db: AppDatabase,
    packageName: String,
    since: Long
): List<AppBlockedRecord> = try {
    val dnsRecords = db.dnsLogDao()
        .blockedLogsByPackage(packageName, since, LogResult.BLOCKED.value, BLOCKED_RECORD_LIMIT)
        .map { AppBlockedRecord(timestamp = it.timestamp, source = SOURCE_DNS, target = it.queryName) }
    val httpsRecords = db.httpRequestLogDao()
        .blockedLogsByPackage(packageName, since, HttpRequestOutcome.BLOCKED.storageValue, BLOCKED_RECORD_LIMIT)
        .map {
            AppBlockedRecord(
                timestamp = it.timestamp,
                source = SOURCE_HTTPS,
                target = it.authority ?: "未取得 authority"
            )
        }
    (dnsRecords + httpsRecords)
        .sortedByDescending { it.timestamp }
        .take(BLOCKED_RECORD_LIMIT)
} catch (e: CancellationException) {
    throw e
} catch (_: Exception) {
    emptyList()
}

package com.haoze.diting.data.repository

import androidx.sqlite.db.SimpleSQLiteQuery
import com.haoze.diting.data.RequestSource
import com.haoze.diting.data.RequestStatus
import com.haoze.diting.data.dao.DnsLogDao
import com.haoze.diting.data.dao.HttpRequestLogDao
import com.haoze.diting.data.entity.DnsLogEntity
import com.haoze.diting.data.entity.HttpRequestLogEntity
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope

data class RequestLogBatch(
    val dns: List<DnsLogEntity>,
    val http: List<HttpRequestLogEntity>,
    val hasMore: Boolean
)

class RequestLogRepository(
    private val dnsDao: DnsLogDao,
    private val httpDao: HttpRequestLogDao? = null,
    private val onFlushRequested: (suspend () -> Unit)? = null
) {
    companion object {
        var activeFlusher: (suspend () -> Unit)? = null
    }

    suspend fun load(
        limit: Int,
        source: RequestSource = RequestSource.ALL,
        status: RequestStatus = RequestStatus.ALL,
        query: String = "",
        sinceMillis: Long? = null
    ): RequestLogBatch = coroutineScope {
        runCatching {
            onFlushRequested?.invoke() ?: activeFlusher?.invoke()
        }

        val dns = async {
            if (source == RequestSource.HTTPS) {
                emptyList()
            } else {
                val filter = dnsFilter(status, query, sinceMillis)
                dnsDao.queryList(
                    SimpleSQLiteQuery(
                        "SELECT * FROM dns_log WHERE 1=1${filter.sql} ORDER BY timestamp DESC LIMIT $limit",
                        filter.args
                    )
                )
            }
        }

        val http = async {
            val dao = httpDao
            if (source == RequestSource.DNS || dao == null) {
                emptyList()
            } else {
                val filter = httpFilter(status, query, sinceMillis)
                dao.queryList(
                    SimpleSQLiteQuery(
                        "SELECT * FROM http_request_log WHERE 1=1${filter.sql} ORDER BY timestamp DESC LIMIT $limit",
                        filter.args
                    )
                )
            }
        }

        val dnsRows = dns.await()
        val httpRows = http.await()
        RequestLogBatch(dnsRows, httpRows, dnsRows.size >= limit || httpRows.size >= limit)
    }

    /** Wipes both log tables. The caller confirms destructive intent first. */
    suspend fun clearAll() = coroutineScope {
        val dns = async { dnsDao.clearAll() }
        val http = async { httpDao?.clearAll() }
        dns.await()
        http.await()
    }

    /**
     * Counts the records matching the same filters as [load], so the UI can show
     * "loaded x / total y" instead of leaving the user guessing whether more exist.
     */
    suspend fun count(
        source: RequestSource = RequestSource.ALL,
        status: RequestStatus = RequestStatus.ALL,
        query: String = "",
        sinceMillis: Long? = null
    ): Int = coroutineScope {
        val dns = async {
            if (source == RequestSource.HTTPS) {
                0
            } else {
                val filter = dnsFilter(status, query, sinceMillis)
                dnsDao.count(
                    SimpleSQLiteQuery("SELECT COUNT(*) FROM dns_log WHERE 1=1${filter.sql}", filter.args)
                )
            }
        }
        val http = async {
            val dao = httpDao
            if (source == RequestSource.DNS || dao == null) {
                0
            } else {
                val filter = httpFilter(status, query, sinceMillis)
                dao.count(
                    SimpleSQLiteQuery("SELECT COUNT(*) FROM http_request_log WHERE 1=1${filter.sql}", filter.args)
                )
            }
        }
        dns.await() + http.await()
    }

    private class SqlFilter(val sql: String, val args: Array<Any>)

    private fun dnsFilter(status: RequestStatus, query: String, sinceMillis: Long?): SqlFilter {
        val sql = StringBuilder()
        val args = mutableListOf<Any>()

        // Bounded first: `timestamp` is indexed, so the engine can narrow before the LIKE scans.
        if (sinceMillis != null) {
            sql.append(" AND timestamp >= ?")
            args.add(sinceMillis)
        }

        val trimmed = query.trim()

        if (trimmed.isNotEmpty()) {
            sql.append(" AND (queryName LIKE ? OR packageName LIKE ? OR message LIKE ?)")
            val pattern = "%${trimmed.lowercase()}%"
            args.add(pattern)
            args.add(pattern)
            args.add(pattern)
        }

        when (status) {
            RequestStatus.ALL -> Unit
            RequestStatus.PASSED -> {
                sql.append(" AND result = 'PASSED' AND queryType > 0 AND queryName NOT LIKE 'TCP %' AND queryName NOT LIKE 'UDP %' AND (message IS NULL OR (message NOT LIKE '%blocked_by=connection%' AND message NOT LIKE '%blocked_by=rewrite=%' AND message NOT LIKE '%matched rewrite rule%' AND message NOT LIKE '%复写%' AND message NOT LIKE '%覆写%'))")
            }
            RequestStatus.REWRITTEN -> {
                sql.append(" AND (result = 'REWRITTEN' OR message LIKE '%blocked_by=rewrite=%' OR message LIKE '%matched rewrite rule%' OR message LIKE '%复写%' OR message LIKE '%覆写%') AND queryType > 0 AND queryName NOT LIKE 'TCP %' AND queryName NOT LIKE 'UDP %' AND (message IS NULL OR message NOT LIKE '%blocked_by=connection%')")
            }
            RequestStatus.BLOCKED -> {
                sql.append(" AND (result = 'BLOCKED' OR message LIKE '%blocked_by=firewall:%') AND (message IS NULL OR (message NOT LIKE '%blocked_by=rewrite=%' AND message NOT LIKE '%matched rewrite rule%' AND message NOT LIKE '%复写%' AND message NOT LIKE '%覆写%'))")
            }
            RequestStatus.ERROR -> {
                sql.append(" AND result = 'ERROR' AND queryType > 0 AND queryName NOT LIKE 'TCP %' AND queryName NOT LIKE 'UDP %' AND (message IS NULL OR (message NOT LIKE '%blocked_by=connection%' AND message NOT LIKE '%blocked_by=rewrite=%' AND message NOT LIKE '%matched rewrite rule%' AND message NOT LIKE '%复写%' AND message NOT LIKE '%覆写%'))")
            }
            RequestStatus.BYPASSED -> {
                sql.append(" AND (queryType = 0 OR queryName LIKE 'TCP %' OR queryName LIKE 'UDP %' OR message LIKE '%blocked_by=connection%') AND (message IS NULL OR message NOT LIKE '%blocked_by=firewall:%')")
            }
        }

        return SqlFilter(sql.toString(), args.toTypedArray())
    }

    private fun httpFilter(status: RequestStatus, query: String, sinceMillis: Long?): SqlFilter {
        val sql = StringBuilder()
        val args = mutableListOf<Any>()

        // Bounded first: `timestamp` is indexed, so the engine can narrow before the LIKE scans.
        if (sinceMillis != null) {
            sql.append(" AND timestamp >= ?")
            args.add(sinceMillis)
        }

        val trimmed = query.trim()

        if (trimmed.isNotEmpty()) {
            sql.append(" AND (authority LIKE ? OR packageName LIKE ? OR matchedRule LIKE ?)")
            val pattern = "%${trimmed.lowercase()}%"
            args.add(pattern)
            args.add(pattern)
            args.add(pattern)
        }

        when (status) {
            RequestStatus.ALL -> Unit
            RequestStatus.PASSED -> {
                sql.append(" AND outcome = 'allowed'")
            }
            RequestStatus.REWRITTEN -> {
                sql.append(" AND outcome = 'rewritten'")
            }
            RequestStatus.BLOCKED -> {
                sql.append(" AND outcome IN ('blocked', 'invalid')")
            }
            RequestStatus.ERROR -> {
                sql.append(" AND outcome NOT IN ('allowed', 'rewritten', 'blocked', 'invalid', 'passthrough', 'bypassed', 'unsupported_protocol', 'resource_bypass')")
            }
            RequestStatus.BYPASSED -> {
                sql.append(" AND outcome IN ('passthrough', 'bypassed', 'unsupported_protocol', 'resource_bypass')")
            }
        }

        return SqlFilter(sql.toString(), args.toTypedArray())
    }
}

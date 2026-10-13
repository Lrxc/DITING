package com.haoze.diting.data.dao

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.Query
import androidx.room.RawQuery
import androidx.sqlite.db.SupportSQLiteQuery
import com.haoze.diting.data.entity.HttpRequestLogEntity

@Dao
interface HttpRequestLogDao {
    @Insert
    suspend fun insertAll(entities: List<HttpRequestLogEntity>)

    @RawQuery
    suspend fun queryList(query: SupportSQLiteQuery): List<HttpRequestLogEntity>

    @RawQuery
    suspend fun count(query: SupportSQLiteQuery): Int

    @Query("SELECT * FROM http_request_log ORDER BY timestamp DESC LIMIT :limit")
    suspend fun recent(limit: Int): List<HttpRequestLogEntity>

    @Query("SELECT outcome, COUNT(*) AS count FROM http_request_log WHERE timestamp >= :since GROUP BY outcome")
    suspend fun dailyStats(since: Long): List<HttpDailyStatRow>

    @Query("SELECT COUNT(*) FROM http_request_log WHERE timestamp >= :since")
    suspend fun countSince(since: Long): Int

    @Query("""
        SELECT blockSubscriptionId AS subscriptionId, COUNT(*) AS hits
        FROM http_request_log
        WHERE timestamp >= :since
            AND outcome = :blockedOutcome
            AND blockSubscriptionId IS NOT NULL
        GROUP BY blockSubscriptionId
    """)
    suspend fun subscriptionInterceptionStats(
        since: Long,
        blockedOutcome: String
    ): List<SubscriptionInterceptionStatRow>

    @Query("""
        SELECT packageName AS packageName, COUNT(*) AS hits
        FROM http_request_log
        WHERE timestamp >= :since
            AND outcome = :blockedOutcome
        GROUP BY packageName
    """)
    suspend fun blockedCountByPackage(
        since: Long,
        blockedOutcome: String
    ): List<PackageBlockedStatRow>

    @Query("""
        SELECT * FROM http_request_log
        WHERE packageName = :packageName
            AND outcome = :blockedOutcome
            AND timestamp >= :since
        ORDER BY timestamp DESC
        LIMIT :limit
    """)
    suspend fun blockedLogsByPackage(
        packageName: String,
        since: Long,
        blockedOutcome: String,
        limit: Int
    ): List<HttpRequestLogEntity>

    @Query("DELETE FROM http_request_log WHERE timestamp < :before")
    suspend fun deleteBefore(before: Long)

    @Query("DELETE FROM http_request_log")
    suspend fun clearAll()
}

data class HttpDailyStatRow(
    val outcome: String,
    val count: Int
)

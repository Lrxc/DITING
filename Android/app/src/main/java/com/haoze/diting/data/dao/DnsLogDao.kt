package com.haoze.diting.data.dao

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.Query
import androidx.room.RawQuery
import androidx.sqlite.db.SupportSQLiteQuery
import com.haoze.diting.data.entity.DnsLogEntity

@Dao
interface DnsLogDao {
    @Insert
    suspend fun insert(entity: DnsLogEntity)

    @Insert
    suspend fun insertAll(entities: List<DnsLogEntity>)

    @Query("DELETE FROM dns_log WHERE timestamp < :before")
    suspend fun deleteBefore(before: Long)

    @Query("DELETE FROM dns_log")
    suspend fun clearAll()

    @RawQuery
    suspend fun queryList(query: SupportSQLiteQuery): List<DnsLogEntity>

    @RawQuery
    suspend fun count(query: SupportSQLiteQuery): Int

    @Query("SELECT result, cached, COUNT(*) as count FROM dns_log WHERE timestamp >= :since GROUP BY result, cached")
    suspend fun dailyStats(since: Long): List<DailyStatRow>

    @Query("SELECT COUNT(*) FROM dns_log WHERE timestamp >= :since")
    suspend fun countSince(since: Long): Int

    @Query("""
        SELECT blockSubscriptionId AS subscriptionId, COUNT(*) AS hits
        FROM dns_log
        WHERE timestamp >= :since
            AND result = :blockedResult
            AND blockSubscriptionId IS NOT NULL
        GROUP BY blockSubscriptionId
    """)
    suspend fun subscriptionInterceptionStats(
        since: Long,
        blockedResult: String
    ): List<SubscriptionInterceptionStatRow>

    @Query("""
        SELECT packageName AS packageName, COUNT(*) AS hits
        FROM dns_log
        WHERE timestamp >= :since
            AND result = :blockedResult
            AND packageName IS NOT NULL
        GROUP BY packageName
    """)
    suspend fun blockedCountByPackage(
        since: Long,
        blockedResult: String
    ): List<PackageBlockedStatRow>

    @Query("""
        SELECT packageName AS packageName, COUNT(*) AS hits
        FROM dns_log
        WHERE timestamp >= :since
            AND packageName IS NOT NULL
        GROUP BY packageName
    """)
    suspend fun requestCountByPackage(since: Long): List<PackageBlockedStatRow>
}

data class DailyStatRow(
    val result: String,
    val cached: Boolean,
    val count: Int
)

data class SubscriptionInterceptionStatRow(
    val subscriptionId: Long,
    val hits: Int
)

data class PackageBlockedStatRow(
    val packageName: String,
    val hits: Int
)

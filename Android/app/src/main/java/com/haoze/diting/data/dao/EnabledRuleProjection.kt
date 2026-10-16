package com.haoze.diting.data.dao

/** Projection of an enabled rule (rule + source + modifier fields) shared by the allow/block rule caches. */
data class EnabledRule(
    val pattern: String,
    val source: String,
    val important: Boolean = false,
    val appScope: String? = null,
    val appInverted: Boolean = false,
    val isWildcard: Boolean = false,
    val denyallow: String? = null,
    val isRegex: Boolean = false,
    val dnsType: String? = null
)

data class EnabledRuleKeyset(
    val id: Long,
    val pattern: String,
    val source: String,
    val important: Boolean = false,
    val appScope: String? = null,
    val appInverted: Boolean = false,
    val isWildcard: Boolean = false,
    val denyallow: String? = null,
    val isRegex: Boolean = false,
    val dnsType: String? = null
) {
    fun toEnabledRule(): EnabledRule = EnabledRule(
        pattern = pattern,
        source = source,
        important = important,
        appScope = appScope,
        appInverted = appInverted,
        isWildcard = isWildcard,
        denyallow = denyallow,
        isRegex = isRegex,
        dnsType = dnsType
    )
}

/**
 * Enabled rule count attributed to a single rule source. Subscription rules store `sub_<id>`
 * as their source, so this is what lets the UI show how many rules a subscription currently
 * contributes without loading the rules themselves.
 */
data class SourceRuleCount(
    val source: String,
    val ruleCount: Int
)

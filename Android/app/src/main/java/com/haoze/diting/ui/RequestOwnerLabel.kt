package com.haoze.diting.ui

import androidx.compose.runtime.Composable

/**
 * Display name for a request whose owner could not be resolved to an installed app: either a
 * tunnel marker (`uid:<n>` / `unknown` / `RootProxy`) or nothing at all — a package that was
 * recorded but can no longer be resolved, e.g. an uninstalled app.
 * Composable because the label has to go through [localizedText] for the English locale.
 */
@Composable
fun requestOwnerLabel(owner: RequestOwner?): String = when (owner) {
    is RequestOwner.Uid -> localizedText("系统") + " (uid ${owner.uid})"
    RequestOwner.RootProxy -> localizedText("本机请求")
    // "no owner recorded" and "recorded owner no longer resolves" read the same to the user.
    RequestOwner.Unknown, null -> localizedText("未知应用")
}

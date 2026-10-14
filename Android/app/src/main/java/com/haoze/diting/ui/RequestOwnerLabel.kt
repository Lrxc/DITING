package com.haoze.diting.ui

import androidx.compose.runtime.Composable

/**
 * Display name for a request whose owner is a tunnel marker rather than a real package.
 * Composable because the label has to go through [localizedText] for the English locale.
 */
@Composable
fun requestOwnerLabel(owner: RequestOwner?): String = when (owner) {
    is RequestOwner.Uid -> localizedText("系统") + " (uid ${owner.uid})"
    RequestOwner.Unknown -> localizedText("未知应用")
    RequestOwner.RootProxy -> localizedText("本机请求")
    null -> ""
}

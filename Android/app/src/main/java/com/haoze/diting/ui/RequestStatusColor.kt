package com.haoze.diting.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import com.haoze.diting.data.RequestStatus

/**
 * Status colour shared by the request log rows and the detail sheet. Keeping it in one place
 * stops the list and the dialog from drifting apart when the palette changes.
 */
@Composable
fun requestStatusColor(status: RequestStatus): Color = when (status) {
    RequestStatus.BLOCKED,
    RequestStatus.ERROR -> MaterialTheme.colorScheme.error

    RequestStatus.BYPASSED -> MaterialTheme.colorScheme.onSurfaceVariant
    RequestStatus.REWRITTEN -> MaterialTheme.colorScheme.tertiary
    else -> MaterialTheme.colorScheme.primary
}

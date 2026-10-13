package com.haoze.diting.ui.components

import android.content.Context
import android.util.LruCache
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Android
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.core.graphics.drawable.toBitmap

private const val APP_ICON_CACHE_SIZE = 128
private const val APP_ICON_PIXEL_SIZE = 96

/** Process-wide cache of launcher icons decoded for list rows. */
private val appIconCache = LruCache<String, ImageBitmap>(APP_ICON_CACHE_SIZE)

/**
 * Renders the launcher icon of [packageName], or nothing when the package is
 * unknown to the system or its icon cannot be loaded.
 */
@Composable
fun AppIconImage(
    packageName: String?,
    size: Dp = 24.dp,
    modifier: Modifier = Modifier
) {
    if (!packageName.isNullOrBlank()) {
        val context = LocalContext.current
        val icon = remember(packageName) { loadAppIcon(context, packageName) }
        if (icon != null) {
            Image(
                bitmap = icon,
                contentDescription = null,
                modifier = modifier.size(size)
            )
        }
    }
}

private fun loadAppIcon(context: Context, packageName: String): ImageBitmap? {
    appIconCache.get(packageName)?.let { return it }
    val appContext = context.applicationContext
    return runCatching {
        appContext.packageManager.getApplicationIcon(packageName)
            .toBitmap(APP_ICON_PIXEL_SIZE, APP_ICON_PIXEL_SIZE)
            .asImageBitmap()
    }.getOrNull()?.also { appIconCache.put(packageName, it) }
}

/**
 * Neutral placeholder for entries whose owner could not be resolved to a package (for example
 * a connection attributed to uid 0). Keeps list rows visually aligned with icon-bearing rows.
 */
@Composable
fun SystemAppPlaceholder(
    size: Dp = 24.dp,
    modifier: Modifier = Modifier
) {
    Box(
        modifier = modifier
            .size(size)
            .clip(RoundedCornerShape(size / 4))
            .background(MaterialTheme.colorScheme.surfaceVariant),
        contentAlignment = Alignment.Center
    ) {
        Icon(
            imageVector = Icons.Default.Android,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.size(size * 0.6f)
        )
    }
}

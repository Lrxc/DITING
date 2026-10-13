package com.haoze.diting.ui.components

import android.content.Context
import android.util.LruCache
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
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

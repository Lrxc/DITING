package com.haoze.diting.ui.components

import android.content.Context
import android.graphics.drawable.Drawable
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
import androidx.compose.runtime.getValue
import androidx.compose.runtime.produceState
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.core.graphics.drawable.toBitmap
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

private const val APP_ICON_CACHE_SIZE = 128
private const val APP_ICON_PIXEL_SIZE = 96

/**
 * Process-wide cache of icons decoded for list rows. Keyed by package name for [AppIconImage],
 * and by a caller-provided key for [DrawableIconImage].
 */
private val appIconCache = LruCache<String, ImageBitmap>(APP_ICON_CACHE_SIZE)

/**
 * Renders the launcher icon of [packageName], falling back to [fallback] when it cannot be
 * resolved.
 *
 * A cache hit renders in the same frame; a miss is decoded off the main thread, so scrolling a
 * long list never blocks on PackageManager IPC + Bitmap conversion.
 */
@Composable
fun AppIconImage(
    packageName: String?,
    size: Dp = 24.dp,
    modifier: Modifier = Modifier,
    fallback: (@Composable () -> Unit)? = null
) {
    if (packageName.isNullOrBlank()) {
        fallback?.invoke()
        return
    }
    val context = LocalContext.current
    val icon by produceState(initialValue = appIconCache.get(packageName), key1 = packageName) {
        if (value == null) {
            value = loadIconFromPackage(context.applicationContext, packageName)
        }
    }
    if (icon != null) {
        Image(bitmap = icon, contentDescription = null, modifier = modifier.size(size))
    } else {
        fallback?.invoke()
    }
}

/**
 * Same as [AppIconImage] for callers that already hold a [Drawable] (bulk app pickers load icons
 * in one pass). The Drawable to Bitmap conversion also happens off the main thread, and
 * [cacheKey] must be stable per icon — a package name works well.
 */
@Composable
fun DrawableIconImage(
    drawable: Drawable?,
    cacheKey: String,
    size: Dp = 24.dp,
    modifier: Modifier = Modifier,
    fallback: (@Composable () -> Unit)? = null
) {
    if (drawable == null) {
        fallback?.invoke()
        return
    }
    val icon by produceState(initialValue = appIconCache.get(cacheKey), key1 = cacheKey) {
        if (value == null) {
            value = decodeDrawable(drawable, cacheKey)
        }
    }
    if (icon != null) {
        Image(bitmap = icon, contentDescription = null, modifier = modifier.size(size))
    } else {
        fallback?.invoke()
    }
}

private suspend fun loadIconFromPackage(context: Context, packageName: String): ImageBitmap? =
    withContext(Dispatchers.IO) {
        runCatching {
            context.packageManager.getApplicationIcon(packageName)
                .toBitmap(APP_ICON_PIXEL_SIZE, APP_ICON_PIXEL_SIZE)
                .asImageBitmap()
        }.getOrNull()?.also { appIconCache.put(packageName, it) }
    }

private suspend fun decodeDrawable(drawable: Drawable, cacheKey: String): ImageBitmap? =
    withContext(Dispatchers.Default) {
        runCatching {
            drawable.toBitmap(APP_ICON_PIXEL_SIZE, APP_ICON_PIXEL_SIZE).asImageBitmap()
        }.getOrNull()?.also { appIconCache.put(cacheKey, it) }
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

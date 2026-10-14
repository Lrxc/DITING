package com.haoze.diting.ui.dashboard

import java.text.NumberFormat
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/**
 * Formatters live per thread: both are called from list rows, and allocating a fresh
 * NumberFormat/SimpleDateFormat on every recomposition shows up in scroll profiles.
 */
private val countFormat = ThreadLocal.withInitial {
    NumberFormat.getIntegerInstance(Locale.CHINA)
}

private val clockFormat = ThreadLocal.withInitial {
    SimpleDateFormat("HH:mm:ss", Locale.CHINA)
}

fun formatCount(value: Int): String = countFormat.get()?.format(value) ?: value.toString()

fun formatClockTime(millis: Long): String {
    if (millis <= 0L) return "无"
    return clockFormat.get()?.format(Date(millis)).orEmpty()
}

package com.haoze.diting.ui.components

import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier

/**
 * Horizontal chip row shared by the request log screens. Generic so one row serves the source
 * switch, the time window, and any future filter without duplicating the layout.
 */
@Composable
fun <T> RequestFilterRow(
    values: List<T>,
    selected: T,
    label: (T) -> String,
    select: (T) -> Unit
) {
    RuleFilterChipRow(
        filters = values,
        selectedFilter = selected,
        onSelect = select,
        labelKeyOf = label,
        modifier = Modifier.padding(horizontal = SettingsCardMargin)
    )
}

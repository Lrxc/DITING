package com.haoze.diting.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Save
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.haoze.diting.ui.components.RuleListEmptyState
import com.haoze.diting.ui.components.RuleSearchField
import com.haoze.diting.ui.components.SettingsActionButton
import com.haoze.diting.ui.components.SettingsCardMargin
import com.haoze.diting.ui.components.SettingsInfoText
import com.haoze.diting.ui.components.SettingsScaffold
import com.haoze.diting.ui.components.SettingsSurfaceItem
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import java.util.Locale

/**
 * Shared skeleton for the installed-app picker screen: debounced search +
 * filter/sort menu + filtering pipeline + list + save FAB.
 * In single-select mode the selection is defined as 0 or 1 package names;
 * the caller converts it into a single package name.
 */
@Composable
internal fun AppPickerScreen(
    title: String,
    infoText: String,
    apps: List<InstalledApp>,
    selectedPackages: Set<String>,
    onSelectedPackagesChange: (Set<String>) -> Unit,
    initialFilter: AppListFilter,
    initialSort: AppListSort,
    onFilterChanged: (AppListFilter) -> Unit,
    onSortChanged: (AppListSort) -> Unit,
    showSelectionActions: Boolean,
    isDirty: Boolean,
    onSave: () -> Unit,
    onBack: () -> Unit,
    singleSelect: Boolean = false,
    headerContent: @Composable () -> Unit = {},
    onReload: (() -> Unit)? = null,
) {
    var filter by remember { mutableStateOf(initialFilter) }
    var sort by remember { mutableStateOf(initialSort) }
    var query by remember { mutableStateOf("") }
    var debouncedQuery by remember { mutableStateOf("") }
    var visibleApps by remember { mutableStateOf(emptyList<InstalledApp>()) }
    // Frozen at first composition: the list is grouped by the selection the user arrived with,
    // so toggling an app no longer makes it jump to the top while they work down the list.
    // The grouping refreshes the next time the screen opens.
    val initialSelection = remember { selectedPackages }

    LaunchedEffect(query) { delay(250); debouncedQuery = query }

    LaunchedEffect(apps, filter, sort, debouncedQuery, selectedPackages, initialSelection) {
        val normalizedQuery = debouncedQuery.trim().lowercase(Locale.ROOT)
        visibleApps = withContext(Dispatchers.Default) {
            apps.filter { app ->
                (filter == AppListFilter.ALL ||
                    (filter == AppListFilter.USER && !app.isSystem) ||
                    (filter == AppListFilter.SYSTEM && app.isSystem) ||
                    (filter == AppListFilter.SELECTED && app.packageName in selectedPackages)) &&
                    (normalizedQuery.isEmpty() || app.normalizedLabel.contains(normalizedQuery) ||
                        app.normalizedPackageName.contains(normalizedQuery))
            }.sortedWith(
                // Selected apps float to the top; the chosen sort order applies within each group.
                compareByDescending<InstalledApp> { it.packageName in initialSelection }
                    .then(sort.comparator)
            )
        }
    }

    // Leaving the screen discards pending edits; only the save button commits them.
    BackHandler {
        onBack()
    }

    SettingsScaffold(
        title = title,
        onBack = onBack,
        actions = {
            AppListOverflowMenu(
                filter = filter,
                sort = sort,
                onSelectAll = { onSelectedPackagesChange(selectedPackages + visibleApps.mapTo(mutableSetOf()) { it.packageName }) },
                onClear = { onSelectedPackagesChange(emptySet()) },
                onInvert = {
                    val visiblePackageNames = visibleApps.mapTo(mutableSetOf()) { it.packageName }
                    onSelectedPackagesChange(selectedPackages - visiblePackageNames + (visiblePackageNames - selectedPackages))
                },
                onFilterChange = {
                    filter = it
                    onFilterChanged(it)
                },
                onSortChange = {
                    sort = it
                    onSortChanged(it)
                },
                showSelectionActions = showSelectionActions
            )
        }
    ) { innerPadding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
        ) {
            Column(
                modifier = Modifier.fillMaxSize(),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                SettingsInfoText(
                    text = infoText,
                    modifier = Modifier.padding(top = 8.dp)
                )
                headerContent()
                RuleSearchField(
                    value = query,
                    onValueChange = { query = it },
                    placeholder = localizedText("搜索应用或包名"),
                    modifier = Modifier.padding(horizontal = SettingsCardMargin)
                )
                if (visibleApps.isEmpty()) {
                    Column(
                        modifier = Modifier.padding(horizontal = SettingsCardMargin),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        RuleListEmptyState(
                            message = localizedText(if (query.isNotEmpty()) "未找到匹配应用" else "暂无应用"),
                            modifier = Modifier.fillMaxWidth()
                        )
                        // Offer a manual reload only when the source list itself is empty; an
                        // empty filter/search result is not a loading failure.
                        if (onReload != null && query.isEmpty() && apps.isEmpty()) {
                            SettingsActionButton(
                                onClick = onReload,
                                modifier = Modifier.padding(top = 8.dp)
                            ) {
                                Text(localizedText("重新加载"))
                            }
                        }
                    }
                } else {
                    LazyColumn(
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f),
                        contentPadding = PaddingValues(bottom = 80.dp),
                        verticalArrangement = Arrangement.spacedBy(2.dp)
                    ) {
                        itemsIndexed(visibleApps, key = { _, app -> app.packageName }) { index, app ->
                            SettingsSurfaceItem(
                                index = index,
                                itemCount = visibleApps.size,
                                modifier = Modifier.padding(horizontal = SettingsCardMargin)
                            ) {
                                if (singleSelect) {
                                    InstalledAppRadioItem(
                                        app = app,
                                        selected = app.packageName in selectedPackages,
                                        onSelected = { onSelectedPackagesChange(setOf(app.packageName)) }
                                    )
                                } else {
                                    InstalledAppCheckboxItem(
                                        app = app,
                                        checked = app.packageName in selectedPackages,
                                        onCheckedChange = { checked ->
                                            onSelectedPackagesChange(
                                                if (checked) selectedPackages + app.packageName
                                                else selectedPackages - app.packageName
                                            )
                                        }
                                    )
                                }
                            }
                        }
                    }
                }
            }

            FloatingActionButton(
                onClick = {
                    if (isDirty) {
                        onSave()
                    }
                    onBack()
                },
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(16.dp),
                containerColor = MaterialTheme.colorScheme.primary,
                contentColor = MaterialTheme.colorScheme.onPrimary
            ) {
                Icon(Icons.Filled.Save, contentDescription = localizedText("保存"))
            }
        }
    }
}

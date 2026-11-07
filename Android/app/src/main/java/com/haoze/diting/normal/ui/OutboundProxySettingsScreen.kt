package com.haoze.diting.normal.ui

import com.haoze.diting.ui.*
import com.haoze.diting.ui.dashboard.*
import android.widget.Toast
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Apps
import androidx.compose.material.icons.filled.Save
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import com.haoze.diting.ui.components.SettingsActionButton
import com.haoze.diting.ui.components.SettingsCornerShape
import com.haoze.diting.ui.components.SettingsInfoText
import com.haoze.diting.ui.components.SettingsItem
import com.haoze.diting.ui.components.SettingsSwitchItem
import com.haoze.diting.ui.components.SettingsScaffold
import com.haoze.diting.ui.components.SettingsSurfaceItem
import com.haoze.diting.ui.components.SettingsSurfaceGroup
import com.haoze.diting.ui.components.SettingsCardMargin
import com.haoze.diting.ui.settings.OutboundProxySettingsStore
import com.haoze.diting.ui.settings.OutboundProxyTester
import com.haoze.diting.normal.DnsVpnService
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OutboundProxySettingsScreen(
    onBack: () -> Unit,
    onSelectApp: () -> Unit,
    selectedAppsOverride: Pair<Boolean, List<String>>? = null
) {
    val context = LocalContext.current
    var draft by remember { mutableStateOf(OutboundProxySettingsStore.getOutboundProxyConfig(context)) }
    var portText by remember { mutableStateOf(draft.port.toString()) }
    var proxyStatus by remember { mutableStateOf(OutboundProxySettingsStore.getOutboundProxyStatus(context)) }
    var proxyProbe by remember { mutableStateOf<ProxyProbe?>(null) }

    // Probe the configured proxy as soon as the page shows a usable address
    // and port, regardless of whether the feature switch is on.
    LaunchedEffect(draft.host, portText, draft.protocol, draft.username, draft.password) {
        val probePort = portText.toIntOrNull() ?: 0
        if (draft.host.isBlank() || probePort !in 1..65535) {
            proxyProbe = null
            return@LaunchedEffect
        }
        proxyProbe = ProxyProbe.Testing
        delay(400)
        proxyProbe = withContext(Dispatchers.IO) {
            when (val result = OutboundProxyTester.test(
                draft.protocol,
                draft.host,
                probePort,
                draft.username,
                draft.password
            )) {
                is OutboundProxyTester.Result.Available -> ProxyProbe.Available
                is OutboundProxyTester.Result.Unavailable -> ProxyProbe.Unavailable(result.reason)
            }
        }
    }

    LaunchedEffect(selectedAppsOverride) {
        selectedAppsOverride?.let { (_, packages) ->
            draft = draft.copy(proxyAppPackages = packages)
            // The picker re-reads the store when it opens, so mirror the
            // confirmed selection right away; otherwise re-entering it would
            // show the previously saved list instead of what was just picked.
            OutboundProxySettingsStore.setOutboundProxyAppPackages(context, packages)
        }
    }
    LaunchedEffect(draft.enabled) {
        while (draft.enabled) {
            proxyStatus = OutboundProxySettingsStore.getOutboundProxyStatus(context)
            delay(1000)
        }
    }

    var selectedLabel by remember { mutableStateOf("未选择") }
    LaunchedEffect(draft.proxyAppPackages) {
        val packages = draft.proxyAppPackages
        selectedLabel = when {
            packages.isEmpty() -> "未选择"
            packages.size == 1 -> runCatching {
                context.packageManager.getApplicationInfo(packages.first(), 0)
                    .loadLabel(context.packageManager).toString()
            }.getOrDefault(packages.first())
            else -> "已选择 ${packages.size} 个应用"
        }
    }
    fun setProxyEnabled(enabled: Boolean) {
        if (draft.enabled == enabled) return
        draft = draft.copy(enabled = enabled)
        OutboundProxySettingsStore.setOutboundProxyEnabled(context, enabled)
        val vpnRunning = DnsVpnService.isRunning(context)
        OutboundProxySettingsStore.setOutboundProxyStatus(
            context,
            when {
                !enabled -> "disabled"
                vpnRunning -> "connecting"
                // The proxy only works while the VPN is running, and enabling
                // the switch must not start the VPN by itself.
                else -> "waiting"
            },
            ""
        )
        if (vpnRunning) RuntimeDnsSettingsRefresher.refreshAppExclusionsIfRunning(context)
    }
    val successColor = if (isSystemInDarkTheme()) Color(0xFF6FD68B) else Color(0xFF1B7F3B)
    val statusColor = when {
        proxyProbe is ProxyProbe.Available -> successColor
        proxyProbe is ProxyProbe.Unavailable -> MaterialTheme.colorScheme.error
        proxyProbe == ProxyProbe.Testing -> null
        proxyStatus.first == "ready" -> successColor
        proxyStatus.first == "error" -> MaterialTheme.colorScheme.error
        else -> null
    }
    val statusText = when (val probe = proxyProbe) {
        ProxyProbe.Testing -> localizedText("正在检测代理")
        ProxyProbe.Available -> localizedText("代理可用")
        is ProxyProbe.Unavailable -> localizedText("代理不可用") + " · " +
            OutboundProxySettingsStore.formatProxyErrorMessage(context, probe.reason)
        null -> when (proxyStatus.first) {
            "connecting" -> localizedText("正在连接")
            "ready" -> localizedText("代理可用")
            "error" -> OutboundProxySettingsStore.formatProxyErrorMessage(context, proxyStatus.second)
            "waiting" -> localizedText("等待 VPN 启动")
            else -> localizedText("未运行")
        }
    }

    SettingsScaffold(title = localizedText("出站代理"), onBack = onBack) { padding ->
        Column(
            Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            SettingsSurfaceGroup(content = listOf(
                {
                    SettingsSwitchItem(
                        title = localizedText("启用出站代理功能"),
                        subtitle = statusText,
                        subtitleColor = statusColor,
                        checked = draft.enabled,
                        onCheckedChange = ::setProxyEnabled
                    )
                },
                {
                    SettingsItem(
                        title = localizedText("走代理的应用"),
                        subtitle = localizedText(selectedLabel),
                        onClick = onSelectApp
                    ) { Icon(Icons.Default.Apps, contentDescription = null) }
                }
            ))

            Column(
                Modifier.fillMaxWidth().padding(horizontal = SettingsCardMargin),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                    OutboundProxyProtocol.entries.forEachIndexed { index, protocol ->
                        SegmentedButton(
                            selected = draft.protocol == protocol,
                            onClick = { draft = draft.copy(protocol = protocol) },
                            shape = SegmentedButtonDefaults.itemShape(index, OutboundProxyProtocol.entries.size),
                            label = { Text(protocol.displayName) }
                        )
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = draft.host,
                        onValueChange = { draft = draft.copy(host = it.trim()) },
                        modifier = Modifier.weight(1f),
                        label = { Text(localizedText("地址")) },
                        singleLine = true,
                        shape = SettingsCornerShape
                    )
                    OutlinedTextField(
                        value = portText,
                        onValueChange = { value -> if (value.all(Char::isDigit)) portText = value },
                        modifier = Modifier.weight(0.65f),
                        label = { Text(localizedText("端口")) },
                        singleLine = true,
                        shape = SettingsCornerShape
                    )
                }
                OutlinedTextField(
                    value = draft.username,
                    onValueChange = { draft = draft.copy(username = it) },
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text(localizedText("账号（可选）")) },
                    singleLine = true,
                    shape = SettingsCornerShape
                )
                OutlinedTextField(
                    value = draft.password,
                    onValueChange = { draft = draft.copy(password = it) },
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text(localizedText("密码（可选）")) },
                    visualTransformation = PasswordVisualTransformation(),
                    singleLine = true,
                    shape = SettingsCornerShape
                )
            }

            SettingsInfoText(
                if (draft.protocol == OutboundProxyProtocol.HTTP) {
                    localizedText("勾选应用的流量经 HTTP CONNECT 转发；HTTP CONNECT 不支持 UDP，UDP 会被严格阻断，QUIC 可回退到 TCP。未勾选的应用保持直连，DNS 始终直连。请勿勾选提供代理端口的应用，否则会形成流量环路。")
                } else {
                    localizedText("勾选应用的流量经 SOCKS5 转发（UDP 通过 UDP ASSOCIATE）。未勾选的应用保持直连，DNS 始终直连。请勿勾选提供代理端口的应用（如 Clash），否则会形成流量环路。")
                }
            )
            SettingsActionButton(
                onClick = {
                    val savedConfig = draft.copy(port = portText.toIntOrNull() ?: 0)
                    val error = if (savedConfig.enabled) savedConfig.validationError(context) else null
                    if (error != null) {
                        context.showToast(error, Toast.LENGTH_LONG)
                    } else {
                        draft = savedConfig
                        OutboundProxySettingsStore.setOutboundProxyConfig(context, savedConfig)
                        OutboundProxySettingsStore.setOutboundProxyStatus(
                            context,
                            when {
                                !savedConfig.enabled -> "disabled"
                                DnsVpnService.isRunning(context) -> "connecting"
                                else -> "waiting"
                            },
                            ""
                        )
                        RuntimeDnsSettingsRefresher.refreshAppExclusionsIfRunning(context)
                        context.showToast("出站代理设置已保存", Toast.LENGTH_SHORT)
                    }
                },
                modifier = Modifier.fillMaxWidth().padding(horizontal = SettingsCardMargin)
            ) {
                Icon(Icons.Default.Save, contentDescription = null)
                Text(localizedText("保存"), Modifier.padding(start = 8.dp))
            }
        }
    }

}

@Composable
fun OutboundProxyAppsScreen(onBack: () -> Unit, onSave: (List<String>) -> Unit) {
    val context = LocalContext.current
    val initialPackages = remember { OutboundProxySettingsStore.getOutboundProxyAppPackages(context).toSet() }
    var selectedPackages by remember { mutableStateOf(initialPackages) }
    val access = rememberAppListAccessState { loadInstalledApps(context) }
    AppListDisclosureDialog(access)
    val apps = access.apps
    if (access.unavailable) {
        return SettingsScaffold(title = localizedText("选择走代理的应用"), onBack = onBack) {
            AppListUnavailableContent(Modifier.padding(it), access.retry)
        }
    }
    if (apps == null) {
        return SettingsScaffold(title = localizedText("选择走代理的应用"), onBack = onBack) {
            AppListLoadingContent(Modifier.padding(it))
        }
    }

    AppPickerScreen(
        title = localizedText("选择走代理的应用"),
        infoText = localizedText("勾选的应用流量经出站代理转发，未勾选的应用保持直连；DNS 解析始终直连。请勿勾选提供代理端口的应用，否则会形成流量环路。"),
        apps = apps,
        selectedPackages = selectedPackages,
        onSelectedPackagesChange = { selectedPackages = it },
        initialFilter = AppListFilter.USER,
        initialSort = AppListSort.LABEL_ASC,
        onFilterChanged = {},
        onSortChanged = {},
        showSelectionActions = false,
        isDirty = selectedPackages != initialPackages,
        onSave = { onSave(selectedPackages.sorted()) },
        onBack = onBack,
        singleSelect = false
    )
}

private sealed interface ProxyProbe {
    data object Testing : ProxyProbe
    data object Available : ProxyProbe
    data class Unavailable(val reason: String) : ProxyProbe
}

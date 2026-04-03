package com.lionheart.vpn.ui.screens
import androidx.compose.animation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.lionheart.vpn.R
import com.lionheart.vpn.data.ServerProfile
import com.lionheart.vpn.ui.components.*
import com.lionheart.vpn.viewmodel.VpnViewModel
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ServerDetailScreen(
    vm: VpnViewModel,
    serverId: String,
    onBack: () -> Unit,
    onNavigateSplitTunnel: (String) -> Unit
) {
    val servers by vm.servers.collectAsState()
    val server = servers.find { it.id == serverId }
    if (server == null) { LaunchedEffect(Unit) { onBack() }; return }
    var showDeleteDialog by remember { mutableStateOf(false) }
    var showUninstallDialog by remember { mutableStateOf(false) }
    var showDnsDialog by remember { mutableStateOf(false) }
    var showMtuDialog by remember { mutableStateOf(false) }
    var showRenameDialog by remember { mutableStateOf(false) }
    var showIpModeDialog by remember { mutableStateOf(false) }
    var showConnModeDialog by remember { mutableStateOf(false) }
    var showPingUrlDialog by remember { mutableStateOf(false) }
    fun update(transform: (ServerProfile) -> ServerProfile) = vm.updateServerSetting(serverId, transform)
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(server.name) },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, stringResource(R.string.back)) } },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.surface)
            )
        }
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState())) {
            Spacer(Modifier.height(8.dp))
            SettingsGroup(stringResource(R.string.server_section)) {
                SettingsClickable(stringResource(R.string.name), server.name, Icons.Filled.Label) { showRenameDialog = true }
                SettingsDivider()
                SettingsInfo(stringResource(R.string.ip_address), if (server.showServerIP) server.serverIP.ifBlank { "—" } else "••••••")
                SettingsDivider()
                SettingsInfo(stringResource(R.string.country), if (server.countryCode.isNotBlank()) countryCodeToName(server.countryCode) else "—")
                SettingsDivider()
                SettingsSwitch(stringResource(R.string.show_ip), stringResource(R.string.show_ip_desc), Icons.Filled.Visibility, server.showServerIP) {
                    update { it.copy(showServerIP = !it.showServerIP) }
                }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.connection_mode)) {
                SettingsClickable(
                    stringResource(R.string.mode_label),
                    when (server.connectionMode) {
                        "socks5" -> stringResource(R.string.mode_socks5_desc)
                        "mtproto" -> stringResource(R.string.mode_mtproto_desc)
                        else -> stringResource(R.string.mode_vpn_desc)
                    },
                    Icons.Filled.Tune
                ) { showConnModeDialog = true }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.network)) {
                SettingsClickable(stringResource(R.string.dns_server), if (server.adBlock) stringResource(R.string.adguard_block_ads) else server.dns, Icons.Filled.Dns) { showDnsDialog = true }
                SettingsDivider()
                SettingsClickable(stringResource(R.string.mtu_label), server.mtu.toString(), Icons.Filled.SettingsEthernet) { showMtuDialog = true }
                SettingsDivider()
                SettingsClickable(
                    stringResource(R.string.ip_protocol),
                    when (server.ipMode) {
                        "prefer_v6" -> stringResource(R.string.prefer_ipv6)
                        "only_v6" -> stringResource(R.string.only_ipv6)
                        "only_v4" -> stringResource(R.string.only_ipv4)
                        else -> stringResource(R.string.prefer_ipv4)
                    },
                    Icons.Filled.Language
                ) { showIpModeDialog = true }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.security)) {
                SettingsSwitch(
                    stringResource(R.string.ad_block),
                    stringResource(R.string.ad_block_desc),
                    Icons.Filled.Shield, server.adBlock
                ) { update { it.copy(adBlock = !it.adBlock) } }
                SettingsDivider()
                SettingsSwitch(
                    stringResource(R.string.kill_switch),
                    stringResource(R.string.kill_switch_desc),
                    Icons.Filled.GppBad, server.killSwitch
                ) { update { it.copy(killSwitch = !it.killSwitch) } }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.split_tunnel)) {
                SettingsSwitch(
                    stringResource(R.string.split_enable),
                    when {
                        !server.splitEnabled -> stringResource(R.string.split_all_vpn)
                        server.splitMode == "bypass" -> stringResource(R.string.split_bypass)
                        else -> stringResource(R.string.split_only)
                    },
                    Icons.Filled.CallSplit, server.splitEnabled
                ) { update { it.copy(splitEnabled = !it.splitEnabled) } }
                if (server.splitEnabled) {
                    SettingsDivider()
                    SettingsClickable(stringResource(R.string.split_mode), if (server.splitMode == "bypass") stringResource(R.string.split_mode_bypass) else stringResource(R.string.split_mode_only), Icons.Filled.SwapHoriz) {
                        update { it.copy(splitMode = if (it.splitMode == "bypass") "only" else "bypass") }
                    }
                    SettingsDivider()
                    SettingsClickable(stringResource(R.string.apps), stringResource(R.string.apps_selected, server.splitApps.size), Icons.Filled.Apps) { onNavigateSplitTunnel(serverId) }
                }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.diagnostics)) {
                SettingsClickable(stringResource(R.string.ping_url), server.pingUrl, Icons.Filled.Speed) { showPingUrlDialog = true }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.management)) {
                SettingsClickable(stringResource(R.string.remove_from_app), stringResource(R.string.remove_from_app_desc), Icons.Filled.RemoveCircleOutline) { showDeleteDialog = true }
                if (server.sshUser.isNotBlank()) {
                    SettingsDivider()
                    SettingsClickable(stringResource(R.string.remove_from_server), stringResource(R.string.remove_from_server_desc), Icons.Filled.DeleteForever) { showUninstallDialog = true }
                }
            }
            Spacer(Modifier.height(40.dp))
        }
    }
    if (showRenameDialog) {
        TextInputDialog(stringResource(R.string.server_name), server.name, { showRenameDialog = false }) { newName -> update { it.copy(name = newName) }; showRenameDialog = false }
    }
    if (showDnsDialog) {
        DnsDialog(server.dns, { showDnsDialog = false }) { dns -> update { it.copy(dns = dns) }; showDnsDialog = false }
    }
    if (showMtuDialog) {
        MtuDialog(server.mtu, { showMtuDialog = false }) { mtu -> update { it.copy(mtu = mtu) }; showMtuDialog = false }
    }
    if (showIpModeDialog) {
        RadioDialog(
            stringResource(R.string.ip_protocol),
            listOf(
                "prefer_v4" to stringResource(R.string.prefer_ipv4_default),
                "only_v4" to stringResource(R.string.only_ipv4),
                "prefer_v6" to stringResource(R.string.prefer_ipv6_desc),
                "only_v6" to stringResource(R.string.only_ipv6_desc)
            ),
            server.ipMode,
            { showIpModeDialog = false }
        ) { mode -> update { it.copy(ipMode = mode) }; showIpModeDialog = false }
    }
    if (showConnModeDialog) {
        RadioDialog(
            stringResource(R.string.connection_mode),
            listOf(
                "vpn" to stringResource(R.string.mode_vpn_desc),
                "socks5" to stringResource(R.string.mode_socks5_desc),
                "mtproto" to stringResource(R.string.mode_mtproto_desc)
            ),
            server.connectionMode,
            { showConnModeDialog = false }
        ) { mode -> update { it.copy(connectionMode = mode) }; showConnModeDialog = false }
    }
    if (showPingUrlDialog) {
        TextInputDialog(stringResource(R.string.ping_url), server.pingUrl, { showPingUrlDialog = false }) { url ->
            update { it.copy(pingUrl = url) }; showPingUrlDialog = false
        }
    }
    if (showDeleteDialog) {
        AlertDialog(
            onDismissRequest = { showDeleteDialog = false },
            title = { Text(stringResource(R.string.delete_server_confirm)) },
            text = { Text(stringResource(R.string.delete_server_msg, server.name)) },
            confirmButton = { TextButton(onClick = { vm.removeServer(serverId); showDeleteDialog = false; onBack() }) { Text(stringResource(R.string.delete), color = MaterialTheme.colorScheme.error) } },
            dismissButton = { TextButton(onClick = { showDeleteDialog = false }) { Text(stringResource(R.string.cancel)) } }
        )
    }
    if (showUninstallDialog) {
        UninstallDialog(server, { showUninstallDialog = false }) { pw -> showUninstallDialog = false; vm.uninstallServer(serverId, pw) { onBack() } }
    }
}
@Composable
private fun TextInputDialog(title: String, initial: String, onDismiss: () -> Unit, onConfirm: (String) -> Unit) {
    var text by remember { mutableStateOf(initial) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = { OutlinedTextField(text, { text = it }, singleLine = true, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp)) },
        confirmButton = { TextButton(onClick = { onConfirm(text.trim()) }) { Text(stringResource(R.string.ok)) } },
        dismissButton = { TextButton(onClick = onDismiss) { Text(stringResource(R.string.cancel)) } }
    )
}
@Composable
private fun DnsDialog(currentDns: String, onDismiss: () -> Unit, onConfirm: (String) -> Unit) {
    var text by remember { mutableStateOf(currentDns) }
    val presets = listOf("1.1.1.1" to "Cloudflare", "8.8.8.8" to "Google", "9.9.9.9" to "Quad9", "94.140.14.14" to "AdGuard")
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.dns_server)) },
        text = {
            Column {
                presets.forEach { (ip, name) ->
                    Row(Modifier.fillMaxWidth().padding(vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
                        RadioButton(selected = text == ip, onClick = { text = ip })
                        Spacer(Modifier.width(8.dp))
                        Column {
                            Text(name, style = MaterialTheme.typography.bodyMedium)
                            Text(ip, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                }
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(text, { text = it }, label = { Text(stringResource(R.string.custom_dns)) }, singleLine = true, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp))
            }
        },
        confirmButton = { TextButton(onClick = { onConfirm(text.trim()) }) { Text(stringResource(R.string.ok)) } },
        dismissButton = { TextButton(onClick = onDismiss) { Text(stringResource(R.string.cancel)) } }
    )
}
@Composable
private fun MtuDialog(currentMtu: Int, onDismiss: () -> Unit, onConfirm: (Int) -> Unit) {
    var text by remember { mutableStateOf(currentMtu.toString()) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.mtu_label)) },
        text = {
            Column {
                Text(stringResource(R.string.mtu_hint), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Spacer(Modifier.height(12.dp))
                OutlinedTextField(text, { text = it.filter { c -> c.isDigit() } }, label = { Text(stringResource(R.string.mtu_label)) }, singleLine = true, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp))
            }
        },
        confirmButton = { TextButton(onClick = { onConfirm(text.toIntOrNull() ?: 1500) }) { Text(stringResource(R.string.ok)) } },
        dismissButton = { TextButton(onClick = onDismiss) { Text(stringResource(R.string.cancel)) } }
    )
}
@Composable
private fun RadioDialog(title: String, options: List<Pair<String, String>>, current: String, onDismiss: () -> Unit, onSelect: (String) -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            Column {
                options.forEach { (value, label) ->
                    Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), verticalAlignment = Alignment.Top) {
                        RadioButton(selected = current == value, onClick = { onSelect(value) })
                        Spacer(Modifier.width(8.dp))
                        Text(label, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(top = 12.dp))
                    }
                }
            }
        },
        confirmButton = {},
        dismissButton = { TextButton(onClick = onDismiss) { Text(stringResource(R.string.close)) } }
    )
}
@Composable
private fun UninstallDialog(server: ServerProfile, onDismiss: () -> Unit, onConfirm: (String) -> Unit) {
    var password by remember { mutableStateOf("") }
    var showPw by remember { mutableStateOf(false) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.remove_from_vps)) },
        text = {
            Column {
                Text(stringResource(R.string.remove_from_vps_desc, server.serverIP), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Spacer(Modifier.height(12.dp))
                OutlinedTextField(password, { password = it }, label = { Text(stringResource(R.string.ssh_password_for, server.sshUser)) }, singleLine = true,
                    visualTransformation = if (showPw) VisualTransformation.None else PasswordVisualTransformation(),
                    trailingIcon = { IconButton(onClick = { showPw = !showPw }) { Icon(if (showPw) Icons.Filled.VisibilityOff else Icons.Filled.Visibility, "") } },
                    modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp))
            }
        },
        confirmButton = { TextButton(onClick = { onConfirm(password) }, enabled = password.isNotBlank()) { Text(stringResource(R.string.delete), color = MaterialTheme.colorScheme.error) } },
        dismissButton = { TextButton(onClick = onDismiss) { Text(stringResource(R.string.cancel)) } }
    )
}
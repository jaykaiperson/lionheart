package com.lionheart.vpn.ui.screens
import android.app.Activity
import android.app.LocaleManager
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.LocaleList
import android.provider.Settings
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.CallSplit
import androidx.compose.material.icons.filled.*
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lionheart.vpn.R
import com.lionheart.vpn.ui.components.*
import com.lionheart.vpn.viewmodel.APP_VERSION
import com.lionheart.vpn.viewmodel.VersionStatus
import com.lionheart.vpn.viewmodel.VpnStatus
import com.lionheart.vpn.viewmodel.VpnViewModel
import java.util.Locale
private fun isGrapheneOS(): Boolean {
    return try {
        val p = Runtime.getRuntime().exec("getprop ro.grapheneos.version")
        val result = p.inputStream.bufferedReader().readText().trim()
        p.waitFor()
        result.isNotBlank()
    } catch (_: Exception) { false }
}
private fun setAppLocale(context: Context, tag: String) {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
        val lm = context.getSystemService(LocaleManager::class.java)
        lm.applicationLocales = LocaleList.forLanguageTags(tag)
    } else {
        val locale = Locale.forLanguageTag(tag)
        Locale.setDefault(locale)
        val cfg = context.resources.configuration
        cfg.setLocale(locale)
        cfg.setLocales(LocaleList(locale))
        @Suppress("DEPRECATION")
        context.resources.updateConfiguration(cfg, context.resources.displayMetrics)
        context.getSharedPreferences("lh_prefs", Context.MODE_PRIVATE)
            .edit().putString("app_locale", tag).apply()
        (context as? Activity)?.recreate()
    }
}
private fun getDefaultLanguageForLocale(): String {
    val sysLang = Locale.getDefault().language
    val sysCountry = Locale.getDefault().country.uppercase()
    if (sysCountry == "BY") return "be"
    val cisCountries = setOf("RU", "KZ", "KG", "TJ", "UZ", "TM", "AM", "AZ", "GE", "MD", "UA")
    if (sysCountry in cisCountries || sysLang == "ru") return "ru"
    if (sysLang == "tt") return "tt"
    return "en"
}
data class LangOption(val code: String, val flagRes: Int, val native: String, val english: String)
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    vm: VpnViewModel,
    onNavigateSettings: () -> Unit,
    onNavigateLogs: () -> Unit,
    onNavigateSetup: () -> Unit,
    onNavigateServerDetail: (String) -> Unit
) {
    val context = LocalContext.current
    val status by vm.status.collectAsState()
    val connectionTime by vm.connectionTime.collectAsState()
    val txBytes by vm.txBytes.collectAsState()
    val rxBytes by vm.rxBytes.collectAsState()
    val serverIP by vm.serverIP.collectAsState()
    val turnServer by vm.turnServer.collectAsState()
    val smartKey by vm.smartKey.collectAsState()
    val countryCode by vm.countryCode.collectAsState()
    val servers by vm.servers.collectAsState()
    val activeServer by vm.activeServer.collectAsState()
    val pingMs by vm.pingMs.collectAsState()
    val txSpeed by vm.txSpeed.collectAsState()
    val rxSpeed by vm.rxSpeed.collectAsState()
    val versionStatus by vm.versionStatus.collectAsState()
    val languages = remember {
        listOf(
            LangOption("en", R.drawable.ic_flag_gb, "English", "English"),
            LangOption("ru", R.drawable.ic_flag_ru, "Русский", "Russian"),
            LangOption("be", R.drawable.ic_flag_by, "Беларуская", "Belarusian"),
            LangOption("tt", R.drawable.ic_flag_tt, "Татарча", "Tatar"),
        )
    }
    LaunchedEffect(Unit) {
        val prefs = context.getSharedPreferences("lh_prefs", Context.MODE_PRIVATE)
        if (!prefs.contains("locale_initialized")) {
            val defaultLang = getDefaultLanguageForLocale()
            setAppLocale(context, defaultLang)
            prefs.edit().putBoolean("locale_initialized", true).apply()
        }
    }
    var showServerSheet by remember { mutableStateOf(false) }
    var showUpdateDialog by remember { mutableStateOf(false) }
    var updatePassword by remember { mutableStateOf("") }
    var showLanguageSheet by remember { mutableStateOf(false) }
    var showGrapheneWarning by remember {
        mutableStateOf(
            isGrapheneOS() && context.getSharedPreferences("lh_prefs", Context.MODE_PRIVATE)
                .getBoolean("show_graphene_warning", true)
        )
    }
    val isOutdated = activeServer?.serverVersion?.let { v -> v.isNotBlank() && vm.isServerOutdated() } ?: false
    val vpnPermissionLauncher = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        if (result.resultCode == Activity.RESULT_OK) vm.connect()
    }
    if (showGrapheneWarning) {
        AlertDialog(onDismissRequest = {},
            icon = { Icon(Icons.Filled.Security, null, modifier = Modifier.size(32.dp)) },
            title = { Text(stringResource(R.string.graphene_title), style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold)) },
            text = { Text(stringResource(R.string.graphene_message), style = MaterialTheme.typography.bodyMedium) },
            confirmButton = { Button(onClick = { context.getSharedPreferences("lh_prefs", Context.MODE_PRIVATE).edit().putBoolean("show_graphene_warning", false).apply(); showGrapheneWarning = false }, shape = RoundedCornerShape(14.dp)) { Text(stringResource(R.string.graphene_ok)) } }
        )
    }
    if (showLanguageSheet) {
        ModalBottomSheet(onDismissRequest = { showLanguageSheet = false }) {
            Column(
                modifier = Modifier
                    .padding(horizontal = 24.dp)
                    .padding(bottom = 32.dp)
                    .navigationBarsPadding()
            ) {
                Text(stringResource(R.string.language_title), style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold))
                Spacer(Modifier.height(4.dp))
                Text(stringResource(R.string.language_subtitle), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Spacer(Modifier.height(16.dp))
                languages.forEach { lang ->
                    Card(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 3.dp)
                            .clip(RoundedCornerShape(16.dp))
                            .clickable { setAppLocale(context, lang.code); showLanguageSheet = false },
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerHigh),
                        shape = RoundedCornerShape(16.dp)
                    ) {
                        Row(Modifier.padding(14.dp), verticalAlignment = Alignment.CenterVertically) {
                            Image(
                                painter = painterResource(lang.flagRes),
                                contentDescription = lang.native,
                                modifier = Modifier.size(width = 28.dp, height = 21.dp)
                            )
                            Spacer(Modifier.width(16.dp))
                            Column(Modifier.weight(1f)) {
                                Text(lang.native, style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.Medium))
                                if (lang.english != lang.native) {
                                    Text(lang.english, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                }
                            }
                        }
                    }
                }
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                    Spacer(Modifier.height(12.dp))
                    OutlinedButton(
                        onClick = {
                            try { context.startActivity(Intent(Settings.ACTION_APP_LOCALE_SETTINGS).apply { data = android.net.Uri.parse("package:${context.packageName}") }) } catch (_: Exception) {}
                            showLanguageSheet = false
                        },
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(14.dp)
                    ) {
                        Icon(Icons.Filled.Settings, null, modifier = Modifier.size(18.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.system_language_settings))
                    }
                }
            }
        }
    }
    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Image(
                            painter = painterResource(R.drawable.ic_launcher_foreground),
                            contentDescription = null,
                            modifier = Modifier.size(52.dp).scale(1.4f)
                        )
                        Text(
                            "Lionheart",
                            style = MaterialTheme.typography.headlineMedium.copy(fontWeight = FontWeight.Bold)
                        )
                        Spacer(Modifier.width(8.dp))
                        Surface(
                            shape = RoundedCornerShape(8.dp),
                            color = MaterialTheme.colorScheme.primaryContainer
                        ) {
                            Text(
                                "v$APP_VERSION",
                                style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                            )
                        }
                    }
                },
                actions = { FilledIconButton(onClick = { showLanguageSheet = true }, colors = IconButtonDefaults.filledIconButtonColors(containerColor = MaterialTheme.colorScheme.surfaceContainerHigh), modifier = Modifier.size(40.dp)) { Icon(Icons.Filled.Translate, "Language", modifier = Modifier.size(18.dp)) }; Spacer(Modifier.width(8.dp)) },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.surface)
            )
        }
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState()), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
            Spacer(Modifier.weight(0.3f))
            Row(modifier = Modifier.fillMaxWidth().padding(horizontal = 24.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Card(modifier = Modifier.weight(1f).height(56.dp).clickable(onClick = onNavigateLogs), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerHigh), shape = RoundedCornerShape(16.dp)) { Row(Modifier.fillMaxSize().padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.Center) { Icon(Icons.Outlined.Terminal, null, modifier = Modifier.size(20.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.logs), style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold)) } }
                Card(modifier = Modifier.weight(1f).height(56.dp).clickable(onClick = onNavigateSettings), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerHigh), shape = RoundedCornerShape(16.dp)) { Row(Modifier.fillMaxSize().padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.Center) { Icon(Icons.Outlined.Settings, null, modifier = Modifier.size(20.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.settings), style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold)) } }
            }
            Spacer(Modifier.height(20.dp))
            if (servers.isNotEmpty()) {
                val cc = activeServer?.countryCode ?: countryCode
                AssistChip(onClick = { showServerSheet = true }, label = { Text(activeServer?.name ?: serverIP.ifBlank { stringResource(R.string.select_server) }, maxLines = 1, overflow = TextOverflow.Ellipsis) }, leadingIcon = { if (cc.isNotBlank()) CountryBadge(cc) else Icon(Icons.Filled.Cloud, null, modifier = Modifier.size(18.dp)) }, trailingIcon = { Icon(Icons.Filled.UnfoldMore, null, modifier = Modifier.size(18.dp)) }, shape = RoundedCornerShape(14.dp))
                activeServer?.let { srv ->
                    Row(Modifier.padding(top = 6.dp), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        if (srv.connectionMode != "vpn") { SuggestionChip(onClick = { onNavigateServerDetail(srv.id) }, label = { Text(if (srv.connectionMode == "socks5") stringResource(R.string.mode_socks5) else stringResource(R.string.mode_mtproto), style = MaterialTheme.typography.labelSmall) }, icon = { Icon(Icons.Filled.Tune, null, modifier = Modifier.size(14.dp)) }, shape = RoundedCornerShape(10.dp)) }
                        if (srv.splitEnabled) { SuggestionChip(onClick = { onNavigateServerDetail(srv.id) }, label = { Text("Split: ${srv.splitApps.size}", style = MaterialTheme.typography.labelSmall) }, icon = { Icon(Icons.AutoMirrored.Filled.CallSplit, null, modifier = Modifier.size(14.dp)) }, shape = RoundedCornerShape(10.dp)) }
                        if (srv.adBlock) { SuggestionChip(onClick = {}, label = { Text("AdBlock", style = MaterialTheme.typography.labelSmall) }, icon = { Icon(Icons.Filled.Shield, null, modifier = Modifier.size(14.dp)) }, shape = RoundedCornerShape(10.dp)) }
                        if (srv.killSwitch) { SuggestionChip(onClick = {}, label = { Text("KillSwitch", style = MaterialTheme.typography.labelSmall) }, icon = { Icon(Icons.Filled.GppBad, null, modifier = Modifier.size(14.dp)) }, shape = RoundedCornerShape(10.dp)) }
                    }
                }
                Spacer(Modifier.height(16.dp))
            }
            AnimatedVisibility(visible = status == VpnStatus.CONNECTED && countryCode.isNotBlank(), enter = fadeIn(tween(300)) + scaleIn(initialScale = 0.7f), exit = fadeOut(tween(200))) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) { CountryBadge(countryCode, large = true); Spacer(Modifier.height(8.dp)); Text(countryCodeToName(countryCode), style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.onSurfaceVariant); Spacer(Modifier.height(16.dp)) }
            }
            Spacer(Modifier.height(8.dp))
            ConnectButton(status = status, onClick = {
                if (isOutdated) { showUpdateDialog = true }
                else if (status == VpnStatus.DISCONNECTED || status == VpnStatus.ERROR) { val intent = vm.needsVpnPermission(); if (intent != null) vpnPermissionLauncher.launch(intent) else vm.connect() }
                else { vm.toggle() }
            })
            AnimatedVisibility(visible = isOutdated && status == VpnStatus.DISCONNECTED, enter = fadeIn() + expandVertically(), exit = fadeOut() + shrinkVertically()) {
                Card(Modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 8.dp), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer), shape = RoundedCornerShape(16.dp)) {
                    Column(Modifier.padding(16.dp)) { Text(stringResource(R.string.server_outdated_title), style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold), color = MaterialTheme.colorScheme.onErrorContainer); Spacer(Modifier.height(4.dp)); Text(stringResource(R.string.server_outdated_msg, activeServer?.serverVersion ?: "?", APP_VERSION), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onErrorContainer); Spacer(Modifier.height(12.dp)); Button(onClick = { showUpdateDialog = true }, colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error), shape = RoundedCornerShape(12.dp), modifier = Modifier.fillMaxWidth()) { Icon(Icons.Filled.SystemUpdateAlt, null, modifier = Modifier.size(18.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.update_server)) } }
                }
            }
            AnimatedVisibility(visible = versionStatus == VersionStatus.UPDATING) { Card(Modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 8.dp), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.tertiaryContainer), shape = RoundedCornerShape(16.dp)) { Row(Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically) { CircularProgressIndicator(modifier = Modifier.size(24.dp), strokeWidth = 3.dp); Spacer(Modifier.width(12.dp)); Text(stringResource(R.string.updating_server), style = MaterialTheme.typography.bodyMedium) } } }
            AnimatedVisibility(visible = versionStatus == VersionStatus.UPDATED) { Card(Modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 8.dp), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer), shape = RoundedCornerShape(16.dp)) { Row(Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically) { Icon(Icons.Filled.CheckCircle, null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(24.dp)); Spacer(Modifier.width(12.dp)); Text(stringResource(R.string.server_updated), style = MaterialTheme.typography.bodyMedium) } } }
            Spacer(Modifier.height(20.dp)); StatusText(status)
            AnimatedVisibility(visible = status == VpnStatus.CONNECTED) { Surface(shape = RoundedCornerShape(8.dp), color = MaterialTheme.colorScheme.surfaceContainerHigh, modifier = Modifier.padding(top = 12.dp)) { Text(formatDuration(connectionTime), style = MaterialTheme.typography.labelLarge.copy(letterSpacing = 2.sp), color = MaterialTheme.colorScheme.onSurface, modifier = Modifier.padding(horizontal = 12.dp, vertical = 4.dp)) } }
            Spacer(Modifier.height(24.dp))
            AnimatedVisibility(visible = status == VpnStatus.CONNECTED, enter = fadeIn(tween(400, 200)) + expandVertically(), exit = fadeOut(tween(150)) + shrinkVertically()) { Box(Modifier.padding(horizontal = 24.dp)) { CompactTrafficPill(txSpeed = txSpeed, rxSpeed = rxSpeed, txBytes = txBytes, rxBytes = rxBytes) } }
            AnimatedVisibility(visible = status == VpnStatus.CONNECTED, enter = fadeIn(tween(400, 300)) + expandVertically(), exit = fadeOut(tween(150)) + shrinkVertically()) {
                Card(Modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 8.dp), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow), shape = RoundedCornerShape(16.dp)) {
                    Row(Modifier.clickable { vm.measurePing() }.padding(16.dp), verticalAlignment = Alignment.CenterVertically) { Icon(Icons.Filled.Speed, null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(20.dp)); Spacer(Modifier.width(12.dp)); Column(Modifier.weight(1f)) { Text(stringResource(R.string.ping), style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium)); Text(activeServer?.pingUrl ?: "cp.cloudflare.com", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }; Text(when { pingMs < 0 -> "..."; pingMs == 0L -> stringResource(R.string.tap_to_ping); else -> stringResource(R.string.ms_suffix, pingMs.toInt()) }, style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.SemiBold), color = when { pingMs <= 0 -> MaterialTheme.colorScheme.onSurfaceVariant; pingMs < 100 -> MaterialTheme.colorScheme.primary; pingMs < 300 -> MaterialTheme.colorScheme.tertiary; else -> MaterialTheme.colorScheme.error }) }
                }
            }
            Spacer(Modifier.height(12.dp))
            Card(Modifier.fillMaxWidth().padding(horizontal = 24.dp), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow), shape = RoundedCornerShape(20.dp)) {
                Column(Modifier.padding(20.dp)) {
                    if (servers.isEmpty() && smartKey.isBlank()) {
                        Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.fillMaxWidth()) {
                            Box(Modifier.size(48.dp).clip(RoundedCornerShape(14.dp)).background(MaterialTheme.colorScheme.primaryContainer), contentAlignment = Alignment.Center) { Icon(Icons.Filled.Add, null, tint = MaterialTheme.colorScheme.onPrimaryContainer, modifier = Modifier.size(24.dp)) }
                            Spacer(Modifier.height(14.dp)); Text(stringResource(R.string.add_first_server), style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Medium)); Spacer(Modifier.height(6.dp))
                            Text(stringResource(R.string.add_first_server_desc), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center); Spacer(Modifier.height(16.dp))
                            Button(onClick = onNavigateSetup, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp)) { Icon(Icons.Filled.RocketLaunch, null, modifier = Modifier.size(18.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.add_server)) }
                        }
                    } else {
                        val showIP = activeServer?.showServerIP != false
                        InfoRow(stringResource(R.string.server_label), if (showIP) serverIP.ifBlank { "—" } else stringResource(R.string.hidden))
                        if (turnServer.isNotBlank()) { Spacer(Modifier.height(8.dp)); InfoRow(stringResource(R.string.turn_label), turnServer) }
                        Spacer(Modifier.height(8.dp)); InfoRow(stringResource(R.string.mode_label), when (activeServer?.connectionMode) { "socks5" -> stringResource(R.string.mode_socks5); "mtproto" -> stringResource(R.string.mode_mtproto); else -> stringResource(R.string.mode_vpn) })
                        Spacer(Modifier.height(8.dp)); InfoRow(stringResource(R.string.dns_label), if (activeServer?.adBlock == true) stringResource(R.string.adguard_no_ads) else (activeServer?.dns ?: "1.1.1.1"))
                        Spacer(Modifier.height(8.dp)); InfoRow(stringResource(R.string.mtu_label), (activeServer?.mtu ?: 1500).toString())
                        if (activeServer?.serverVersion?.isNotBlank() == true) { Spacer(Modifier.height(8.dp)); InfoRow(stringResource(R.string.server_version), "v${activeServer?.serverVersion}") }
                    }
                }
            }
            Spacer(Modifier.height(20.dp)); Text(stringResource(R.string.traffic_disguised), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f))
            Spacer(Modifier.weight(0.5f)); Spacer(Modifier.windowInsetsBottomHeight(WindowInsets.navigationBars)); Spacer(Modifier.height(16.dp))
        }
    }
    if (showServerSheet) { ModalBottomSheet(onDismissRequest = { showServerSheet = false }) { Column(Modifier.padding(horizontal = 24.dp)) { Text(stringResource(R.string.servers), style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold)); Spacer(Modifier.height(16.dp)); servers.forEach { server -> val isActive = server.id == (activeServer?.id ?: ""); Card(Modifier.fillMaxWidth().padding(vertical = 4.dp), colors = CardDefaults.cardColors(containerColor = if (isActive) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceContainerHigh), shape = RoundedCornerShape(16.dp)) { Row(Modifier.clickable { vm.selectServer(server.id); showServerSheet = false }.padding(16.dp), verticalAlignment = Alignment.CenterVertically) { if (server.countryCode.isNotBlank()) { CountryBadge(server.countryCode); Spacer(Modifier.width(14.dp)) }; Column(Modifier.weight(1f)) { Text(server.name, style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.Medium)); Text(if (server.showServerIP) server.serverIP else "••••", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }; IconButton(onClick = { showServerSheet = false; onNavigateServerDetail(server.id) }) { Icon(Icons.Outlined.Settings, stringResource(R.string.settings), modifier = Modifier.size(20.dp)) }; if (isActive) Icon(Icons.Filled.Check, null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(20.dp)) } } }; Spacer(Modifier.height(12.dp)); OutlinedButton(onClick = { showServerSheet = false; onNavigateSetup() }, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp)) { Icon(Icons.Filled.Add, null, modifier = Modifier.size(18.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.add_server)) }; Spacer(Modifier.height(32.dp)) } } }
    if (showUpdateDialog) { AlertDialog(onDismissRequest = { showUpdateDialog = false }, title = { Text(stringResource(R.string.update_confirm_title)) }, text = { Column { Text(stringResource(R.string.update_confirm_msg, activeServer?.serverVersion ?: "?", APP_VERSION), style = MaterialTheme.typography.bodyMedium); Spacer(Modifier.height(4.dp)); Text(stringResource(R.string.update_confirm_desc), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant); if (activeServer?.sshUser?.isNotBlank() == true) { Spacer(Modifier.height(12.dp)); OutlinedTextField(value = updatePassword, onValueChange = { updatePassword = it }, label = { Text(stringResource(R.string.ssh_password_for, activeServer?.sshUser ?: "")) }, singleLine = true, visualTransformation = PasswordVisualTransformation(), modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp)) } else { Spacer(Modifier.height(8.dp)); Text(stringResource(R.string.no_ssh_access), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error) } } }, confirmButton = { TextButton(onClick = { activeServer?.let { vm.updateServer(it.id, updatePassword) }; showUpdateDialog = false; updatePassword = "" }, enabled = updatePassword.isNotBlank() && activeServer?.sshUser?.isNotBlank() == true) { Text(stringResource(R.string.update_btn)) } }, dismissButton = { TextButton(onClick = { showUpdateDialog = false; updatePassword = "" }) { Text(stringResource(R.string.cancel)) } }) }
}
@Composable
private fun InfoRow(label: String, value: String) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
        Text(label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, style = MaterialTheme.typography.bodySmall.copy(fontWeight = FontWeight.Medium), color = MaterialTheme.colorScheme.onSurface, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.widthIn(max = 220.dp), textAlign = TextAlign.End)
    }
}
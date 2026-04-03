package com.lionheart.vpn.ui.screens
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lionheart.vpn.R
import com.lionheart.vpn.data.AppDisguise
import com.lionheart.vpn.ui.components.*
import com.lionheart.vpn.viewmodel.APP_VERSION
import com.lionheart.vpn.viewmodel.VpnViewModel
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(vm: VpnViewModel, onBack: () -> Unit, onScanQR: () -> Unit) {
    val context = LocalContext.current
    val autoConnect by vm.autoConnect.collectAsState()
    val bootConnect by vm.bootConnect.collectAsState()
    val theme by vm.theme.collectAsState()
    val activeServer by vm.activeServer.collectAsState()
    var showThemeDialog by remember { mutableStateOf(false) }
    var showResetDialog by remember { mutableStateOf(false) }
    var currentDisguise by remember { mutableStateOf(AppDisguise.getCurrent(context)) }
    val killSwitch = activeServer?.killSwitch ?: false
    Scaffold(
        topBar = { TopAppBar(title = { Text(stringResource(R.string.settings)) }, navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, stringResource(R.string.back)) } }, colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.surface)) }
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState())) {
            Spacer(Modifier.height(8.dp))
            SettingsGroup(stringResource(R.string.automation)) {
                SettingsSwitch(stringResource(R.string.auto_connect), stringResource(R.string.auto_connect_desc), Icons.Filled.FlashOn, autoConnect) { vm.setAutoConnect(it) }
                SettingsDivider()
                SettingsSwitch(stringResource(R.string.boot_connect), stringResource(R.string.boot_connect_desc), Icons.Filled.PhoneAndroid, bootConnect) { vm.setBootConnect(it) }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.security)) {
                SettingsSwitch(stringResource(R.string.kill_switch), stringResource(R.string.kill_switch_desc), Icons.Filled.GppBad, killSwitch) { enabled ->
                    activeServer?.let { srv -> vm.updateServerSetting(srv.id) { it.copy(killSwitch = enabled) } }
                }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.appearance)) {
                SettingsClickable(stringResource(R.string.theme), when (theme) { "dark" -> stringResource(R.string.theme_dark); "light" -> stringResource(R.string.theme_light); else -> stringResource(R.string.theme_system) }, Icons.Filled.DarkMode) { showThemeDialog = true }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.icon_color)) {
                Column(Modifier.padding(16.dp)) {
                    Text(stringResource(R.string.icon_color_desc), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Spacer(Modifier.height(16.dp))
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceEvenly) {
                        LionIconOption(isSelected = currentDisguise == "default", label = stringResource(R.string.icon_standard), bgColor = Color(0xFF103BB0), fgRes = R.drawable.ic_launcher_foreground, fgScale = 1.15f, onClick = { AppDisguise.apply(context, "default"); currentDisguise = "default" })
                        LionIconOption(isSelected = currentDisguise == "gold", label = stringResource(R.string.icon_gold), bgColor = Color(0xFF6E1319), fgRes = R.drawable.ic_lion_gold, fgScale = 1.15f, onClick = { AppDisguise.apply(context, "gold"); currentDisguise = "gold" })
                        LionIconOption(isSelected = currentDisguise == "red", label = stringResource(R.string.icon_red), bgColor = Color(0xFF1A1A2E), fgRes = R.drawable.ic_lion_red, fgScale = 1.15f, onClick = { AppDisguise.apply(context, "red"); currentDisguise = "red" })
                        LionIconOption(isSelected = currentDisguise == "blue", label = stringResource(R.string.icon_blue), bgColor = Color(0xFF0D1B2A), fgRes = R.drawable.ic_lion_blue, fgScale = 1.15f, onClick = { AppDisguise.apply(context, "blue"); currentDisguise = "blue" })
                        LionIconOption(isSelected = currentDisguise == "green", label = stringResource(R.string.icon_green), bgColor = Color(0xFF1B3A1B), fgRes = R.drawable.ic_lion_green, fgScale = 1.15f, onClick = { AppDisguise.apply(context, "green"); currentDisguise = "green" })
                        LionIconOption(isSelected = currentDisguise == "white", label = stringResource(R.string.icon_white), bgColor = Color(0xFF2C2C2C), fgRes = R.drawable.ic_lion_white, fgScale = 1.15f, onClick = { AppDisguise.apply(context, "white"); currentDisguise = "white" })
                    }
                }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.hide_app)) {
                Column(Modifier.padding(16.dp)) {
                    Text(stringResource(R.string.hide_app_desc), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Spacer(Modifier.height(16.dp))
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceEvenly) {
                        InlineDisguiseOption(isSelected = !AppDisguise.DISGUISE_OPTIONS.any { it.id == currentDisguise }, label = stringResource(R.string.hide_off), icon = Icons.Filled.Shield, bgColor = MaterialTheme.colorScheme.primary, iconTint = Color.White, onClick = { AppDisguise.apply(context, "default"); currentDisguise = "default" })
                        InlineDisguiseOption(isSelected = currentDisguise == "calculator", label = "Calc", icon = Icons.Filled.Calculate, bgColor = Color(0xFF37474F), iconTint = Color.White, onClick = { AppDisguise.apply(context, "calculator"); currentDisguise = "calculator" })
                        InlineDisguiseOption(isSelected = currentDisguise == "clock", label = "Clock", icon = Icons.Filled.Schedule, bgColor = Color(0xFF263238), iconTint = Color.White, onClick = { AppDisguise.apply(context, "clock"); currentDisguise = "clock" })
                        InlineDisguiseOption(isSelected = currentDisguise == "music", label = "Music", icon = Icons.Filled.MusicNote, bgColor = Color(0xFFE91E63), iconTint = Color.White, onClick = { AppDisguise.apply(context, "music"); currentDisguise = "music" })
                        InlineDisguiseOption(isSelected = currentDisguise == "notes", label = "Notes", icon = Icons.Filled.EditNote, bgColor = Color(0xFFFFF8E1), iconTint = Color(0xFF333333), onClick = { AppDisguise.apply(context, "notes"); currentDisguise = "notes" })
                        InlineDisguiseOption(isSelected = currentDisguise == "weather", label = "Weather", icon = Icons.Filled.WbSunny, bgColor = Color(0xFF42A5F5), iconTint = Color.White, onClick = { AppDisguise.apply(context, "weather"); currentDisguise = "weather" })
                    }
                }
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.protocol)) {
                SettingsInfo(stringResource(R.string.transport), "KCP over UDP"); SettingsDivider(); SettingsInfo(stringResource(R.string.multiplexer), "Yamux"); SettingsDivider(); SettingsInfo(stringResource(R.string.encryption), "AES-256-CFB"); SettingsDivider(); SettingsInfo(stringResource(R.string.proxy), "SOCKS5"); SettingsDivider(); SettingsInfo(stringResource(R.string.relay), "TURN (WebRTC ICE)")
            }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.data)) { SettingsClickable(stringResource(R.string.reset_all), stringResource(R.string.reset_all_desc), Icons.Filled.RestartAlt) { showResetDialog = true } }
            Spacer(Modifier.height(16.dp))
            SettingsGroup(stringResource(R.string.about)) {
                Column(Modifier.fillMaxWidth().padding(16.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("lionheart", style = MaterialTheme.typography.headlineMedium.copy(fontWeight = FontWeight.Bold, letterSpacing = 1.sp), color = MaterialTheme.colorScheme.primary)
                    Spacer(Modifier.height(4.dp)); Text(stringResource(R.string.version_format, APP_VERSION), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Spacer(Modifier.height(12.dp)); Text(stringResource(R.string.about_description), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.7f), textAlign = TextAlign.Center)
                }
            }
            Spacer(Modifier.windowInsetsBottomHeight(WindowInsets.navigationBars)); Spacer(Modifier.height(40.dp))
        }
    }
    if (showThemeDialog) {
        val opts = listOf("system" to stringResource(R.string.theme_system), "dark" to stringResource(R.string.theme_dark), "light" to stringResource(R.string.theme_light))
        AlertDialog(onDismissRequest = { showThemeDialog = false }, title = { Text(stringResource(R.string.theme)) }, text = {
            Column { opts.forEach { (v, l) -> Row(Modifier.fillMaxWidth().padding(vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) { RadioButton(selected = theme == v, onClick = { vm.setTheme(v); showThemeDialog = false }); Spacer(Modifier.width(8.dp)); Text(l) } } }
        }, confirmButton = {}, dismissButton = { TextButton(onClick = { showThemeDialog = false }) { Text(stringResource(R.string.close)) } })
    }
    if (showResetDialog) {
        AlertDialog(onDismissRequest = { showResetDialog = false }, title = { Text(stringResource(R.string.reset_all_confirm)) }, text = { Text(stringResource(R.string.reset_all_warning)) },
            confirmButton = { TextButton(onClick = { vm.resetAll(); showResetDialog = false; onBack() }) { Text(stringResource(R.string.reset), color = MaterialTheme.colorScheme.error) } },
            dismissButton = { TextButton(onClick = { showResetDialog = false }) { Text(stringResource(R.string.cancel)) } })
    }
}
@Composable
private fun LionIconOption(isSelected: Boolean, label: String, bgColor: Color, fgRes: Int, fgScale: Float = 1f, onClick: () -> Unit) {
    Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.width(50.dp).clickable(onClick = onClick)) {
        Box(
            modifier = Modifier.size(44.dp).clip(RoundedCornerShape(13.dp))
                .then(if (isSelected) Modifier.border(2.5.dp, MaterialTheme.colorScheme.primary, RoundedCornerShape(13.dp)) else Modifier)
                .background(bgColor),
            contentAlignment = Alignment.Center
        ) {
            Image(painter = painterResource(fgRes), contentDescription = label, modifier = Modifier.fillMaxSize().scale(fgScale))
        }
        Spacer(Modifier.height(4.dp))
        Text(label, style = MaterialTheme.typography.labelSmall, color = if (isSelected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center, maxLines = 1, fontSize = 10.sp)
    }
}
@Composable
private fun InlineDisguiseOption(isSelected: Boolean, label: String, icon: ImageVector, bgColor: Color, iconTint: Color, onClick: () -> Unit) {
    Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.width(50.dp).clickable(onClick = onClick)) {
        Box(
            modifier = Modifier.size(44.dp).clip(RoundedCornerShape(13.dp))
                .then(if (isSelected) Modifier.border(2.5.dp, MaterialTheme.colorScheme.primary, RoundedCornerShape(13.dp)) else Modifier)
                .background(bgColor),
            contentAlignment = Alignment.Center
        ) { Icon(icon, null, tint = iconTint, modifier = Modifier.size(24.dp)) }
        Spacer(Modifier.height(4.dp))
        Text(label, style = MaterialTheme.typography.labelSmall, color = if (isSelected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center, maxLines = 1, fontSize = 10.sp)
    }
}
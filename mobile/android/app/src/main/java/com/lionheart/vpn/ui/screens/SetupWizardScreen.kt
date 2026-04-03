package com.lionheart.vpn.ui.screens

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.automirrored.filled.Label
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.lionheart.vpn.R
import com.lionheart.vpn.viewmodel.VpnViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SetupWizardScreen(
    vm: VpnViewModel,
    onBack: () -> Unit,
    onNavigateQR: () -> Unit
) {
    val setupState by vm.setupState.collectAsState()
    val showAddKeyDialog by vm.showAddKeyDialog.collectAsState() // ДОБАВЛЕНО
    
    var host by remember { mutableStateOf("") }
    var port by remember { mutableStateOf("22") }
    var username by remember { mutableStateOf("root") }
    var password by remember { mutableStateOf("") }
    var serverName by remember { mutableStateOf("") }
    var showPassword by remember { mutableStateOf(false) }
    var currentStep by remember { mutableIntStateOf(0) }
    val isInstalling = setupState != null && setupState?.error == null && setupState?.smartKey == null
    val isFinished = setupState?.smartKey != null
    val hasError = setupState?.error != null
    
    LaunchedEffect(isInstalling) { if (isInstalling) currentStep = 3 }
    LaunchedEffect(isFinished) { if (isFinished) currentStep = 4 }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.setup_server)) },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, stringResource(R.string.back)) } },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.surface)
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState()).padding(horizontal = 24.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Spacer(Modifier.height(16.dp))
            Text(stringResource(R.string.auto_setup), style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold))
            Spacer(Modifier.height(8.dp))
            Text(stringResource(R.string.auto_setup_desc), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center)
            Spacer(Modifier.height(8.dp))
            if (currentStep < 3) {
                Row(modifier = Modifier.fillMaxWidth().padding(vertical = 12.dp), horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
                    for (i in 0..2) {
                        val isActive = i == currentStep; val isDone = i < currentStep
                        Box(modifier = Modifier.size(if (isActive) 36.dp else 28.dp).clip(RoundedCornerShape(12.dp)).background(when { isActive -> MaterialTheme.colorScheme.primary; isDone -> MaterialTheme.colorScheme.primaryContainer; else -> MaterialTheme.colorScheme.surfaceContainerHigh }), contentAlignment = Alignment.Center) {
                            if (isDone) Icon(Icons.Filled.Check, null, tint = MaterialTheme.colorScheme.onPrimaryContainer, modifier = Modifier.size(16.dp))
                            else Text("${i + 1}", style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.Bold), color = if (isActive) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                        if (i < 2) Box(Modifier.width(32.dp).height(3.dp).background(if (i < currentStep) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceContainerHighest, RoundedCornerShape(2.dp)))
                    }
                }
            }
            Spacer(Modifier.height(8.dp))
            if (currentStep < 3) {
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.tertiaryContainer), shape = RoundedCornerShape(16.dp), modifier = Modifier.fillMaxWidth()) {
                    Row(Modifier.padding(14.dp), verticalAlignment = Alignment.Top) {
                        Icon(Icons.Filled.Info, null, tint = MaterialTheme.colorScheme.onTertiaryContainer, modifier = Modifier.size(20.dp)); Spacer(Modifier.width(10.dp))
                        Text(stringResource(R.string.reassurance_msg), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onTertiaryContainer)
                    }
                }
                Spacer(Modifier.height(16.dp))
            }
            
            AnimatedVisibility(visible = currentStep == 0 && !isInstalling && !isFinished, enter = fadeIn() + slideInHorizontally(), exit = fadeOut() + slideOutHorizontally(targetOffsetX = { -it })) {
                Column {
                    Text(stringResource(R.string.step_name_title), style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.SemiBold))
                    Spacer(Modifier.height(4.dp)); Text(stringResource(R.string.step_name_desc), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Spacer(Modifier.height(16.dp))
                    OutlinedTextField(value = serverName, onValueChange = { serverName = it }, label = { Text(stringResource(R.string.server_name)) }, placeholder = { Text(stringResource(R.string.server_name_hint)) }, singleLine = true, leadingIcon = { Icon(Icons.AutoMirrored.Filled.Label, null) }, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp))
                    Spacer(Modifier.height(24.dp))
                    Button(onClick = { currentStep = 1 }, modifier = Modifier.fillMaxWidth().height(52.dp), shape = RoundedCornerShape(14.dp)) {
                        Text(stringResource(R.string.next), style = MaterialTheme.typography.titleMedium); Spacer(Modifier.width(8.dp)); Icon(Icons.AutoMirrored.Filled.ArrowForward, null, modifier = Modifier.size(20.dp))
                    }
                }
            }
            
            AnimatedVisibility(visible = currentStep == 1 && !isInstalling && !isFinished, enter = fadeIn() + slideInHorizontally(initialOffsetX = { it }), exit = fadeOut() + slideOutHorizontally(targetOffsetX = { -it })) {
                Column {
                    Text(stringResource(R.string.step_address_title), style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.SemiBold))
                    Spacer(Modifier.height(12.dp))
                    OutlinedTextField(value = host, onValueChange = { host = it }, label = { Text(stringResource(R.string.ip_address)) }, placeholder = { Text("123.45.67.89") }, singleLine = true, leadingIcon = { Icon(Icons.Filled.Cloud, null) }, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp), keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri))
                    Card(modifier = Modifier.fillMaxWidth().padding(top = 8.dp), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerHigh), shape = RoundedCornerShape(12.dp)) {
                        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.Top) {
                            Icon(Icons.Filled.Lightbulb, null, tint = MaterialTheme.colorScheme.tertiary, modifier = Modifier.size(18.dp)); Spacer(Modifier.width(8.dp))
                            Column {
                                Text(stringResource(R.string.step_ip_hint), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                Spacer(Modifier.height(4.dp))
                                Text("curl 2ip.ru", style = MaterialTheme.typography.bodySmall.copy(fontFamily = FontFamily.Monospace, fontWeight = FontWeight.Bold), color = MaterialTheme.colorScheme.primary)
                            }
                        }
                    }
                    Spacer(Modifier.height(16.dp))
                    OutlinedTextField(value = port, onValueChange = { port = it }, label = { Text(stringResource(R.string.ssh_port)) }, singleLine = true, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp), keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number))
                    Text(stringResource(R.string.step_ssh_port_hint), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 6.dp, start = 4.dp))
                    Spacer(Modifier.height(24.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        OutlinedButton(onClick = { currentStep = 0 }, modifier = Modifier.weight(1f).height(52.dp), shape = RoundedCornerShape(14.dp)) { Icon(Icons.AutoMirrored.Filled.ArrowBack, null, modifier = Modifier.size(18.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.back)) }
                        Button(onClick = { currentStep = 2 }, enabled = host.isNotBlank(), modifier = Modifier.weight(1f).height(52.dp), shape = RoundedCornerShape(14.dp)) { Text(stringResource(R.string.next)); Spacer(Modifier.width(8.dp)); Icon(Icons.AutoMirrored.Filled.ArrowForward, null, modifier = Modifier.size(18.dp)) }
                    }
                }
            }
            
            AnimatedVisibility(visible = currentStep == 2 && !isInstalling && !isFinished, enter = fadeIn() + slideInHorizontally(initialOffsetX = { it }), exit = fadeOut() + slideOutHorizontally(targetOffsetX = { -it })) {
                Column {
                    Text(stringResource(R.string.step_auth_title), style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.SemiBold))
                    Spacer(Modifier.height(12.dp))
                    OutlinedTextField(value = username, onValueChange = { username = it }, label = { Text(stringResource(R.string.username)) }, singleLine = true, leadingIcon = { Icon(Icons.Filled.Person, null) }, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp))
                    Spacer(Modifier.height(12.dp))
                    OutlinedTextField(value = password, onValueChange = { password = it }, label = { Text(stringResource(R.string.ssh_password)) }, singleLine = true, leadingIcon = { Icon(Icons.Filled.Lock, null) },
                        visualTransformation = if (showPassword) VisualTransformation.None else PasswordVisualTransformation(),
                        trailingIcon = { IconButton(onClick = { showPassword = !showPassword }) { Icon(if (showPassword) Icons.Filled.VisibilityOff else Icons.Filled.Visibility, stringResource(R.string.show_password)) } },
                        modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp))
                    Card(modifier = Modifier.fillMaxWidth().padding(top = 12.dp), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerHigh), shape = RoundedCornerShape(12.dp)) {
                        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.Top) {
                            Icon(Icons.Filled.Lightbulb, null, tint = MaterialTheme.colorScheme.tertiary, modifier = Modifier.size(18.dp)); Spacer(Modifier.width(8.dp))
                            Text(stringResource(R.string.step_auth_hint), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                    if (hasError) {
                        Spacer(Modifier.height(12.dp))
                        Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer), shape = RoundedCornerShape(14.dp)) {
                            Text(setupState!!.error!!, modifier = Modifier.padding(16.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onErrorContainer)
                        }
                    }
                    Spacer(Modifier.height(24.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        OutlinedButton(onClick = { currentStep = 1 }, modifier = Modifier.weight(1f).height(52.dp), shape = RoundedCornerShape(14.dp)) { Icon(Icons.AutoMirrored.Filled.ArrowBack, null, modifier = Modifier.size(18.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.back)) }
                        Button(onClick = { vm.startSetup(host = host.trim(), port = port.toIntOrNull() ?: 22, username = username.trim(), password = password, serverName = serverName.trim().ifBlank { host.trim() }) }, enabled = host.isNotBlank() && password.isNotBlank(), modifier = Modifier.weight(1f).height(52.dp), shape = RoundedCornerShape(14.dp)) {
                            Icon(Icons.Filled.RocketLaunch, null, modifier = Modifier.size(20.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.install))
                        }
                    }
                }
            }
            
            AnimatedVisibility(visible = isInstalling, enter = fadeIn() + expandVertically(), exit = fadeOut() + shrinkVertically()) {
                setupState?.let { state ->
                    Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.padding(vertical = 32.dp)) {
                        CircularProgressIndicator(modifier = Modifier.size(56.dp), strokeWidth = 5.dp)
                        Spacer(Modifier.height(24.dp))
                        Text(state.message, style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Medium), textAlign = TextAlign.Center)
                        Spacer(Modifier.height(16.dp))
                        LinearProgressIndicator(
                            progress = { 
                                val p = state.step.toFloat() / state.totalSteps
                                if (p.isNaN()) 0f else p
                            }, 
                            modifier = Modifier.fillMaxWidth().height(8.dp).clip(RoundedCornerShape(4.dp)), 
                            trackColor = MaterialTheme.colorScheme.surfaceContainerHighest
                        )
                        Spacer(Modifier.height(8.dp))
                        Text(stringResource(R.string.step_of, state.step, state.totalSteps), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
            
            AnimatedVisibility(visible = isFinished, enter = fadeIn() + scaleIn(initialScale = 0.8f), exit = fadeOut()) {
                Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.padding(vertical = 32.dp)) {
                    Box(modifier = Modifier.size(80.dp).clip(RoundedCornerShape(24.dp)).background(MaterialTheme.colorScheme.primaryContainer), contentAlignment = Alignment.Center) { Icon(Icons.Filled.Check, null, tint = MaterialTheme.colorScheme.onPrimaryContainer, modifier = Modifier.size(40.dp)) }
                    Spacer(Modifier.height(24.dp)); Text(stringResource(R.string.server_installed), style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold))
                    Spacer(Modifier.height(8.dp)); Text(stringResource(R.string.server_installed_desc), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center)
                    Spacer(Modifier.height(28.dp))
                    Button(onClick = { vm.clearSetupState(); onBack() }, modifier = Modifier.fillMaxWidth().height(56.dp), shape = RoundedCornerShape(16.dp)) { Text(stringResource(R.string.connect_now), style = MaterialTheme.typography.titleMedium) }
                }
            }
            
            Spacer(Modifier.height(32.dp))
            if (currentStep < 3 || hasError) {
                HorizontalDivider(modifier = Modifier.padding(vertical = 16.dp))
                Text(stringResource(R.string.or_add_existing), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Spacer(Modifier.height(12.dp))
                OutlinedButton(onClick = { vm.showAddByKey() }, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp)) { Icon(Icons.Filled.Key, null, modifier = Modifier.size(18.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.enter_smart_key)) }
                Spacer(Modifier.height(8.dp))
                OutlinedButton(onClick = onNavigateQR, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(14.dp)) { Icon(Icons.Filled.QrCodeScanner, null, modifier = Modifier.size(18.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.scan_qr)) }
            }
            Spacer(Modifier.height(40.dp))
        }

        // ДОБАВЛЕНО: Реализация диалогового окна для ввода смарт-ключа
        if (showAddKeyDialog) {
            var manualKey by remember { mutableStateOf("") }
            AlertDialog(
                onDismissRequest = { vm.hideAddKeyDialog() },
                title = { Text(stringResource(R.string.enter_smart_key)) },
                text = {
                    OutlinedTextField(
                        value = manualKey,
                        onValueChange = { manualKey = it },
                        label = { Text("Smart Key") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth()
                    )
                },
                confirmButton = {
                    TextButton(onClick = {
                        if (manualKey.isNotBlank()) {
                            // Оставляем имя сервера пустым, ViewModel сама подставит IP из ключа
                            vm.addServerByKey("", manualKey.trim())
                            onBack() // Возвращаемся на главную
                        }
                    }) { Text(stringResource(R.string.ok)) }
                },
                dismissButton = {
                    TextButton(onClick = { vm.hideAddKeyDialog() }) { Text(stringResource(R.string.cancel)) }
                }
            )
        }
    }
}
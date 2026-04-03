package com.lionheart.vpn.ui.screens
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Android
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.lionheart.vpn.R
import com.lionheart.vpn.viewmodel.VpnViewModel
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SplitTunnelScreen(
    vm: VpnViewModel,
    serverId: String = "",
    onBack: () -> Unit
) {
    val apps by vm.installedApps.collectAsState()
    val servers by vm.servers.collectAsState()
    val server = servers.find { it.id == serverId }
    val selectedApps = server?.splitApps ?: emptyList()
    val splitMode = server?.splitMode ?: "bypass"
    var searchQuery by remember { mutableStateOf("") }
    var showSystemApps by remember { mutableStateOf(false) }
    LaunchedEffect(Unit) { vm.loadInstalledApps() }
    val filteredApps = apps.filter { app ->
        (showSystemApps || !app.isSystem) &&
        (searchQuery.isBlank() || app.label.contains(searchQuery, ignoreCase = true) ||
            app.packageName.contains(searchQuery, ignoreCase = true))
    }
    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        if (splitMode == "bypass") stringResource(R.string.bypass_vpn) else stringResource(R.string.only_through_vpn)
                    )
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, stringResource(R.string.back))
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.background
                )
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            OutlinedTextField(
                value = searchQuery,
                onValueChange = { searchQuery = it },
                placeholder = { Text(stringResource(R.string.search_apps)) },
                leadingIcon = { Icon(Icons.Filled.Search, null) },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                singleLine = true
            )
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { showSystemApps = !showSystemApps }
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Checkbox(checked = showSystemApps, onCheckedChange = { showSystemApps = it })
                Spacer(Modifier.width(8.dp))
                Text(stringResource(R.string.show_system_apps), style = MaterialTheme.typography.bodyMedium)
            }
            Text(
                stringResource(R.string.selected_count, selectedApps.size),
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp)
            )
            HorizontalDivider()
            if (apps.isEmpty()) {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    CircularProgressIndicator()
                }
            } else {
                LazyColumn {
                    items(filteredApps, key = { it.packageName }) { app ->
                        val isSelected = selectedApps.contains(app.packageName)
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable { vm.toggleSplitApp(serverId, app.packageName) }
                                .padding(horizontal = 16.dp, vertical = 10.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Checkbox(
                                checked = isSelected,
                                onCheckedChange = { vm.toggleSplitApp(serverId, app.packageName) }
                            )
                            Spacer(Modifier.width(12.dp))
                            Icon(
                                Icons.Filled.Android,
                                contentDescription = null,
                                tint = if (app.isSystem) MaterialTheme.colorScheme.onSurfaceVariant
                                    else MaterialTheme.colorScheme.primary,
                                modifier = Modifier.size(32.dp)
                            )
                            Spacer(Modifier.width(12.dp))
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = app.label,
                                    style = MaterialTheme.typography.bodyMedium,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis
                                )
                                Text(
                                    text = app.packageName,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
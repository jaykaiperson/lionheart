package com.lionheart.vpn.ui.screens
import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.DeleteOutline
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lionheart.vpn.R
import com.lionheart.vpn.viewmodel.VpnViewModel
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LogsScreen(
    vm: VpnViewModel,
    onBack: () -> Unit
) {
    val logs by vm.logs.collectAsState()
    val listState = rememberLazyListState()
    LaunchedEffect(logs.size) {
        if (logs.isNotEmpty()) listState.animateScrollToItem(logs.size - 1)
    }
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.logs)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, stringResource(R.string.back))
                    }
                },
                actions = {
                    IconButton(onClick = { vm.clearLogs() }) {
                        Icon(Icons.Filled.DeleteOutline, "Clear")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface
                )
            )
        }
    ) { padding ->
        val termBg = MaterialTheme.colorScheme.surfaceContainerLowest
        val termFg = MaterialTheme.colorScheme.onSurface
        val termMuted = MaterialTheme.colorScheme.onSurfaceVariant
        val infoColor = MaterialTheme.colorScheme.primary
        val warnColor = MaterialTheme.colorScheme.tertiary
        val errColor = MaterialTheme.colorScheme.error
        LazyColumn(
            state = listState,
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .background(termBg),
            contentPadding = PaddingValues(8.dp)
        ) {
            if (logs.isEmpty()) {
                item {
                    Text(
                        stringResource(R.string.logs_hint),
                        style = MaterialTheme.typography.bodyMedium,
                        color = termMuted,
                        modifier = Modifier.padding(16.dp)
                    )
                }
            }
            items(logs, key = { "${it.time}_${it.message.hashCode()}" }) { entry ->
                val levelColor = when (entry.level) {
                    "info" -> infoColor
                    "warn" -> warnColor
                    "error" -> errColor
                    else -> termMuted
                }
                val levelTag = when (entry.level) {
                    "info" -> "INF"; "warn" -> "WRN"; "error" -> "ERR"; else -> "???"
                }
                Text(
                    text = buildAnnotatedString {
                        withStyle(SpanStyle(color = termMuted)) { append(entry.time) }
                        append(" ")
                        withStyle(SpanStyle(color = levelColor)) { append("[$levelTag]") }
                        append(" ")
                        withStyle(SpanStyle(color = termFg)) { append(entry.message) }
                    },
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    lineHeight = 18.sp,
                    modifier = Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState())
                        .padding(vertical = 1.dp)
                )
            }
        }
    }
}
package com.lionheart.vpn

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.os.Bundle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.lionheart.vpn.service.LionheartVpnService
import com.lionheart.vpn.ui.screens.*
import com.lionheart.vpn.ui.theme.LionheartTheme
import com.lionheart.vpn.viewmodel.VpnStatus
import com.lionheart.vpn.viewmodel.VpnViewModel

class MainActivity : AppCompatActivity() {
    private var pendingVpnConnect = false
    
    private val vpnPermissionLauncher = registerForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        if (result.resultCode == Activity.RESULT_OK) {
            startForegroundService(Intent(this, LionheartVpnService::class.java).apply { action = LionheartVpnService.ACTION_START })
        }
        pendingVpnConnect = false
    }

    override fun attachBaseContext(newBase: Context) {
        val prefs = newBase.getSharedPreferences("lh_prefs", MODE_PRIVATE)
        val savedLocale = prefs.getString("app_locale", null)
        if (savedLocale != null && savedLocale.isNotBlank()) {
            val locale = java.util.Locale.forLanguageTag(savedLocale)
            val config = newBase.resources.configuration
            config.setLocale(locale)
            super.attachBaseContext(newBase.createConfigurationContext(config))
        } else {
            super.attachBaseContext(newBase)
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            val vm: VpnViewModel = viewModel()
            val theme by vm.theme.collectAsState()
            val dynamicColor by vm.dynamicColor.collectAsState()
            
            LionheartTheme(themeMode = theme, dynamicColor = dynamicColor) {
                Surface(Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
                    LionheartNavigation(vm) { requestVpnPermission(vm) }
                }
            }
            
            LaunchedEffect(Unit) {
                vm.autoConnect.collect { auto ->
                    if (auto && vm.smartKey.value.isNotBlank() && vm.status.value == VpnStatus.DISCONNECTED) requestVpnPermission(vm)
                }
            }
        }
    }

    private fun requestVpnPermission(vm: VpnViewModel) {
        if (pendingVpnConnect) return
        pendingVpnConnect = true
        val intent = vm.needsVpnPermission()
        if (intent != null) vpnPermissionLauncher.launch(intent)
        else { vm.connect(); pendingVpnConnect = false }
    }
}

@Composable
fun LionheartNavigation(vm: VpnViewModel, onRequestVpnPermission: () -> Unit) {
    val navController = rememberNavController()
    
    NavHost(navController, startDestination = "home") {
        composable("home") {
            HomeScreen(
                vm = vm,
                onNavigateSettings = { navController.navigate("settings") },
                onNavigateLogs = { navController.navigate("logs") },
                onNavigateSetup = { navController.navigate("setup_wizard") },
                onNavigateServerDetail = { id -> navController.navigate("server_detail/$id") }
            )
        }
        composable("settings") {
            SettingsScreen(vm, onBack = { navController.popBackStack() }, onScanQR = { navController.navigate("qr_scanner") })
        }
        composable("logs") { 
            LogsScreen(vm = vm, onBack = { navController.popBackStack() }) 
        }
        composable("split_tunnel/{serverId}") { entry ->
            val serverId = entry.arguments?.getString("serverId") ?: ""
            SplitTunnelScreen(vm = vm, serverId = serverId, onBack = { navController.popBackStack() })
        }
        composable("qr_scanner") { 
            QRScannerScreen(vm = vm, onBack = { navController.popBackStack() }) 
        }
        composable("setup_wizard") { 
            SetupWizardScreen(
                vm = vm, 
                onBack = { navController.popBackStack() },
                // ИСПРАВЛЕНО: Теперь навигация на сканер работает
                onNavigateQR = { navController.navigate("qr_scanner") } 
            ) 
        }
        composable("server_detail/{serverId}") { entry ->
            val serverId = entry.arguments?.getString("serverId") ?: ""
            ServerDetailScreen(
                vm = vm,
                serverId = serverId,
                onBack = { navController.popBackStack() },
                onNavigateSplitTunnel = { id -> navController.navigate("split_tunnel/$id") }
            )
        }
    }
}
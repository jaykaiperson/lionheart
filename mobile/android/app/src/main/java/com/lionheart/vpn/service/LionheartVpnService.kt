package com.lionheart.vpn.service

import android.app.Notification
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.ParcelFileDescriptor
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import com.lionheart.vpn.LionheartApp
import com.lionheart.vpn.MainActivity
import com.lionheart.vpn.R
import com.lionheart.vpn.data.AdBlockConfig
import com.lionheart.vpn.data.PrefsRepository
import com.lionheart.vpn.data.ServerProfile
import golib.Golib
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.first

class LionheartVpnService : VpnService() {
    companion object {
        const val ACTION_START = "com.lionheart.vpn.START"
        const val ACTION_STOP = "com.lionheart.vpn.STOP"
        private const val TUN_ADDR = "10.0.85.1"
        private const val TAG = "LH_SVC"
        private const val CONNECT_TIMEOUT_MS = 30_000L
        @Volatile
        var isRunning = false
            private set
    }

    private var tunInterface: ParcelFileDescriptor? = null
    private var tunnelJob: Job? = null
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private lateinit var prefs: PrefsRepository
    @Volatile
    private var stopping = false

    override fun onCreate() {
        super.onCreate()
        prefs = PrefsRepository(this)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        Log.d(TAG, "onStartCommand action=${intent?.action}")
        if (intent?.action == ACTION_START) {
            startForeground(LionheartApp.NOTIF_ID, buildNotification("Подключение..."))
        }
        when (intent?.action) {
            ACTION_START -> startVpn()
            ACTION_STOP -> requestStop()
        }
        return START_STICKY
    }

    override fun onDestroy() {
        Log.d(TAG, "onDestroy")
        requestStop()
        scope.cancel()
        super.onDestroy()
    }

    override fun onRevoke() {
        Log.d(TAG, "onRevoke")
        requestStop()
        super.onRevoke()
    }

    private fun startVpn() {
        if (isRunning) return
        stopping = false
        isRunning = true
        tunnelJob = scope.launch {
            try {
                val smartKey = prefs.smartKey.first()
                if (smartKey.isBlank()) {
                    cleanup()
                    return@launch
                }
                val profile = loadActiveProfile()
                val dns = AdBlockConfig.getDns(profile?.adBlock == true, profile?.dns ?: prefs.dns.first())
                val mtu = profile?.mtu ?: 1500
                val splitEnabled = profile?.splitEnabled ?: false
                val splitMode = profile?.splitMode ?: "bypass"
                val splitApps = profile?.splitApps ?: emptyList()
                val ipMode = profile?.ipMode ?: "prefer_v4"
                val killSwitch = profile?.killSwitch ?: false
                val connectionMode = profile?.connectionMode ?: "vpn"
                
                if (stopping) { cleanup(); return@launch }
                
                // ИСПРАВЛЕНО: В режиме socks5 не перехватываем трафик (не вызываем builder.establish())
                if (connectionMode == "socks5") {
                    updateNotification("SOCKS5 Proxy (127.0.0.1:1080)")
                    Log.d(TAG, "Starting Golib in SOCKS5 mode (no tun interface)")
                    Golib.start(smartKey, -1L, mtu.toLong(), dns)
                } else {
                    val builder = Builder()
                        .setSession("lionheart")
                        .setMtu(mtu)
                        .addAddress(TUN_ADDR, 24)
                        .addDnsServer(dns)
                        .setBlocking(true)
                        
                    when (ipMode) {
                        "only_v4" -> builder.addRoute("0.0.0.0", 0)
                        "only_v6" -> { builder.addRoute("::", 0); builder.addAddress("fd00::1", 128) }
                        "prefer_v6" -> { builder.addRoute("0.0.0.0", 0); builder.addRoute("::", 0); builder.addAddress("fd00::1", 128) }
                        else -> builder.addRoute("0.0.0.0", 0)
                    }
                    
                    if (!killSwitch) {
                        try { builder.addDisallowedApplication(packageName) } catch (_: Exception) {}
                        if (splitEnabled && splitApps.isNotEmpty()) {
                            when (splitMode) {
                                "bypass" -> splitApps.forEach { try { builder.addDisallowedApplication(it) } catch (_: Exception) {} }
                                "only" -> splitApps.forEach { try { builder.addAllowedApplication(it) } catch (_: Exception) {} }
                            }
                        }
                    }
                    
                    if (stopping) { cleanup(); return@launch }
                    tunInterface = builder.establish() ?: run { cleanup(); return@launch }
                    updateNotification("Подключено")
                    
                    val goFd = tunInterface!!.fd
                    if (stopping) { cleanup(); return@launch }
                    Log.d(TAG, "Golib.start() fd=$goFd")
                    Golib.start(smartKey, goFd.toLong(), mtu.toLong(), dns)
                }
                
            } catch (e: CancellationException) {
                Log.d(TAG, "tunnel cancelled")
            } catch (e: Exception) {
                Log.e(TAG, "tunnel error", e)
            } finally {
                cleanup()
            }
        }
        scope.launch {
            delay(CONNECT_TIMEOUT_MS)
            if (isRunning && !stopping) {
                try {
                    val status = Golib.getStatus()
                    if (status == "connecting") {
                        Log.w(TAG, "Connection timeout after ${CONNECT_TIMEOUT_MS}ms")
                        requestStop()
                    }
                } catch (_: Exception) {}
            }
        }
    }

    private fun requestStop() {
        Log.d(TAG, "requestStop stopping=$stopping isRunning=$isRunning")
        if (stopping) return
        stopping = true
        isRunning = false
        tunnelJob?.cancel()
        tunnelJob = null
        scope.launch(Dispatchers.IO) {
            try {
                Log.d(TAG, "Golib.stop() on IO...")
                Golib.stop()
                Log.d(TAG, "Golib.stop() done")
            } catch (e: Exception) {
                Log.e(TAG, "Golib.stop() err: ${e.message}")
            }
            withContext(Dispatchers.Main) {
                performServiceShutdown()
            }
        }
    }

    private fun cleanup() {
        if (stopping) return
        stopping = true
        isRunning = false
        scope.launch(Dispatchers.Main) {
            performServiceShutdown()
        }
    }

    private fun performServiceShutdown() {
        try { tunInterface?.close() } catch (_: Exception) {}
        tunInterface = null
        try { stopForeground(STOP_FOREGROUND_REMOVE) } catch (_: Exception) {}
        try { stopSelf() } catch (_: Exception) {}
        Log.d(TAG, "service stopped")
    }

    private suspend fun loadActiveProfile(): ServerProfile? = try {
        val json = prefs.getServersJsonSync(); val id = prefs.getActiveServerIdSync()
        if (json.isBlank() || id.isBlank()) null
        else Gson().fromJson<List<ServerProfile>>(json, object : TypeToken<List<ServerProfile>>() {}.type).find { it.id == id }
    } catch (_: Exception) { null }

    private fun buildNotification(text: String): Notification {
        val pi = PendingIntent.getActivity(this, 0, Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        return NotificationCompat.Builder(this, LionheartApp.CHANNEL_ID)
            .setContentTitle("lionheart")
            .setContentText(text)
            .setSmallIcon(R.drawable.ic_vpn_key)
            .setContentIntent(pi)
            .setOngoing(true)
            .setSilent(true)
            .build()
    }

    private fun updateNotification(text: String) {
        (getSystemService(NOTIFICATION_SERVICE) as android.app.NotificationManager)
            .notify(LionheartApp.NOTIF_ID, buildNotification(text))
    }
}
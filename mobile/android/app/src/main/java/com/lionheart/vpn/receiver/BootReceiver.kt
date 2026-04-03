package com.lionheart.vpn.receiver
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.lionheart.vpn.data.PrefsRepository
import com.lionheart.vpn.service.LionheartVpnService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_BOOT_COMPLETED &&
            intent.action != Intent.ACTION_MY_PACKAGE_REPLACED
        ) return
        val prefs = PrefsRepository(context)
        CoroutineScope(Dispatchers.IO).launch {
            val boot = prefs.getBootConnectSync()
            val key = prefs.getSmartKeySync()
            if (boot && key.isNotBlank()) {
                val vpnIntent = Intent(context, LionheartVpnService::class.java).apply {
                    action = LionheartVpnService.ACTION_START
                }
                context.startForegroundService(vpnIntent)
            }
        }
    }
}
package com.lionheart.vpn
import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build
class LionheartApp : Application() {
    companion object {
        const val CHANNEL_ID = "lionheart_vpn"
        const val NOTIF_ID = 1
        lateinit var instance: LionheartApp private set
    }
    override fun onCreate() {
        super.onCreate()
        instance = this
        createNotificationChannel()
    }
    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "lionheart",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "VPN connection status"
                setShowBadge(false)
            }
            getSystemService(NotificationManager::class.java)
                .createNotificationChannel(channel)
        }
    }
}
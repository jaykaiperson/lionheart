package com.lionheart.vpn.service
import android.content.Intent
import android.os.Build
import android.service.quicksettings.Tile
import android.service.quicksettings.TileService
import androidx.annotation.RequiresApi
@RequiresApi(Build.VERSION_CODES.N)
class LionheartTileService : TileService() {
    override fun onStartListening() {
        super.onStartListening()
        updateTile()
    }
    override fun onClick() {
        super.onClick()
        val intent = if (LionheartVpnService.isRunning) {
            Intent(this, LionheartVpnService::class.java).apply {
                action = LionheartVpnService.ACTION_STOP
            }
        } else {
            Intent(this, LionheartVpnService::class.java).apply {
                action = LionheartVpnService.ACTION_START
            }
        }
        startService(intent)
        updateTile()
    }
    private fun updateTile() {
        qsTile?.let { tile ->
            tile.state = if (LionheartVpnService.isRunning) Tile.STATE_ACTIVE else Tile.STATE_INACTIVE
            tile.label = "lionheart"
            tile.subtitle = if (LionheartVpnService.isRunning) "Connected" else "Disconnected"
            tile.updateTile()
        }
    }
}
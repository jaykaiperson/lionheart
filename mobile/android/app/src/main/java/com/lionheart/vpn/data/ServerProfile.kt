package com.lionheart.vpn.data
import android.content.Context
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import java.util.UUID
data class ServerProfile(
    val id: String = UUID.randomUUID().toString(),
    val name: String = "",
    val smartKey: String = "",
    val serverIP: String = "",
    val countryCode: String = "",
    val dns: String = "1.1.1.1",
    val mtu: Int = 1500,
    val ipMode: String = "prefer_v4",
    val adBlock: Boolean = false,
    val killSwitch: Boolean = false,
    val connectionMode: String = "vpn",
    val splitEnabled: Boolean = false,
    val splitMode: String = "bypass",
    val splitApps: List<String> = emptyList(),
    val sshUser: String = "",
    val sshPort: Int = 22,
    val serverVersion: String = "",
    val showServerIP: Boolean = true,
    val pingUrl: String = "https://cp.cloudflare.com",
    val createdAt: Long = System.currentTimeMillis()
) {
    fun splitAppsSet(): Set<String> = splitApps.toSet()
}
class ServerRepository(private val context: Context) {
    private val prefs = PrefsRepository(context)
    private val gson = Gson()
    val servers: Flow<List<ServerProfile>> = prefs.serversJson.map { json -> parseServers(json) }
    val activeServerId: Flow<String> = prefs.activeServerId
    suspend fun addServer(server: ServerProfile) {
        val current = getCurrentServers().toMutableList()
        current.add(server)
        saveServers(current)
        if (current.size == 1) setActive(server.id)
    }
    suspend fun removeServer(id: String) {
        val current = getCurrentServers().toMutableList()
        val removed = current.removeAll { it.id == id }
        if (removed) saveServers(current)
    }
    suspend fun updateServer(server: ServerProfile) {
        val current = getCurrentServers().toMutableList()
        val idx = current.indexOfFirst { it.id == server.id }
        if (idx >= 0) { current[idx] = server; saveServers(current) }
    }
    suspend fun setActive(id: String) {
        prefs.setActiveServerId(id)
        val server = getCurrentServers().find { it.id == id }
        if (server != null) {
            prefs.setSmartKey(server.smartKey)
            prefs.setServerIP(server.serverIP)
            prefs.setDns(server.dns)
        }
    }
    suspend fun getActive(): ServerProfile? {
        val id = prefs.getActiveServerIdSync()
        return getCurrentServers().find { it.id == id }
    }
    suspend fun getById(id: String): ServerProfile? = getCurrentServers().find { it.id == id }
    suspend fun clearAll() {
        prefs.setServersJson(""); prefs.setActiveServerId(""); prefs.setSmartKey(""); prefs.setServerIP("")
    }
    private suspend fun getCurrentServers(): List<ServerProfile> = parseServers(prefs.getServersJsonSync())
    private fun parseServers(json: String): List<ServerProfile> {
        if (json.isBlank()) return emptyList()
        return try {
            gson.fromJson(json, object : TypeToken<List<ServerProfile>>() {}.type)
        } catch (_: Exception) { emptyList() }
    }
    private suspend fun saveServers(servers: List<ServerProfile>) {
        prefs.setServersJson(gson.toJson(servers))
    }
}
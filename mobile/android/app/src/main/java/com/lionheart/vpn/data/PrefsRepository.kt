package com.lionheart.vpn.data
import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.*
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
private val Context.dataStore: DataStore<Preferences> by preferencesDataStore(name = "lionheart_prefs")
class PrefsRepository(private val context: Context) {
    companion object {
        private val KEY_SMART_KEY = stringPreferencesKey("smart_key")
        private val KEY_SERVER_IP = stringPreferencesKey("server_ip")
        private val KEY_AUTO_CONNECT = booleanPreferencesKey("auto_connect")
        private val KEY_BOOT_CONNECT = booleanPreferencesKey("boot_connect")
        private val KEY_DNS = stringPreferencesKey("dns_server")
        private val KEY_SPLIT_TUNNEL_ENABLED = booleanPreferencesKey("split_tunnel_enabled")
        private val KEY_SPLIT_TUNNEL_APPS = stringSetPreferencesKey("split_tunnel_apps")
        private val KEY_SPLIT_TUNNEL_MODE = stringPreferencesKey("split_tunnel_mode")
        private val KEY_THEME = stringPreferencesKey("theme")
        private val KEY_DYNAMIC_COLOR = booleanPreferencesKey("dynamic_color")
        private val KEY_SERVERS_JSON = stringPreferencesKey("servers_json")
        private val KEY_ACTIVE_SERVER_ID = stringPreferencesKey("active_server_id")
    }
    val smartKey: Flow<String> = context.dataStore.data.map { it[KEY_SMART_KEY] ?: "" }
    val serverIP: Flow<String> = context.dataStore.data.map { it[KEY_SERVER_IP] ?: "" }
    val autoConnect: Flow<Boolean> = context.dataStore.data.map { it[KEY_AUTO_CONNECT] ?: false }
    val bootConnect: Flow<Boolean> = context.dataStore.data.map { it[KEY_BOOT_CONNECT] ?: false }
    val dns: Flow<String> = context.dataStore.data.map { it[KEY_DNS] ?: "1.1.1.1" }
    val splitTunnelEnabled: Flow<Boolean> = context.dataStore.data.map { it[KEY_SPLIT_TUNNEL_ENABLED] ?: false }
    val splitTunnelApps: Flow<Set<String>> = context.dataStore.data.map { it[KEY_SPLIT_TUNNEL_APPS] ?: emptySet() }
    val splitTunnelMode: Flow<String> = context.dataStore.data.map { it[KEY_SPLIT_TUNNEL_MODE] ?: "bypass" }
    val theme: Flow<String> = context.dataStore.data.map { it[KEY_THEME] ?: "system" }
    val dynamicColor: Flow<Boolean> = context.dataStore.data.map { it[KEY_DYNAMIC_COLOR] ?: true }
    val serversJson: Flow<String> = context.dataStore.data.map { it[KEY_SERVERS_JSON] ?: "" }
    val activeServerId: Flow<String> = context.dataStore.data.map { it[KEY_ACTIVE_SERVER_ID] ?: "" }
    suspend fun setSmartKey(key: String) { context.dataStore.edit { it[KEY_SMART_KEY] = key } }
    suspend fun setServerIP(ip: String) { context.dataStore.edit { it[KEY_SERVER_IP] = ip } }
    suspend fun setAutoConnect(enabled: Boolean) { context.dataStore.edit { it[KEY_AUTO_CONNECT] = enabled } }
    suspend fun setBootConnect(enabled: Boolean) { context.dataStore.edit { it[KEY_BOOT_CONNECT] = enabled } }
    suspend fun setDns(dns: String) { context.dataStore.edit { it[KEY_DNS] = dns } }
    suspend fun setSplitTunnelEnabled(enabled: Boolean) { context.dataStore.edit { it[KEY_SPLIT_TUNNEL_ENABLED] = enabled } }
    suspend fun setSplitTunnelApps(apps: Set<String>) { context.dataStore.edit { it[KEY_SPLIT_TUNNEL_APPS] = apps } }
    suspend fun setSplitTunnelMode(mode: String) { context.dataStore.edit { it[KEY_SPLIT_TUNNEL_MODE] = mode } }
    suspend fun setTheme(theme: String) { context.dataStore.edit { it[KEY_THEME] = theme } }
    suspend fun setDynamicColor(enabled: Boolean) { context.dataStore.edit { it[KEY_DYNAMIC_COLOR] = enabled } }
    suspend fun setServersJson(json: String) { context.dataStore.edit { it[KEY_SERVERS_JSON] = json } }
    suspend fun setActiveServerId(id: String) { context.dataStore.edit { it[KEY_ACTIVE_SERVER_ID] = id } }
    suspend fun getSmartKeySync(): String = context.dataStore.data.first()[KEY_SMART_KEY] ?: ""
    suspend fun getBootConnectSync(): Boolean = context.dataStore.data.first()[KEY_BOOT_CONNECT] ?: false
    suspend fun getServersJsonSync(): String = context.dataStore.data.first()[KEY_SERVERS_JSON] ?: ""
    suspend fun getActiveServerIdSync(): String = context.dataStore.data.first()[KEY_ACTIVE_SERVER_ID] ?: ""
}
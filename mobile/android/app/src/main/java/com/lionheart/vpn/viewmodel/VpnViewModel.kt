package com.lionheart.vpn.viewmodel
import android.app.Application
import android.content.Intent
import android.content.pm.ApplicationInfo
import android.content.pm.PackageManager
import android.net.VpnService
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.lionheart.vpn.data.*
import com.lionheart.vpn.service.LionheartVpnService
import golib.Golib
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL
const val APP_VERSION = "1.2"
enum class VpnStatus { DISCONNECTED, CONNECTING, CONNECTED, RECONNECTING, ERROR }
enum class VersionStatus { UNKNOWN, CHECKING, UP_TO_DATE, OUTDATED, UPDATE_FAILED, UPDATING, UPDATED, NO_SSH }
data class AppInfo(val packageName: String, val label: String, val isSystem: Boolean)
class VpnViewModel(application: Application) : AndroidViewModel(application) {
    val prefs = PrefsRepository(application)
    val serverRepo = ServerRepository(application)
    private val _status = MutableStateFlow(VpnStatus.DISCONNECTED)
    val status: StateFlow<VpnStatus> = _status.asStateFlow()
    private val _connectionTime = MutableStateFlow(0L)
    val connectionTime: StateFlow<Long> = _connectionTime.asStateFlow()
    private val _txBytes = MutableStateFlow(0L)
    val txBytes: StateFlow<Long> = _txBytes.asStateFlow()
    private val _rxBytes = MutableStateFlow(0L)
    val rxBytes: StateFlow<Long> = _rxBytes.asStateFlow()
    private val _txSpeed = MutableStateFlow(0L)
    val txSpeed: StateFlow<Long> = _txSpeed.asStateFlow()
    private val _rxSpeed = MutableStateFlow(0L)
    val rxSpeed: StateFlow<Long> = _rxSpeed.asStateFlow()
    private var lastTx = 0L
    private var lastRx = 0L
    private val _turnServer = MutableStateFlow("")
    val turnServer: StateFlow<String> = _turnServer.asStateFlow()
    private val _countryCode = MutableStateFlow("")
    val countryCode: StateFlow<String> = _countryCode.asStateFlow()
    private val _pingMs = MutableStateFlow(0L)
    val pingMs: StateFlow<Long> = _pingMs.asStateFlow()
    val servers = serverRepo.servers.stateIn(viewModelScope, SharingStarted.Eagerly, emptyList())
    val activeServerId = serverRepo.activeServerId.stateIn(viewModelScope, SharingStarted.Eagerly, "")
    val activeServer: StateFlow<ServerProfile?> = combine(servers, activeServerId) { list, id -> list.find { it.id == id } }
        .stateIn(viewModelScope, SharingStarted.Eagerly, null)
    private val _setupState = MutableStateFlow<SetupProgress?>(null)
    val setupState: StateFlow<SetupProgress?> = _setupState.asStateFlow()
    private val _showAddKeyDialog = MutableStateFlow(false)
    val showAddKeyDialog: StateFlow<Boolean> = _showAddKeyDialog.asStateFlow()
    private val _logs = MutableStateFlow<List<LogEntry>>(emptyList())
    val logs: StateFlow<List<LogEntry>> = _logs.asStateFlow()
    val smartKey = prefs.smartKey.stateIn(viewModelScope, SharingStarted.Eagerly, "")
    val serverIP = prefs.serverIP.stateIn(viewModelScope, SharingStarted.Eagerly, "")
    val autoConnect = prefs.autoConnect.stateIn(viewModelScope, SharingStarted.Eagerly, false)
    val bootConnect = prefs.bootConnect.stateIn(viewModelScope, SharingStarted.Eagerly, false)
    val dns = prefs.dns.stateIn(viewModelScope, SharingStarted.Eagerly, "1.1.1.1")
    val theme = prefs.theme.stateIn(viewModelScope, SharingStarted.Eagerly, "system")
    val dynamicColor = prefs.dynamicColor.stateIn(viewModelScope, SharingStarted.Eagerly, true)
    private val _installedApps = MutableStateFlow<List<AppInfo>>(emptyList())
    val installedApps: StateFlow<List<AppInfo>> = _installedApps.asStateFlow()
    private val _versionStatus = MutableStateFlow(VersionStatus.UNKNOWN)
    val versionStatus: StateFlow<VersionStatus> = _versionStatus.asStateFlow()
    private val _serverVersionStr = MutableStateFlow("")
    val serverVersionStr: StateFlow<String> = _serverVersionStr.asStateFlow()
    private var isToggling = false
    private val eventListener = object : golib.EventListener {
        override fun onStatusChanged(status: String) {
            _status.value = when (status) {
                "connecting" -> VpnStatus.CONNECTING
                "connected" -> VpnStatus.CONNECTED
                "reconnecting" -> VpnStatus.RECONNECTING
                "error" -> VpnStatus.ERROR
                else -> VpnStatus.DISCONNECTED
            }
        }
        override fun onLog(level: String, message: String) { addLog(level, message) }
        override fun onStatsUpdate(tx: Long, rx: Long) {
            _txSpeed.value = tx - lastTx; _rxSpeed.value = rx - lastRx
            lastTx = tx; lastRx = rx; _txBytes.value = tx; _rxBytes.value = rx
        }
        override fun onTurnInfo(url: String) { _turnServer.value = url }
    }
    init {
        Golib.setListener(eventListener)
        viewModelScope.launch {
            _status.collect { st ->
                if (st == VpnStatus.CONNECTED) {
                    val start = System.currentTimeMillis()
                    while (_status.value == VpnStatus.CONNECTED) {
                        _connectionTime.value = (System.currentTimeMillis() - start) / 1000
                        kotlinx.coroutines.delay(1000)
                    }
                } else if (st == VpnStatus.DISCONNECTED || st == VpnStatus.ERROR) {
                    _connectionTime.value = 0; _txBytes.value = 0; _rxBytes.value = 0
                    _txSpeed.value = 0; _rxSpeed.value = 0; _countryCode.value = ""; _pingMs.value = 0
                    lastTx = 0; lastRx = 0
                }
            }
        }
        viewModelScope.launch {
            combine(_status, serverIP) { s, ip -> s to ip }.collect { (status, ip) ->
                if (status == VpnStatus.CONNECTED && ip.isNotBlank() && _countryCode.value.isBlank()) lookupCountry(ip)
            }
        }
        viewModelScope.launch {
            try {
                val key = prefs.getSmartKeySync()
                if (key.isNotBlank() && serverRepo.getActive() == null) {
                    val ip = try { Golib.parseSmartKeyInfo(key) } catch (_: Exception) { "" }
                    serverRepo.addServer(ServerProfile(name = ip.ifBlank { "Сервер" }, smartKey = key, serverIP = ip))
                }
            } catch (_: Exception) {}
        }
    }
    fun measurePing() {
        _pingMs.value = -1
        viewModelScope.launch(Dispatchers.IO) {
            try {
                val url = activeServer.value?.pingUrl ?: "https://cp.cloudflare.com"
                val start = System.currentTimeMillis()
                val conn = URL(url).openConnection() as HttpURLConnection
                conn.connectTimeout = 5000; conn.readTimeout = 5000; conn.requestMethod = "HEAD"
                conn.connect(); val ms = System.currentTimeMillis() - start; conn.disconnect()
                _pingMs.value = ms
            } catch (_: Exception) { _pingMs.value = -2 }
        }
    }
    fun selectServer(id: String) = viewModelScope.launch {
        try { serverRepo.setActive(id) } catch (_: Exception) {}
    }
    fun addServerByKey(name: String, key: String, onSuccess: (() -> Unit)? = null) = viewModelScope.launch {
        var ok = false
        try {
            val k = key.trim()
            val ip = try { Golib.parseSmartKeyInfo(k) } catch (_: Exception) { "" }
            val server = ServerProfile(name = name.ifBlank { ip.ifBlank { "Сервер" } }, smartKey = k, serverIP = ip)
            withContext(Dispatchers.IO) {
                serverRepo.addServer(server)
                serverRepo.setActive(server.id)
            }
            ok = true
        } catch (_: Exception) {}
        _showAddKeyDialog.value = false
        if (ok) onSuccess?.invoke()
    }
    /** Удаление из списка + сброс активного сервера / prefs (общая логика для «удалить из приложения» и uninstall). */
    private suspend fun performRemoveServer(id: String) {
        val wasActive = activeServerId.value == id
        withContext(Dispatchers.IO) {
            serverRepo.removeServer(id)
            if (wasActive) {
                val remaining = serverRepo.servers.first()
                if (remaining.isNotEmpty()) {
                    serverRepo.setActive(remaining.first().id)
                } else {
                    prefs.setSmartKey("")
                    prefs.setServerIP("")
                    prefs.setActiveServerId("")
                }
            }
        }
    }

    fun removeServer(id: String, onDone: (() -> Unit)? = null) = viewModelScope.launch {
        try {
            performRemoveServer(id)
            onDone?.invoke()
        } catch (e: Exception) {
            addLog("error", "removeServer: ${e.message}")
        }
    }
    fun updateServerSetting(id: String, transform: (ServerProfile) -> ServerProfile) = viewModelScope.launch {
        try {
            val server = serverRepo.getById(id) ?: return@launch
            val updated = transform(server)
            serverRepo.updateServer(updated)
            if (id == activeServerId.value) {
                prefs.setDns(AdBlockConfig.getDns(updated.adBlock, updated.dns))
                prefs.setSmartKey(updated.smartKey)
                prefs.setServerIP(updated.serverIP)
            }
        } catch (_: Exception) {}
    }
    fun uninstallServer(id: String, sshPassword: String, onDone: () -> Unit) = viewModelScope.launch {
        try {
            val server = serverRepo.getById(id) ?: return@launch
            val r = ServerInstaller().uninstall(server.serverIP, server.sshPort, server.sshUser, sshPassword) { _setupState.value = it }
            if (r.isSuccess) {
                performRemoveServer(id)
                _setupState.value = null
                onDone()
            } else {
                _setupState.value = SetupProgress(0, 0, "", error = "Ошибка: ${r.exceptionOrNull()?.message ?: "unknown"}")
            }
        } catch (e: Exception) {
            _setupState.value = SetupProgress(0, 0, "", error = "Ошибка: ${e.message}")
        }
    }
    fun showAddByKey() { _showAddKeyDialog.value = true }
    fun hideAddKeyDialog() { _showAddKeyDialog.value = false }
    fun startSetup(host: String, port: Int, username: String, password: String, serverName: String) = viewModelScope.launch {
        try {
            ServerInstaller().install(host, port, username, password) { _setupState.value = it }.fold(
                onSuccess = { s -> val n = s.copy(name = serverName); serverRepo.addServer(n); serverRepo.setActive(n.id); lookupCountry(n.serverIP) },
                onFailure = { e -> _setupState.value = SetupProgress(0, 0, "", error = e.message ?: "Ошибка") }
            )
        } catch (e: Exception) {
            _setupState.value = SetupProgress(0, 0, "", error = e.message ?: "Ошибка")
        }
    }
    fun clearSetupState() { _setupState.value = null }
    fun resetAll() = viewModelScope.launch {
        try {
            serverRepo.clearAll()
            prefs.setAutoConnect(false); prefs.setBootConnect(false); prefs.setDns("1.1.1.1"); prefs.setTheme("system")
            _logs.value = emptyList()
        } catch (_: Exception) {}
    }
    fun connect() {
        if (smartKey.value.isBlank()) { addLog("error", "Smart key not set"); return }
        try {
            getApplication<Application>().startForegroundService(
                Intent(getApplication(), LionheartVpnService::class.java).apply { action = LionheartVpnService.ACTION_START }
            )
        } catch (e: Exception) {
            addLog("error", "connect: ${e.message}")
        }
    }
    fun disconnect() {
        try {
            getApplication<Application>().startService(
                Intent(getApplication(), LionheartVpnService::class.java).apply { action = LionheartVpnService.ACTION_STOP }
            )
        } catch (e: Exception) {
            _status.value = VpnStatus.DISCONNECTED
            addLog("error", "disconnect: ${e.message}")
        }
    }
    fun toggle() {
        if (isToggling) return
        isToggling = true
        if (_status.value == VpnStatus.CONNECTED || _status.value == VpnStatus.CONNECTING || _status.value == VpnStatus.RECONNECTING) {
            disconnect()
        } else {
            connect()
        }
        viewModelScope.launch {
            delay(800)
            isToggling = false
        }
    }
    fun needsVpnPermission(): Intent? = VpnService.prepare(getApplication())
    fun clearLogs() { _logs.value = emptyList() }

    /** Проверяет версию Lionheart на сервере. Нужен SSH-пароль. */
    fun checkServerVersion(serverId: String, sshPassword: String) = viewModelScope.launch(Dispatchers.IO) {
        try {
            val server = serverRepo.getById(serverId) ?: return@launch
            if (server.sshUser.isBlank()) {
                _versionStatus.value = VersionStatus.NO_SSH
                return@launch
            }
            _versionStatus.value = VersionStatus.CHECKING
            ServerInstaller().checkVersion(
                host = server.serverIP,
                port = server.sshPort,
                username = server.sshUser,
                password = sshPassword
            ).fold(
                onSuccess = { ver ->
                    _serverVersionStr.value = ver
                    serverRepo.updateServer(server.copy(serverVersion = ver))
                    _versionStatus.value = if (isVersionCompatible(ver)) VersionStatus.UP_TO_DATE else VersionStatus.OUTDATED
                },
                onFailure = {
                    _versionStatus.value = VersionStatus.UPDATE_FAILED
                    addLog("error", "Version check: ${it.message}")
                }
            )
        } catch (e: Exception) {
            _versionStatus.value = VersionStatus.UPDATE_FAILED
            addLog("error", "checkVersion: ${e.message}")
        }
    }

    /** Обновляет Lionheart на сервере до последней версии. */
    fun updateServer(serverId: String, sshPassword: String) = viewModelScope.launch(Dispatchers.IO) {
        try {
            val server = serverRepo.getById(serverId) ?: return@launch
            _versionStatus.value = VersionStatus.UPDATING
            ServerInstaller().updateServer(
                host = server.serverIP,
                port = server.sshPort,
                username = server.sshUser,
                password = sshPassword,
                onProgress = { _setupState.value = it }
            ).fold(
                onSuccess = { newVer ->
                    _serverVersionStr.value = newVer
                    serverRepo.updateServer(server.copy(serverVersion = newVer))
                    _versionStatus.value = VersionStatus.UPDATED
                    _setupState.value = null
                    addLog("info", "Server updated to v$newVer")
                },
                onFailure = {
                    _versionStatus.value = VersionStatus.UPDATE_FAILED
                    _setupState.value = SetupProgress(0, 0, "", error = it.message)
                    addLog("error", "Update failed: ${it.message}")
                }
            )
        } catch (e: Exception) {
            _versionStatus.value = VersionStatus.UPDATE_FAILED
            addLog("error", "updateServer: ${e.message}")
        }
    }

    fun resetVersionStatus() { _versionStatus.value = VersionStatus.UNKNOWN }

    /** true если серверная версия совместима с клиентом */
    private fun isVersionCompatible(serverVer: String): Boolean {
        if (serverVer.isBlank() || serverVer == "unknown" || serverVer == "not_installed") return false
        try {
            val sParts = serverVer.split(".").map { it.toInt() }
            val cParts = APP_VERSION.split(".").map { it.toInt() }
            // Мажорная версия должна совпадать, минорная серверная >= клиентской
            if (sParts.size < 2 || cParts.size < 2) return false
            return sParts[0] == cParts[0] && sParts[1] >= cParts[1]
        } catch (_: Exception) { return false }
    }

    /** Устарел ли сервер (по кэшированной версии, без SSH) */
    fun isServerOutdated(): Boolean {
        val server = activeServer.value ?: return false
        val ver = server.serverVersion
        if (ver.isBlank()) return false // неизвестно — не блокируем
        return !isVersionCompatible(ver)
    }

    fun loadInstalledApps() = viewModelScope.launch(Dispatchers.IO) {
        try {
            val pm = getApplication<Application>().packageManager
            _installedApps.value = pm.getInstalledApplications(PackageManager.GET_META_DATA)
                .filter { it.packageName != getApplication<Application>().packageName }
                .map { AppInfo(it.packageName, it.loadLabel(pm).toString(), (it.flags and ApplicationInfo.FLAG_SYSTEM) != 0) }
                .sortedWith(compareBy({ it.isSystem }, { it.label.lowercase() }))
        } catch (_: Exception) {}
    }
    fun setSmartKey(key: String) = viewModelScope.launch {
        prefs.setSmartKey(key); try { prefs.setServerIP(Golib.parseSmartKeyInfo(key)) } catch (_: Exception) { prefs.setServerIP("") }
    }
    fun setAutoConnect(v: Boolean) = viewModelScope.launch { prefs.setAutoConnect(v) }
    fun setBootConnect(v: Boolean) = viewModelScope.launch { prefs.setBootConnect(v) }
    fun setTheme(v: String) = viewModelScope.launch { prefs.setTheme(v) }
    fun setDynamicColor(v: Boolean) = viewModelScope.launch { prefs.setDynamicColor(v) }
    fun toggleSplitApp(serverId: String, pkg: String) = viewModelScope.launch {
        try {
            val s = serverRepo.getById(serverId) ?: return@launch
            val apps = s.splitApps.toMutableList()
            if (apps.contains(pkg)) apps.remove(pkg) else apps.add(pkg)
            serverRepo.updateServer(s.copy(splitApps = apps))
        } catch (_: Exception) {}
    }
    private fun lookupCountry(ip: String) = viewModelScope.launch(Dispatchers.IO) {
        try {
            val conn = URL("http://ip-api.com/json/$ip") as HttpURLConnection
            conn.connectTimeout = 5000; conn.readTimeout = 5000
            if (conn.responseCode == 200) {
                val code = JSONObject(conn.inputStream.bufferedReader().readText()).optString("countryCode", "")
                if (code.isNotBlank()) {
                    _countryCode.value = code
                    try { val a = serverRepo.getActive(); if (a != null && a.countryCode.isBlank()) serverRepo.updateServer(a.copy(countryCode = code)) } catch (_: Exception) {}
                }
            }; conn.disconnect()
        } catch (_: Exception) {}
    }
    private fun addLog(level: String, message: String) {
        _logs.update { (it + LogEntry(level = level, message = message)).takeLast(500) }
    }
}
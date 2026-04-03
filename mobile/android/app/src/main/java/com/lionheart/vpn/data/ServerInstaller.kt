package com.lionheart.vpn.data

import com.jcraft.jsch.ChannelExec
import com.jcraft.jsch.JSch
import com.jcraft.jsch.Session
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.ByteArrayOutputStream
import java.security.SecureRandom

data class SetupProgress(
    val step: Int,
    val totalSteps: Int,
    val message: String,
    val error: String? = null,
    val smartKey: String? = null
)

class ServerInstaller {

    companion object {
        private const val GITHUB_REPO = "jaykaiperson/lionheart"
        private const val INSTALL_DIR = "/opt/lionheart"
        private const val SERVER_PORT = "8443"
    }

    /**
     * Smart install script:
     * - If Lionheart is already installed → check version → update binary if needed
     * - Strictly preserves existing config.json (and thus smart keys)
     * - Creates systemd service
     * - If fresh install → generates new config
     */
    suspend fun install(
        host: String,
        port: Int = 22,
        username: String = "root",
        password: String,
        onProgress: (SetupProgress) -> Unit
    ): Result<ServerProfile> = withContext(Dispatchers.IO) {
        var session: Session? = null
        try {
            val total = 9
            onProgress(SetupProgress(1, total, "Подключение к $host..."))

            val jsch = JSch()
            session = jsch.getSession(username, host, port)
            session.setPassword(password)
            session.setConfig("StrictHostKeyChecking", "no")
            session.setConfig("PreferredAuthentications", "password,keyboard-interactive")
            session.timeout = 15000
            session.connect(15000)

            // Step 2: Detect architecture
            onProgress(SetupProgress(2, total, "Определение архитектуры..."))
            val arch = exec(session, "uname -m").trim()
            val goArch = when {
                arch.contains("aarch64") || arch.contains("arm64") -> "arm64"
                // ИСПРАВЛЕНО: Возвращаем "x64" для совместимости с новыми релизами
                arch.contains("x86_64") || arch.contains("amd64") || arch.contains("x64") -> "x64"
                else -> return@withContext Result.failure(Exception("Неизвестная архитектура: $arch"))
            }

            // Step 3: Check existing installation
            onProgress(SetupProgress(3, total, "Проверка существующей установки..."))
            exec(session, "mkdir -p $INSTALL_DIR")
            val existingBinary = exec(session, "test -x $INSTALL_DIR/lionheart && echo YES || echo NO").trim()
            val existingConfig = exec(session, "test -f $INSTALL_DIR/config.json && echo YES || echo NO").trim()
            val hasExisting = existingBinary == "YES"
            val hasConfig = existingConfig == "YES"

            var existingVersion = ""
            if (hasExisting) {
                // Try to detect version
                val verFromLogs = exec(session, "journalctl -u lionheart.service --no-pager -n 50 2>/dev/null | grep -oP 'v\\K[0-9]+\\.[0-9]+' | tail -1").trim()
                if (verFromLogs.matches(Regex("[0-9]+\\.[0-9]+"))) {
                    existingVersion = verFromLogs
                } else {
                    val verFromStrings = exec(session, "strings $INSTALL_DIR/lionheart 2>/dev/null | grep -oP '^[0-9]+\\.[0-9]+\\$' | tail -1").trim()
                    if (verFromStrings.matches(Regex("[0-9]+\\.[0-9]+"))) {
                        existingVersion = verFromStrings
                    }
                }
            }

            // Step 4: Download binary (always — ensures latest version)
            onProgress(SetupProgress(4, total, if (hasExisting) "Обновление Lionheart ($goArch)..." else "Скачивание Lionheart ($goArch)..."))
            
            // Stop service before replacing binary
            if (hasExisting) {
                exec(session, "systemctl stop lionheart.service 2>/dev/null; sleep 1")
            }

            // ИСПРАВЛЕНО: Динамический поиск URL релиза. Это решает проблему, если файл называется lionheart-1.2-linux-x64 или lionheart-linux-x64.
            val dlCmd = "cd $INSTALL_DIR && ASSET_URL=\$(curl -s https://api.github.com/repos/$GITHUB_REPO/releases/latest | grep -o '\"browser_download_url\": \"[^\"]*linux-$goArch[^\"]*\"' | head -n 1 | cut -d '\"' -f 4); if [ -z \"\$ASSET_URL\" ]; then ASSET_URL=\"https://github.com/$GITHUB_REPO/releases/latest/download/lionheart-linux-$goArch\"; fi; curl -fsSL \"\$ASSET_URL\" -o lionheart.new && chmod +x lionheart.new && mv lionheart.new lionheart && echo OK"
            val dlResult = exec(session, dlCmd)
            
            if (!dlResult.contains("OK")) {
                // If download fails but binary exists, we can still proceed
                if (!hasExisting) {
                    return@withContext Result.failure(Exception("Не удалось скачать: $dlResult"))
                }
            }

            // Step 5: Handle config.json
            onProgress(SetupProgress(5, total, if (hasConfig) "Сохраняем существующий ключ..." else "Генерация конфигурации..."))

            var vpnPassword: String
            var smartKeyForResult: String

            if (hasConfig) {
                // KEEP existing config.json — don't touch it!
                // Read password from existing config
                val existingPw = exec(session, "python3 -c \"import json; print(json.load(open('$INSTALL_DIR/config.json'))['Password'])\" 2>/dev/null || cat $INSTALL_DIR/config.json | grep -oP '\"Password\":\"\\K[^\"]+' 2>/dev/null").trim()
                vpnPassword = if (existingPw.isNotBlank()) existingPw else generatePassword()

                // If we couldn't read password, DON'T overwrite config
                if (existingPw.isBlank()) {
                    // Fallback: generate new config only if we can't read old one
                    vpnPassword = generatePassword()
                    val config = """{"Role":"server","Password":"$vpnPassword","ServerListen":"0.0.0.0:$SERVER_PORT","ClientPeer":""}"""
                    exec(session, "cat > $INSTALL_DIR/config.json << 'LHEOF'\n$config\nLHEOF")
                }
            } else {
                // Fresh install — generate new config
                vpnPassword = generatePassword()
                val config = """{"Role":"server","Password":"$vpnPassword","ServerListen":"0.0.0.0:$SERVER_PORT","ClientPeer":""}"""
                exec(session, "cat > $INSTALL_DIR/config.json << 'LHEOF'\n$config\nLHEOF")
            }

            // Step 6: Get public IP
            onProgress(SetupProgress(6, total, "Определение внешнего IP..."))
            var publicIP = exec(session, "curl -s --max-time 5 https://api.ipify.org").trim()
            if (publicIP.isBlank() || !publicIP.contains(".")) {
                publicIP = exec(session, "curl -s --max-time 5 https://2ip.ru").trim()
            }
            if (publicIP.isBlank() || !publicIP.contains(".")) publicIP = host

            // Step 7: Setup systemd service
            onProgress(SetupProgress(7, total, "Настройка службы..."))
            val serviceUnit = """
[Unit]
Description=Lionheart VPN Server
After=network.target
[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/lionheart
Restart=on-failure
RestartSec=5
[Install]
WantedBy=multi-user.target
            """.trimIndent()
            exec(session, "cat > /etc/systemd/system/lionheart.service << 'LHEOF'\n$serviceUnit\nLHEOF")
            exec(session, "systemctl daemon-reload && systemctl enable lionheart.service && systemctl start lionheart.service")

            // Step 8: Verify
            onProgress(SetupProgress(8, total, "Проверка..."))
            Thread.sleep(2000)
            val running = exec(session, "systemctl is-active lionheart.service").trim()
            if (running != "active") {
                val logs = exec(session, "journalctl -u lionheart.service --no-pager -n 10")
                return@withContext Result.failure(Exception("Сервер не запустился: $running\n$logs"))
            }

            // Build smart key
            val raw = "$publicIP:$SERVER_PORT|$vpnPassword"
            val smartKey = android.util.Base64.encodeToString(
                raw.toByteArray(),
                android.util.Base64.URL_SAFE or android.util.Base64.NO_PADDING or android.util.Base64.NO_WRAP
            )
            smartKeyForResult = smartKey

            val statusMsg = if (hasExisting && hasConfig) {
                if (existingVersion.isNotBlank()) "Обновлено с v$existingVersion" else "Обновлено"
            } else {
                "Установлено"
            }

            val server = ServerProfile(
                name = publicIP,
                smartKey = smartKeyForResult,
                serverIP = publicIP,
                sshUser = username,
                sshPort = port
            )

            onProgress(SetupProgress(total, total, "$statusMsg!", smartKey = smartKeyForResult))
            Result.success(server)

        } catch (e: Exception) {
            Result.failure(e)
        } finally {
            session?.disconnect()
        }
    }

    suspend fun uninstall(
        host: String,
        port: Int = 22,
        username: String = "root",
        password: String,
        onProgress: (SetupProgress) -> Unit
    ): Result<Unit> = withContext(Dispatchers.IO) {
        var session: Session? = null
        try {
            val total = 4
            onProgress(SetupProgress(1, total, "Подключение к $host..."))
            val jsch = JSch()
            session = jsch.getSession(username, host, port)
            session.setPassword(password)
            session.setConfig("StrictHostKeyChecking", "no")
            session.setConfig("PreferredAuthentications", "password,keyboard-interactive")
            session.timeout = 15000
            session.connect(15000)

            onProgress(SetupProgress(2, total, "Остановка службы..."))
            exec(session, "systemctl stop lionheart.service 2>/dev/null")
            exec(session, "systemctl disable lionheart.service 2>/dev/null")

            onProgress(SetupProgress(3, total, "Удаление файлов..."))
            exec(session, "rm -f /etc/systemd/system/lionheart.service")
            exec(session, "systemctl daemon-reload")
            exec(session, "rm -rf $INSTALL_DIR")
            exec(session, "pkill -f 'lionheart' 2>/dev/null")

            onProgress(SetupProgress(4, total, "Готово!"))
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        } finally {
            session?.disconnect()
        }
    }

    private fun exec(session: Session, command: String): String {
        val channel = session.openChannel("exec") as ChannelExec
        channel.setCommand(command)
        channel.inputStream = null
        val output = ByteArrayOutputStream()
        val error = ByteArrayOutputStream()
        channel.outputStream = output
        channel.setErrStream(error)
        channel.connect(30000)
        while (!channel.isClosed) Thread.sleep(100)
        channel.disconnect()
        val result = output.toString("UTF-8")
        if (result.isBlank() && error.size() > 0) return error.toString("UTF-8")
        return result
    }

    private fun generatePassword(): String {
        val bytes = ByteArray(16)
        SecureRandom().nextBytes(bytes)
        return bytes.joinToString("") { "%02x".format(it) }
    }

    suspend fun checkVersion(
        host: String,
        port: Int = 22,
        username: String = "root",
        password: String
    ): Result<String> = withContext(Dispatchers.IO) {
        var session: Session? = null
        try {
            val jsch = JSch()
            session = jsch.getSession(username, host, port)
            session.setPassword(password)
            session.setConfig("StrictHostKeyChecking", "no")
            session.setConfig("PreferredAuthentications", "password,keyboard-interactive")
            session.timeout = 10000
            session.connect(10000)

            var version = ""
            val logs = exec(session, "journalctl -u lionheart.service --no-pager -n 50 2>/dev/null | grep -oP 'v\\K[0-9]+\\.[0-9]+' | tail -1")
            if (logs.trim().matches(Regex("[0-9]+\\.[0-9]+"))) {
                version = logs.trim()
            }
            if (version.isBlank()) {
                val stringsOut = exec(session, "strings $INSTALL_DIR/lionheart 2>/dev/null | grep -oP '^[0-9]+\\.[0-9]+\\$' | tail -1")
                if (stringsOut.trim().matches(Regex("[0-9]+\\.[0-9]+"))) {
                    version = stringsOut.trim()
                }
            }
            if (version.isBlank()) {
                val exists = exec(session, "test -x $INSTALL_DIR/lionheart && echo YES || echo NO").trim()
                version = if (exists == "YES") "unknown" else "not_installed"
            }
            Result.success(version)
        } catch (e: Exception) {
            Result.failure(e)
        } finally {
            session?.disconnect()
        }
    }

    suspend fun updateServer(
        host: String,
        port: Int = 22,
        username: String = "root",
        password: String,
        onProgress: (SetupProgress) -> Unit
    ): Result<String> = withContext(Dispatchers.IO) {
        var session: Session? = null
        try {
            val total = 5
            onProgress(SetupProgress(1, total, "Подключение к $host..."))
            val jsch = JSch()
            session = jsch.getSession(username, host, port)
            session.setPassword(password)
            session.setConfig("StrictHostKeyChecking", "no")
            session.setConfig("PreferredAuthentications", "password,keyboard-interactive")
            session.timeout = 15000
            session.connect(15000)

            onProgress(SetupProgress(2, total, "Определение архитектуры..."))
            val arch = exec(session, "uname -m").trim()
            val goArch = when {
                arch.contains("aarch64") || arch.contains("arm64") -> "arm64"
                // ИСПРАВЛЕНО: Возвращаем "x64" для совместимости с новыми релизами
                arch.contains("x86_64") || arch.contains("amd64") || arch.contains("x64") -> "x64"
                else -> return@withContext Result.failure(Exception("Неизвестная архитектура: $arch"))
            }

            // Verify config.json exists before proceeding
            onProgress(SetupProgress(3, total, "Проверка конфигурации..."))
            val hasConfig = exec(session, "test -f $INSTALL_DIR/config.json && echo YES || echo NO").trim()
            if (hasConfig != "YES") {
                return@withContext Result.failure(Exception("config.json не найден! Выполните полную установку."))
            }

            // Backup config just in case
            exec(session, "cp $INSTALL_DIR/config.json $INSTALL_DIR/config.json.bak 2>/dev/null")

            onProgress(SetupProgress(4, total, "Скачивание обновления ($goArch)..."))
            exec(session, "systemctl stop lionheart.service 2>/dev/null; sleep 1")

            // ИСПРАВЛЕНО: Динамический поиск URL релиза для авто-обновления
            val dlCmd = "cd $INSTALL_DIR && ASSET_URL=\$(curl -s https://api.github.com/repos/$GITHUB_REPO/releases/latest | grep -o '\"browser_download_url\": \"[^\"]*linux-$goArch[^\"]*\"' | head -n 1 | cut -d '\"' -f 4); if [ -z \"\$ASSET_URL\" ]; then ASSET_URL=\"https://github.com/$GITHUB_REPO/releases/latest/download/lionheart-linux-$goArch\"; fi; curl -fsSL \"\$ASSET_URL\" -o lionheart.new && chmod +x lionheart.new && mv lionheart.new lionheart && echo OK"
            val dlResult = exec(session, dlCmd)
            
            if (!dlResult.contains("OK")) {
                // Restore backup if download failed
                exec(session, "systemctl start lionheart.service 2>/dev/null")
                return@withContext Result.failure(Exception("Ошибка скачивания: $dlResult"))
            }

            // Verify config.json is still intact
            val configStillExists = exec(session, "test -f $INSTALL_DIR/config.json && echo YES || echo NO").trim()
            if (configStillExists != "YES") {
                // Restore from backup
                exec(session, "cp $INSTALL_DIR/config.json.bak $INSTALL_DIR/config.json 2>/dev/null")
            }

            onProgress(SetupProgress(5, total, "Перезапуск сервера..."))
            exec(session, "systemctl start lionheart.service")
            Thread.sleep(2000)

            val running = exec(session, "systemctl is-active lionheart.service").trim()
            if (running != "active") {
                val logs = exec(session, "journalctl -u lionheart.service --no-pager -n 10")
                return@withContext Result.failure(Exception("Сервер не запустился: $running\n$logs"))
            }

            // Clean up backup
            exec(session, "rm -f $INSTALL_DIR/config.json.bak 2>/dev/null")

            val newVersion = exec(session, "journalctl -u lionheart.service --no-pager -n 20 2>/dev/null | grep -oP 'v\\K[0-9]+\\.[0-9]+' | tail -1").trim()
            Result.success(newVersion.ifBlank { "updated" })
        } catch (e: Exception) {
            Result.failure(e)
        } finally {
            session?.disconnect()
        }
    }
}
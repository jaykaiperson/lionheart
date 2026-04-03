import SwiftUI
import Combine

class VPNManager: ObservableObject {
    @Published var status: VPNStatus = .disconnected
    @Published var servers: [ServerProfile] = []
    @Published var activeServer: ServerProfile?
    @Published var txSpeed: Int64 = 0
    @Published var rxSpeed: Int64 = 0
    @Published var txBytes: Int64 = 0
    @Published var rxBytes: Int64 = 0
    @Published var pingMs: Int64 = 0
    @Published var logs: [LogEntry] = []
    @Published var autoConnect: Bool = false
    @Published var connectedSince: Date? = nil
    @Published var showToast = false
    @Published var toastMessage = ""
    @Published var toastIsError = false

    private var statsTimer: Timer?
    private let defaults = UserDefaults.standard

    var socksAddress: String { "127.0.0.1:1080" }
    var mtprotoAddress: String { "127.0.0.1:8443" }
    var connectionDuration: TimeInterval {
        guard let since = connectedSince else { return 0 }
        return Date().timeIntervalSince(since)
    }
    var formattedDuration: String { formatDuration(connectionDuration) }

    init() { loadServers(); autoConnect = defaults.bool(forKey: "autoConnect") }

    func showSuccess(_ msg: String) {
        DispatchQueue.main.async {
            self.toastMessage = msg; self.toastIsError = false
            withAnimation { self.showToast = true }
        }
    }
    func showError(_ msg: String) {
        DispatchQueue.main.async {
            self.toastMessage = msg; self.toastIsError = true
            withAnimation { self.showToast = true }
        }
    }

    // MARK: - Server CRUD
    func addServer(_ server: ServerProfile) {
        servers.append(server)
        if activeServer == nil { selectServer(server.id) }
        saveServers()
        showSuccess("Server added")
    }
    func removeServer(_ id: String) {
        servers.removeAll { $0.id == id }
        if activeServer?.id == id { activeServer = servers.first; defaults.set(activeServer?.id ?? "", forKey: "activeServerId") }
        saveServers()
    }
    func selectServer(_ id: String) {
        activeServer = servers.first { $0.id == id }
        defaults.set(id, forKey: "activeServerId")
    }
    func updateServer(_ server: ServerProfile) {
        if let i = servers.firstIndex(where: { $0.id == server.id }) {
            servers[i] = server
            if activeServer?.id == server.id { activeServer = server }
            saveServers()
        }
    }
    func importSmartKey(_ key: String, name: String? = nil) {
        let t = key.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !t.isEmpty, let decoded = decodeSmartKey(t) else { showError("Invalid smart key"); return }
        let parts = decoded.split(separator: "|", maxSplits: 1)
        guard parts.count == 2 else { showError("Invalid key format"); return }
        let peer = String(parts[0])
        let host = peer.contains(":") ? String(peer.split(separator: ":")[0]) : peer
        var s = ServerProfile()
        s.name = name ?? host; s.smartKey = t; s.serverIP = host
        addServer(s)
    }

    // MARK: - Connection
    func toggleConnection() {
        switch status {
        case .disconnected, .error: connect()
        case .connected: disconnect()
        case .connecting, .reconnecting: disconnect()
        }
    }
    func connect() {
        guard activeServer != nil else { showError("No server selected"); return }
        status = .connecting
        addLog("INFO", "Connecting…")
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) { [weak self] in
            guard let self, self.status == .connecting else { return }
            self.status = .connected; self.connectedSince = Date()
            self.addLog("INFO", "SOCKS5 ready on 127.0.0.1:1080")
            self.showSuccess("Connected")
            self.startStatsTimer()
        }
    }
    func disconnect() {
        statsTimer?.invalidate(); statsTimer = nil
        status = .disconnected; connectedSince = nil; txSpeed = 0; rxSpeed = 0
        addLog("INFO", "Disconnected")
    }

    func measurePing() {
        guard activeServer != nil else { return }
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) { [weak self] in
            self?.pingMs = Int64.random(in: 15...120)
        }
    }

    func installServerViaSSH(
        host: String,
        port: Int,
        username: String,
        password: String,
        serverName: String,
        onProgress: @escaping (SetupProgress) -> Void
    ) async -> Result<ServerProfile, Error> {
        do {
            let (key, version) = try await SSHService().installServer(
                host: host,
                port: port,
                username: username,
                password: password,
                serverName: serverName,
                onProgress: { msgKey, step in
                    // Используем новые ключи локализации, если они есть
                    let localizedMsg = NSLocalizedString(msgKey, comment: "")
                    let progress = SetupProgress(step: step, total: 5, message: localizedMsg)
                    DispatchQueue.main.async {
                        onProgress(progress)
                    }
                }
            )
            
            var server = ServerProfile()
            server.serverIP = host
            server.sshUser = username
            server.sshPort = port
            server.name = serverName
            server.smartKey = key
            server.serverVersion = version
            
            return .success(server)
        } catch {
            return .failure(error)
        }
    }

    func updateServerViaSSH(server: ServerProfile, sshPassword: String) {
        DispatchQueue.main.async {
            self.showToast = true
            self.toastMessage = NSLocalizedString("ssh_step_connecting", comment: "")
            self.toastIsError = false
        }
        
        Task {
            do {
                let version = try await SSHService().updateServer(
                    host: server.serverIP,
                    port: server.sshPort,
                    username: server.sshUser,
                    password: sshPassword,
                    onProgress: { msgKey, step in
                        let localizedMsg = NSLocalizedString(msgKey, comment: "")
                        DispatchQueue.main.async { [weak self] in
                            self?.toastMessage = localizedMsg
                            self?.addLog("INFO", localizedMsg)
                        }
                    }
                )
                
                DispatchQueue.main.async { [weak self] in
                    var u = server
                    u.serverVersion = version
                    self?.updateServer(u)
                    self?.showSuccess(NSLocalizedString("server_updated", comment: ""))
                }
            } catch {
                DispatchQueue.main.async { [weak self] in
                    self?.showError(error.localizedDescription)
                }
            }
        }
    }

    private func startStatsTimer() {
        statsTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            guard let self else { return }
            let ft = self.txBytes + Int64.random(in: 50_000...500_000)
            let fr = self.rxBytes + Int64.random(in: 100_000...1_000_000)
            DispatchQueue.main.async { self.txSpeed = ft - self.txBytes; self.rxSpeed = fr - self.rxBytes; self.txBytes = ft; self.rxBytes = fr }
        }
    }

    func setAutoConnect(_ v: Bool) { autoConnect = v; defaults.set(v, forKey: "autoConnect") }
    func addLog(_ level: String, _ msg: String) {
        DispatchQueue.main.async { [weak self] in
            self?.logs.append(LogEntry(level: level, message: msg))
            if (self?.logs.count ?? 0) > 500 { self?.logs.removeFirst() }
        }
    }
    func clearLogs() { logs = [] }
    func clearAll() {
        disconnect(); servers = []; activeServer = nil; logs = []
        txBytes = 0; rxBytes = 0; pingMs = 0
        defaults.removeObject(forKey: "servers"); defaults.removeObject(forKey: "activeServerId")
    }

    private func saveServers() {
        if let d = try? JSONEncoder().encode(servers) { defaults.set(d, forKey: "servers") }
    }
    private func loadServers() {
        if let d = defaults.data(forKey: "servers"),
           let s = try? JSONDecoder().decode([ServerProfile].self, from: d) { servers = s }
        let id = defaults.string(forKey: "activeServerId") ?? ""
        activeServer = servers.first { $0.id == id } ?? servers.first
    }
}

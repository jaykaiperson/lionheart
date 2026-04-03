import SwiftUI

// Цвета приложения
extension Color {
    static let appAccent = Color(red: 0.35, green: 0.68, blue: 0.95)
}

struct ServerProfile: Identifiable, Codable, Equatable {
    var id: String          = UUID().uuidString
    var name: String        = ""
    var smartKey: String    = ""
    var serverIP: String    = ""
    var countryCode: String = ""
    var dns: String         = "1.1.1.1"
    var mtu: Int            = 1500
    var ipMode: IPMode      = .preferV4
    var adBlock: Bool       = false
    var killSwitch: Bool    = false
    var connectionMode: ConnectionMode = .socks5
    var showIP: Bool        = true
    var pingUrl: String     = "https://cp.cloudflare.com"
    var createdAt: Date     = Date()
    
    var sshUser: String     = "root"
    var sshPort: Int        = 22
    var serverVersion: String = ""
    
    var displayName: String { name.isEmpty ? serverIP : name }
    
    var displayDns: String {
        if adBlock { return "AdGuard DNS" }
        if let custom = ServerProfile.defaultDNS.first(where: { $0.value == dns }) {
            return custom.label
        }
        return dns
    }
    
    var effectiveDns: String { adBlock ? "94.140.14.14" : dns }
    
    var shareURL: String {
        let k = smartKey.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? smartKey
        let n = name.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? name
        return "lionheart://import?key=\(k)&name=\(n)"
    }
    
    static let defaultDNS: [(label: String, value: String)] = [
        ("Cloudflare (1.1.1.1)", "1.1.1.1"),
        ("Google (8.8.8.8)", "8.8.8.8"),
        ("Quad9 (9.9.9.9)", "9.9.9.9")
    ]
    
    static let countries: [(code: String, key: String)] = [
        ("US", "country_US"), ("GB", "country_GB"), ("DE", "country_DE"),
        ("NL", "country_NL"), ("FR", "country_FR"), ("FI", "country_FI"),
        ("SE", "country_SE"), ("PL", "country_PL"), ("LT", "country_LT"),
        ("LV", "country_LV"), ("EE", "country_EE"), ("RU", "country_RU"),
        ("UA", "country_UA"), ("BY", "country_BY"), ("KZ", "country_KZ"),
        ("TR", "country_TR"), ("JP", "country_JP"), ("SG", "country_SG"),
        ("IN", "country_IN"), ("AU", "country_AU"), ("CA", "country_CA"),
        ("BR", "country_BR"), ("HK", "country_HK"), ("TJ", "Tajikistan")
    ]
}

enum VPNStatus: String {
    case disconnected, connecting, connected, reconnecting, error
    
    var displayName: LocalizedStringKey {
        switch self {
        case .disconnected: return "status_disconnected"
        case .connecting: return "status_connecting"
        case .connected: return "status_connected"
        case .reconnecting: return "status_reconnecting"
        case .error: return "status_error"
        }
    }
    
    var color: Color {
        switch self {
        case .disconnected: return .gray
        case .connecting, .reconnecting: return .orange
        case .connected: return .green
        case .error: return .red
        }
    }
    
    var isTransitioning: Bool { self == .connecting || self == .reconnecting }
    
    var iconName: String {
        switch self {
        case .disconnected: return "shield.slash.fill"
        case .connecting, .reconnecting: return "shield.lefthalf.filled"
        case .connected: return "checkmark.shield.fill"
        case .error: return "xmark.shield.fill"
        }
    }
}

enum ConnectionMode: String, Codable, CaseIterable {
    case socks5, mtproto
    
    var displayName: String {
        switch self {
        case .socks5: return "SOCKS5"
        case .mtproto: return "MTProto"
        }
    }
    
    var icon: String {
        switch self {
        case .socks5: return "network"
        case .mtproto: return "paperplane.fill"
        }
    }
}

enum IPMode: String, Codable, CaseIterable {
    case preferV4, onlyV4, preferV6, onlyV6
    
    var label: LocalizedStringKey {
        switch self {
        case .preferV4: return "prefer_ipv4"
        case .onlyV4: return "only_ipv4"
        case .preferV6: return "prefer_ipv6"
        case .onlyV6: return "only_ipv6"
        }
    }
}

struct LogEntry: Identifiable {
    let id = UUID()
    let timestamp: Date
    let level: String
    let message: String
    
    init(timestamp: Date = Date(), level: String, message: String) {
        self.timestamp = timestamp
        self.level = level
        self.message = message
    }
}

struct SetupProgress {
    let step: Int
    let total: Int
    let message: String
    var smartKey: String? = nil
    var error: String? = nil
}

enum AppIcon: String, CaseIterable, Identifiable {
    case standard = "AppIcon"
    case sage = "AppIcon-Sage"
    case forest = "AppIcon-Forest"
    case indigo = "AppIcon-Indigo"
    case plum = "AppIcon-Plum"
    case crimson = "AppIcon-Crimson"
    
    var id: String { rawValue }
    
    var displayName: LocalizedStringKey {
        switch self {
        case .standard: return "icon_standard"
        case .sage: return "icon_sage"
        case .forest: return "icon_forest"
        case .indigo: return "icon_indigo"
        case .plum: return "icon_plum"
        case .crimson: return "icon_crimson"
        }
    }
    
    // SwiftUI сам нарисует превью нужного цвета
    var previewColor: Color {
        switch self {
        case .standard: return Color(red: 0.82, green: 0.71, blue: 0.62) // Бежевый
        case .sage: return Color(red: 0.45, green: 0.55, blue: 0.45) // Шалфей
        case .forest: return Color(red: 0.18, green: 0.35, blue: 0.22) // Лесной
        case .indigo: return Color(red: 0.25, green: 0.25, blue: 0.55) // Индиго
        case .plum: return Color(red: 0.35, green: 0.15, blue: 0.25) // Сливовый
        case .crimson: return Color(red: 0.55, green: 0.15, blue: 0.15) // Бордовый
        }
    }
    
    var iconName: String? {
        self == .standard ? nil : rawValue
    }
    
    static var current: AppIcon {
        if let name = UIApplication.shared.alternateIconName {
            return AppIcon(rawValue: name) ?? .standard
        }
        return .standard
    }
    
    func apply() {
        let name = iconName
        UIApplication.shared.setAlternateIconName(name) { error in
            if let error = error {
                print("Error changing app icon: \(error.localizedDescription)")
            }
        }
    }
}

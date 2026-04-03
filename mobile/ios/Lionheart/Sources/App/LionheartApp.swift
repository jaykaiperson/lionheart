import SwiftUI
import TipKit

@main
struct LionheartApp: App {
    @StateObject private var vpnManager = VPNManager()
    @StateObject private var appSettings = AppSettings()

    init() {
        try? Tips.configure([
            .displayFrequency(.immediate),
            .datastoreLocation(.applicationDefault)
        ])
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(vpnManager)
                .environmentObject(appSettings)
                .environment(\.locale, appSettings.locale)
                .tint(Color.appAccent)
                .onOpenURL { url in
                    guard url.scheme == "lionheart",
                          let c = URLComponents(url: url, resolvingAgainstBaseURL: true),
                          c.host == "import" else { return }
                    let params = Dictionary(uniqueKeysWithValues:
                        (c.queryItems ?? []).compactMap { i in i.value.map { (i.name, $0) } })
                    if let key = params["key"] {
                        vpnManager.importSmartKey(key, name: params["name"])
                    }
                }
        }
    }
}

struct RootView: View {
    @EnvironmentObject var appSettings: AppSettings
    var body: some View {
        if appSettings.hasCompletedOnboarding {
            ContentView()
        } else {
            OnboardingView()
        }
    }
}

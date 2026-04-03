import SwiftUI

enum AppTab: Int {
    case dashboard, servers, logs, settings
}

struct ContentView: View {
    @EnvironmentObject var vm: VPNManager
    @State private var selectedTab: AppTab = .dashboard
    @State private var showAddServer = false
    
    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem { Label("Home", systemImage: "shield.fill") }
                .tag(AppTab.dashboard)
            
            ServerListView(showAddServer: $showAddServer)
                .tabItem { Label("servers", systemImage: "server.rack") } // Используем ключ
                .tag(AppTab.servers)
            
            LogsView()
                .tabItem { Label("logs", systemImage: "doc.text.fill") } // Используем ключ
                .tag(AppTab.logs)
            
            SettingsView()
                .tabItem { Label("settings", systemImage: "gearshape.fill") } // Используем ключ
                .tag(AppTab.settings)
        }
        .onAppear {
            if vm.servers.isEmpty {
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                    selectedTab = .servers
                }
            }
        }
    }
}

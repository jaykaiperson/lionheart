import SwiftUI
import TipKit

struct ServerListView: View {
    @EnvironmentObject var vm: VPNManager
    @Binding var showAddServer: Bool
    @State private var serverToDelete: ServerProfile?
    
    var body: some View {
        NavigationView {
            ZStack {
                if vm.servers.isEmpty {
                    emptyState
                } else {
                    List {
                        ForEach(vm.servers) { server in
                            serverRow(server)
                        }
                        .onDelete { indexSet in
                            for index in indexSet {
                                serverToDelete = vm.servers[index]
                            }
                        }
                    }
                    .listStyle(InsetGroupedListStyle())
                }
            }
            .navigationTitle("servers") // ИСПРАВЛЕНИЕ: Ключ локализации с маленькой буквы
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { showAddServer = true }) {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(isPresented: $showAddServer) {
                AddServerWizard()
                    .environmentObject(vm)
            }
            .alert("delete_server_confirm", isPresented: Binding(
                get: { serverToDelete != nil },
                set: { if !$0 { serverToDelete = nil } }
            )) {
                Button("cancel", role: .cancel) { }
                Button("delete", role: .destructive) {
                    if let s = serverToDelete {
                        withAnimation { vm.removeServer(s.id) }
                    }
                }
            } message: {
                Text("delete_server_msg")
            }
        }
    }
    
    private func serverRow(_ server: ServerProfile) -> some View {
        NavigationLink(destination: ServerDetailView(serverId: server.id)) {
            HStack(spacing: 16) {
                ZStack {
                    Circle()
                        .fill(vm.activeServer?.id == server.id ? Color.green.opacity(0.2) : Color.gray.opacity(0.2))
                        .frame(width: 48, height: 48)
                    flagView(server.countryCode)
                }
                
                VStack(alignment: .leading, spacing: 4) {
                    Text(server.displayName)
                        .font(.headline)
                    Text(server.connectionMode.displayName)
                        .font(.caption)
                        .foregroundColor(.gray)
                }
                
                Spacer()
                
                if vm.activeServer?.id == server.id {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundColor(.green)
                        .font(.title3)
                }
            }
            .padding(.vertical, 4)
        }
        .swipeActions(edge: .leading) {
            Button {
                vm.selectServer(server.id)
            } label: {
                Label("select_server", systemImage: "checkmark")
            }
            .tint(.green)
        }
    }
    
    private var emptyState: some View {
        VStack(spacing: 20) {
            Image(systemName: "server.rack")
                .font(.system(size: 60))
                .foregroundColor(.gray)
            Text("add_first_server")
                .font(.title2.bold())
            Text("add_first_server_desc")
                .foregroundColor(.gray)
            
            Button(action: { showAddServer = true }) {
                Text("add_server")
            }
            .buttonStyle(PremiumButtonStyle())
            .padding(.horizontal, 40)
            .padding(.top, 10)
        }
    }
}

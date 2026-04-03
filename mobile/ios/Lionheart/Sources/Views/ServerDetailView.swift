import SwiftUI

struct ServerDetailView: View {
    @EnvironmentObject var vm: VPNManager
    let serverId: String
    @State private var server = ServerProfile()
    @State private var showDeleteAlert = false
    @State private var showSSHPassword = false
    @State private var showQRShare = false
    @State private var sshPassword = ""
    @Environment(\.dismiss) var dismiss

    var body: some View {
        Form {
            Section("server_section") {
                HStack { Text("name"); Spacer(); TextField("server_name_hint", text: $server.name).multilineTextAlignment(.trailing).foregroundStyle(.secondary).onSubmit { save() } }
                if !server.countryCode.isEmpty {
                    HStack { Text("country"); Spacer(); flagView(server.countryCode); Text(LocalizedStringKey(ServerProfile.countries.first { $0.code == server.countryCode }?.key ?? server.countryCode)).foregroundStyle(.secondary) }
                }
                HStack { Text("IP"); Spacer(); Button { withAnimation { server.showIP.toggle(); save() } } label: { Text(server.showIP ? server.serverIP : "••••••••").foregroundStyle(.secondary) } }
                if !server.serverVersion.isEmpty { HStack { Text("server_version"); Spacer(); Text("v\(server.serverVersion)").foregroundStyle(.secondary) } }
            }
            Section("connection_mode") {
                Picker("mode_label", selection: $server.connectionMode) {
                    ForEach(ConnectionMode.allCases, id: \.self) { Label($0.displayName, systemImage: $0.icon).tag($0) }
                }.pickerStyle(.navigationLink).onChange(of: server.connectionMode) { _, _ in save() }
            }
            Section("network") {
                Picker("dns_server", selection: $server.dns) {
                    ForEach(ServerProfile.defaultDNS, id: \.value) { Text($0.label).tag($0.value) }
                }.pickerStyle(.navigationLink).onChange(of: server.dns) { _, _ in save() }
                Picker("mtu_label", selection: $server.mtu) { Text("1500").tag(1500); Text("1400").tag(1400); Text("1280").tag(1280) }.onChange(of: server.mtu) { _, _ in save() }
                Picker("ip_protocol", selection: $server.ipMode) { ForEach(IPMode.allCases, id: \.self) { Text($0.label).tag($0) } }.onChange(of: server.ipMode) { _, _ in save() }
            }
            Section { Toggle("ad_block", isOn: $server.adBlock).onChange(of: server.adBlock) { _, _ in save() }; Toggle("kill_switch", isOn: $server.killSwitch).onChange(of: server.killSwitch) { _, _ in save() } } header: { Text("security") } footer: { Text("kill_switch_desc") }
            Section("Share") {
                ShareLink(item: server.shareURL, preview: SharePreview(server.displayName, image: Image(systemName: "server.rack"))) { Label("Share Configuration", systemImage: "square.and.arrow.up") }
                Button { showQRShare = true } label: { Label("Show QR Code", systemImage: "qrcode") }
            }
            Section("management") {
                Button { vm.selectServer(server.id) } label: { Label("select_server", systemImage: "checkmark.circle") }.disabled(vm.activeServer?.id == server.id)
                Button { showSSHPassword = true } label: { Label("update_btn", systemImage: "arrow.triangle.2.circlepath") }
            }
            Section { Button(role: .destructive) { showDeleteAlert = true } label: { Label { Text("remove_from_app").foregroundStyle(.red) } icon: { Image(systemName: "trash").foregroundStyle(.red) } } }
        }
        .navigationTitle(server.displayName)
        .navigationBarTitleDisplayMode(.inline)
        .sensoryFeedback(.selection, trigger: server.connectionMode)
        .onAppear { if let s = vm.servers.first(where: { $0.id == serverId }) { server = s } }
        .alert("delete_server_confirm", isPresented: $showDeleteAlert) { Button("cancel", role: .cancel) {}; Button("delete", role: .destructive) { vm.removeServer(server.id); dismiss() } } message: { Text("delete_server_msg") }
        .sheet(isPresented: $showSSHPassword) { SSHPasswordSheet(server: server, sshPassword: $sshPassword) { vm.updateServerViaSSH(server: server, sshPassword: sshPassword); showSSHPassword = false } }
        .sheet(isPresented: $showQRShare) { QRShareSheet(server: server) }
    }

    private func save() { vm.updateServer(server) }
}

struct SSHPasswordSheet: View {
    let server: ServerProfile; @Binding var sshPassword: String; let onUpdate: () -> Void
    @Environment(\.dismiss) var dismiss
    var body: some View {
        NavigationStack {
            Form {
                Section { SecureField("ssh_password", text: $sshPassword) } footer: { Text("step_auth_hint") }
                Section { Button(action: onUpdate) { Text("update_btn").frame(maxWidth: .infinity) }.buttonStyle(PremiumButtonStyle(colors: [.green, Color(red: 0.3, green: 0.7, blue: 0.4)])).disabled(sshPassword.isEmpty).listRowBackground(Color.clear).listRowInsets(EdgeInsets()) }
            }
            .navigationTitle("update_confirm_title").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("cancel") { dismiss() } } }
        }.presentationDetents([.medium])
    }
}

struct QRShareSheet: View {
    let server: ServerProfile; @Environment(\.dismiss) var dismiss
    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Text(server.displayName).font(.headline)
                Image(uiImage: generateQRCode(from: server.shareURL)).interpolation(.none).resizable().scaledToFit().frame(width: 220, height: 220).padding().background(Color.white).clipShape(RoundedRectangle(cornerRadius: 16))
                Text("Scan with another device running Lionheart").font(.footnote).foregroundStyle(.secondary).multilineTextAlignment(.center).padding(.horizontal)
                ShareLink(item: server.shareURL, preview: SharePreview(server.displayName)) { Label("Share Link", systemImage: "link") }.buttonStyle(SecondaryButtonStyle()).padding(.horizontal)
            }.padding()
            .toolbar { ToolbarItem(placement: .confirmationAction) { Button("done") { dismiss() } } }
        }.presentationDetents([.medium, .large])
    }
}

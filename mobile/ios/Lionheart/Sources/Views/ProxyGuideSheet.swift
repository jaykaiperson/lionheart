import SwiftUI

struct ProxyGuideSheet: View {
    @EnvironmentObject var vm: VPNManager
    @Environment(\.dismiss) var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 20) {
                    VStack(spacing: 12) {
                        proxyCard(title: "SOCKS5", address: vm.socksAddress, icon: "network", color: .appAccent)
                        proxyCard(title: "MTProto", address: vm.mtprotoAddress, icon: "paperplane.fill", color: .purple)
                    }
                    VStack(alignment: .leading, spacing: 16) {
                        Text("Setup Instructions").font(.headline)
                        instructionRow(step: "1", icon: "wifi", color: .appAccent, text: "proxy_step_wifi")
                        instructionRow(step: "2", icon: "paperplane.fill", color: .blue, text: "proxy_step_telegram")
                        instructionRow(step: "3", icon: "app.badge", color: .orange, text: "proxy_step_apps_hint")
                    }
                    HStack(spacing: 12) {
                        Image(systemName: "exclamationmark.triangle.fill").font(.title3).foregroundStyle(.orange)
                        Text("proxy_keep_open_warning").font(.footnote).foregroundStyle(.secondary)
                    }
                    .padding().background(Color.orange.opacity(0.1)).clipShape(RoundedRectangle(cornerRadius: 12))
                }.padding()
            }
            .navigationTitle("proxy_info").navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .confirmationAction) { Button("done") { dismiss() } } }
        }.presentationDetents([.large])
    }

    private func proxyCard(title: String, address: String, icon: String, color: Color) -> some View {
        HStack {
            Image(systemName: icon).font(.title3).foregroundStyle(.white)
                .frame(width: 40, height: 40).background(color.gradient)
                .clipShape(RoundedRectangle(cornerRadius: 9, style: .continuous))
            VStack(alignment: .leading, spacing: 2) { Text(title).font(.headline); Text(address).font(.subheadline.monospaced()).foregroundStyle(.secondary) }
            Spacer()
            Button { UIPasteboard.general.string = address; vm.showSuccess("Copied") } label: { Image(systemName: "doc.on.doc") }.buttonStyle(.bordered).controlSize(.small)
        }
        .padding().background(Color(.secondarySystemGroupedBackground)).clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
    }

    private func instructionRow(step: String, icon: String, color: Color, text: LocalizedStringKey) -> some View {
        HStack(alignment: .top, spacing: 12) {
            Text(step).font(.caption.bold()).foregroundStyle(.white).frame(width: 24, height: 24).background(color).clipShape(Circle())
            Text(text).font(.subheadline).foregroundStyle(.secondary)
        }
    }
}

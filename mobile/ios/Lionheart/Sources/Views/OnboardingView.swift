import SwiftUI

struct OnboardingView: View {
    @EnvironmentObject var appSettings: AppSettings
    @State private var page = 0

    private let pages: [OBPage] = [
        OBPage(icon: "shield.checkmark.fill", iconColor: .appAccent,
            titleKey: "onboarding_title_1", descKey: "onboarding_desc_1",
            features: [
                OBFeature(icon: "lock.shield.fill", color: .blue, titleKey: "encryption", descKey: "AES-256-CFB"),
                OBFeature(icon: "antenna.radiowaves.left.and.right", color: .green, titleKey: "relay", descKey: "TURN (WebRTC)"),
                OBFeature(icon: "theatermasks.fill", color: .purple, titleKey: "traffic_disguised", descKey: ""),
            ]),
        OBPage(icon: "server.rack", iconColor: .orange,
            titleKey: "onboarding_title_2", descKey: "onboarding_desc_2",
            features: [
                OBFeature(icon: "bolt.fill", color: .green, titleKey: "auto_setup", descKey: ""),
                OBFeature(icon: "key.fill", color: .orange, titleKey: "enter_smart_key", descKey: ""),
                OBFeature(icon: "qrcode.viewfinder", color: .blue, titleKey: "scan_qr", descKey: ""),
            ]),
        OBPage(icon: "network", iconColor: .teal,
            titleKey: "onboarding_title_3", descKey: "onboarding_desc_3", features: []),
        OBPage(icon: "globe", iconColor: .purple,
            titleKey: "onboarding_title_4", descKey: "onboarding_desc_4", features: []),
    ]

    var body: some View {
        VStack(spacing: 0) {
            TabView(selection: $page) {
                ForEach(Array(pages.enumerated()), id: \.offset) { index, pg in
                    pageContent(pg).tag(index)
                }
            }
            .tabViewStyle(.page(indexDisplayMode: .always))

            VStack(spacing: 12) {
                if page == pages.count - 1 {
                    Button {
                        withAnimation(.spring(response: 0.4)) { appSettings.hasCompletedOnboarding = true }
                    } label: { Text("get_started") }
                    .buttonStyle(PremiumButtonStyle())
                } else {
                    Button { withAnimation { page += 1 } } label: { Text("Next") }
                    .buttonStyle(PremiumButtonStyle())
                }
            }
            .padding(.horizontal, 24)
            .padding(.bottom, 40)
        }
        .sensoryFeedback(.selection, trigger: page)
    }

    private func pageContent(_ pg: OBPage) -> some View {
        VStack(spacing: 0) {
            Spacer()
            OBIcon(icon: pg.icon, color: pg.iconColor).padding(.bottom, 24)
            Text(LocalizedStringKey(pg.titleKey)).font(.largeTitle.bold()).multilineTextAlignment(.center).padding(.horizontal, 32)
            Text(LocalizedStringKey(pg.descKey)).font(.body).foregroundStyle(.secondary).multilineTextAlignment(.center).padding(.horizontal, 40).padding(.top, 8)
            Spacer()
            if !pg.features.isEmpty {
                VStack(alignment: .leading, spacing: 18) {
                    ForEach(pg.features) { f in
                        HStack(spacing: 14) {
                            Image(systemName: f.icon).font(.title3).foregroundStyle(.white)
                                .frame(width: 40, height: 40)
                                .background(f.color.gradient)
                                .clipShape(RoundedRectangle(cornerRadius: 9, style: .continuous))
                            VStack(alignment: .leading, spacing: 2) {
                                Text(f.titleKey).font(.subheadline.weight(.semibold))
                                if !f.descKey.isEmpty { Text(f.descKey).font(.caption).foregroundStyle(.secondary) }
                            }
                        }
                    }
                }.padding(.horizontal, 40)
            }
            Spacer()
        }
    }
}

private struct OBIcon: View {
    let icon: String; let color: Color
    @State private var appeared = false
    var body: some View {
        Image(systemName: icon).font(.system(size: 72)).foregroundStyle(color.gradient)
            .symbolEffect(.bounce, value: appeared).onAppear { appeared = true }
    }
}

private struct OBPage { let icon: String; let iconColor: Color; let titleKey: String; let descKey: String; let features: [OBFeature] }
private struct OBFeature: Identifiable { let id = UUID(); let icon: String; let color: Color; let titleKey: LocalizedStringKey; let descKey: String }

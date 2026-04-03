import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var vm: VPNManager
    @EnvironmentObject var appSettings: AppSettings
    @State private var showResetAlert = false
    @State private var selectedIcon = AppIcon.current
    
    var body: some View {
        NavigationView {
            List {
                Section(header: Text("appearance")) {
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 20) {
                            ForEach(AppIcon.allCases) { icon in
                                VStack(spacing: 8) {
                                    // ИСПРАВЛЕНИЕ: Рисуем иконку кодом, без картинок!
                                    ZStack {
                                        RoundedRectangle(cornerRadius: 14)
                                            .fill(icon.previewColor)
                                            .frame(width: 64, height: 64)
                                        
                                        Image(systemName: "shield.fill")
                                            .resizable()
                                            .scaledToFit()
                                            .frame(width: 28, height: 28)
                                            .foregroundColor(.white)
                                    }
                                    .overlay(
                                        RoundedRectangle(cornerRadius: 14)
                                            .stroke(selectedIcon == icon ? Color.white : Color.clear, lineWidth: 3)
                                    )
                                    
                                    Text(icon.displayName)
                                        .font(.caption)
                                        .foregroundColor(selectedIcon == icon ? .white : .gray)
                                }
                                .padding(.vertical, 8)
                                .onTapGesture {
                                    selectedIcon = icon
                                    icon.apply()
                                }
                            }
                        }
                    }
                    
                    NavigationLink(destination: LanguagePickerView()) {
                        HStack {
                            Text("language")
                            Spacer()
                            Text(currentLanguageName).foregroundColor(.gray)
                        }
                    }
                }
                
                Section(header: Text("automation")) {
                    Toggle("auto_connect", isOn: $vm.autoConnect)
                    Toggle("boot_connect", isOn: $appSettings.connectOnDemand)
                }
                
                Section(header: Text("proxy"), footer: Text("proxy_info_desc")) {
                    proxyRow(proto: "SOCKS5", address: vm.socksAddress)
                    proxyRow(proto: "MTProto", address: vm.mtprotoAddress)
                }
                
                Section(header: Text("data")) {
                    Button(role: .destructive, action: { showResetAlert = true }) {
                        Text("reset_all")
                    }
                }
                
                Section {
                    HStack {
                        Spacer()
                        Text("Lionheart VPN v1.3").font(.footnote).foregroundColor(.gray)
                        Spacer()
                    }
                }
                .listRowBackground(Color.clear)
            }
            .navigationTitle("settings")
            .alert("reset_all_confirm", isPresented: $showResetAlert) {
                Button("cancel", role: .cancel) { }
                Button("reset", role: .destructive) { vm.clearAll() }
            } message: {
                Text("reset_all_warning")
            }
        }
    }
    
    private func proxyRow(proto: String, address: String) -> some View {
        HStack {
            Text(proto)
            Spacer()
            Text(address)
                .font(.system(.body, design: .monospaced))
                .foregroundColor(.gray)
            Button(action: {
                UIPasteboard.general.string = address
                vm.showSuccess(NSLocalizedString("copied", comment: ""))
            }) {
                Image(systemName: "doc.on.doc").foregroundColor(.appAccent)
            }
        }
    }
    
    private var currentLanguageName: String {
        let code = appSettings.languageCode
        return AppSettings.supportedLanguages.first { $0.code == code }?.nativeName ?? code
    }
}

struct LanguagePickerView: View {
    @EnvironmentObject var appSettings: AppSettings
    @Environment(\.dismiss) var dismiss
    
    var body: some View {
        List {
            // ИСПРАВЛЕНИЕ: Перебираем только языки из массива, без ручного дублирования "System"
            ForEach(AppSettings.supportedLanguages, id: \.code) { lang in
                Button(action: {
                    appSettings.languageCode = lang.code
                    dismiss()
                }) {
                    HStack {
                        Text(lang.nativeName)
                            .foregroundColor(.white)
                        
                        Spacer()
                        
                        if appSettings.languageCode == lang.code {
                            Image(systemName: "checkmark").foregroundColor(.blue)
                        }
                    }
                }
            }
        }
        .navigationTitle("language")
    }
}

import SwiftUI

struct AddServerWizard: View {
    @EnvironmentObject var vm: VPNManager
    @Environment(\.dismiss) var dismiss
    @State private var method: AddMethod?
    @State private var showQRScanner = false
    
    enum AddMethod: String, CaseIterable {
        case auto = "auto", smartKey = "smartKey", qr = "qr", importLink = "importLink"
        
        var title: LocalizedStringKey {
            switch self {
            case .auto: return "auto_setup"
            case .smartKey: return "enter_smart_key"
            case .qr: return "scan_qr"
            case .importLink: return "import_link"
            }
        }
        
        var icon: String {
            switch self {
            case .auto: return "bolt.fill"
            case .smartKey: return "key.fill"
            case .qr: return "qrcode.viewfinder"
            case .importLink: return "link"
            }
        }
        
        var desc: LocalizedStringKey {
            switch self {
            case .auto: return "auto_setup_desc"
            case .smartKey: return "smart_key_desc"
            case .qr: return "qr_scan_desc"
            case .importLink: return "import_link_desc"
            }
        }
        
        var color: Color {
            switch self {
            case .auto: return .green
            case .smartKey: return .appAccent
            case .qr: return .purple
            case .importLink: return .orange
            }
        }
    }
    
    var body: some View {
        NavigationView {
            ZStack {
                if let m = method {
                    destinationFor(m)
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 24) {
                            Text("add_server_how")
                                .font(.title2.bold())
                                .padding(.horizontal)
                                .padding(.top, 20)
                            
                            methodPicker
                            
                            Spacer()
                        }
                    }
                }
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    if method != nil {
                        Button("back") { method = nil }
                    } else {
                        Button("cancel") { dismiss() }
                    }
                }
            }
        }
        .fullScreenCover(isPresented: $showQRScanner) {
            QRScannerView { code in
                let cleaned = code.replacingOccurrences(of: "lionheart://import?key=", with: "")
                vm.importSmartKey(cleaned)
                dismiss()
            }
        }
    }
    
    private var methodPicker: some View {
        VStack(spacing: 12) {
            ForEach(AddMethod.allCases, id: \.self) { m in
                Button(action: {
                    if m == .qr {
                        showQRScanner = true
                    } else {
                        method = m
                    }
                }) {
                    HStack(spacing: 16) {
                        ZStack {
                            Circle()
                                .fill(m.color.opacity(0.2))
                                .frame(width: 44, height: 44)
                            Image(systemName: m.icon)
                                .foregroundColor(m.color)
                                .font(.title3)
                        }
                        
                        VStack(alignment: .leading, spacing: 2) {
                            Text(m.title)
                                .font(.headline)
                                .foregroundColor(.white)
                            
                            Text(m.desc)
                                .font(.subheadline)
                                .foregroundColor(.gray)
                                .multilineTextAlignment(.leading)
                        }
                        Spacer()
                        Image(systemName: "chevron.right")
                            .foregroundColor(.gray)
                    }
                    .padding()
                    .background(Color(UIColor.secondarySystemGroupedBackground))
                    .cornerRadius(16)
                }
            }
        }
        .padding(.horizontal)
    }
    
    @ViewBuilder
    private func destinationFor(_ m: AddMethod) -> some View {
        switch m {
        case .auto: AutoSetupFlow(dismiss: dismiss)
        case .smartKey: SmartKeyFlow(dismiss: dismiss)
        case .qr: EmptyView()
        case .importLink: ImportLinkFlow(dismiss: dismiss)
        }
    }
    
    private struct SmartKeyFlow: View {
        let dismiss: DismissAction
        @EnvironmentObject var vm: VPNManager
        @State private var step = 0
        @State private var serverName = ""
        @State private var smartKey = ""
        @State private var errorMsg = ""
        @State private var countryCode = ""
        @State private var showQR = false
        
        var body: some View {
            VStack {
                StepProgressView(totalSteps: 2, currentStep: step + 1).padding()
                if step == 0 { stepKey } else { stepDetails }
            }
            .navigationTitle(step == 0 ? "enter_smart_key" : "server_details")
        }
        
        private var stepKey: some View {
            VStack(alignment: .leading, spacing: 20) {
                Text("smart_key_hint").foregroundColor(.gray)
                TextField("eyJ... (Base64)", text: $smartKey)
                    .textFieldStyle(RoundedBorderTextFieldStyle())
                    .font(.system(.body, design: .monospaced))
                    .autocapitalization(.none)
                    .disableAutocorrection(true)
                
                if !errorMsg.isEmpty {
                    Text(errorMsg).foregroundColor(.red).font(.caption)
                }
                
                Button("next") {
                    let k = smartKey.trimmingCharacters(in: .whitespacesAndNewlines)
                    if k.isEmpty { errorMsg = NSLocalizedString("error_empty_key", comment: ""); return }
                    if decodeSmartKey(k) == nil { errorMsg = NSLocalizedString("error_invalid_key", comment: ""); return }
                    errorMsg = ""
                    withAnimation { step = 1 }
                }
                .buttonStyle(PremiumButtonStyle())
                .padding(.top)
                
                Spacer()
            }
            .padding()
        }
        
        private var stepDetails: some View {
            Form {
                Section(header: Text("server_details_hint")) {
                    TextField("server_name_hint", text: $serverName)
                    Picker("country", selection: $countryCode) {
                        Text("select_server").tag("")
                        ForEach(ServerProfile.countries, id: \.code) { c in
                            Text(LocalizedStringKey(c.key)).tag(c.code)
                        }
                    }
                }
                
                Section {
                    Button("add_server") { addServer() }
                        .frame(maxWidth: .infinity)
                        .foregroundColor(.green)
                }
            }
        }
        
        private func addServer() {
            guard let decoded = decodeSmartKey(smartKey) else { return }
            let parts = decoded.split(separator: "|", maxSplits: 1)
            guard parts.count == 2 else { return }
            let peer = String(parts[0])
            let host = peer.contains(":") ? String(peer.split(separator: ":")[0]) : peer
            
            var s = ServerProfile()
            s.name = serverName.isEmpty ? host : serverName
            s.serverIP = host
            s.smartKey = smartKey
            s.countryCode = countryCode
            
            vm.addServer(s)
            dismiss()
        }
    }
    
    private struct AutoSetupFlow: View {
        let dismiss: DismissAction
        @EnvironmentObject var vm: VPNManager
        @State private var step = 0
        @State private var serverName = ""
        @State private var countryCode = ""
        @State private var ipAddress = ""
        @State private var sshPort = "22"
        @State private var username = "root"
        @State private var password = ""
        @State private var isInstalling = false
        @State private var progressMsg = ""
        @State private var progressStep = 0
        @State private var installError: String?
        
        var body: some View {
            VStack {
                StepProgressView(totalSteps: 3, currentStep: step + 1).padding()
                if step == 0 { stepName }
                else if step == 1 { stepCreds }
                else { stepInstall }
            }
            .navigationTitle(step == 0 ? "server_details" : (step == 1 ? "server_credentials" : "auto_setup"))
        }
        
        private var stepName: some View {
            Form {
                Section(header: Text("server_details_hint")) {
                    TextField("server_name_hint", text: $serverName)
                    Picker("country", selection: $countryCode) {
                        Text("select_server").tag("")
                        ForEach(ServerProfile.countries, id: \.code) { c in
                            Text(LocalizedStringKey(c.key)).tag(c.code)
                        }
                    }
                }
                Section {
                    Button("next") { withAnimation { step = 1 } }
                        .frame(maxWidth: .infinity)
                        .foregroundColor(.appAccent)
                }
            }
        }
        
        private var stepCreds: some View {
            Form {
                Section(header: Text("vps_title"), footer: Text("step_auth_hint")) {
                    IPAddressField(label: "ip_address", text: $ipAddress)
                    TextField("port", text: $sshPort).keyboardType(.numberPad)
                    TextField("username", text: $username).autocapitalization(.none)
                    SecureField("ssh_password", text: $password)
                }
                Section {
                    Button("install") {
                        withAnimation { step = 2 }
                        startInstall()
                    }
                    .frame(maxWidth: .infinity)
                    .foregroundColor(.green)
                    .disabled(ipAddress.isEmpty || password.isEmpty)
                }
            }
        }
        
        private var stepInstall: some View {
            VStack(spacing: 30) {
                Spacer()
                if let err = installError {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 60)).foregroundColor(.orange)
                    
                    // Блок для отображения логов консоли сервера, если скрипт упал
                    ScrollView {
                        Text(err)
                            .font(.system(.footnote, design: .monospaced))
                            .multilineTextAlignment(.leading)
                            .padding()
                    }
                    .frame(maxHeight: 200)
                    .background(Color.black.opacity(0.1))
                    .cornerRadius(12)
                    
                    Button("try_again") {
                        installError = nil
                        progressStep = 0
                        startInstall()
                    }.buttonStyle(PremiumButtonStyle())
                    
                    Button("use_smart_key_instead") {
                        dismiss()
                    }.padding(.top)
                } else if progressStep == 5 {
                    Image(systemName: "checkmark.circle.fill")
                        .font(.system(size: 80)).foregroundColor(.green)
                    Text("server_installed").font(.title2.bold())
                    Button("done") { dismiss() }
                        .buttonStyle(PremiumButtonStyle())
                        .padding(.top)
                } else {
                    ProgressView()
                        .scaleEffect(2.0)
                        .padding(.bottom, 20)
                    Text(LocalizedStringKey(progressMsg))
                        .font(.headline)
                        .foregroundColor(.appAccent)
                        .contentTransition(.numericText())
                    Text("reassurance_msg")
                        .font(.caption)
                        .foregroundColor(.gray)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 40)
                }
                Spacer()
            }.padding()
        }
        
        private func startInstall() {
            isInstalling = true
            installError = nil
            progressStep = 1
            progressMsg = "ssh_step_connecting"
            
            Task {
                let result = await vm.installServerViaSSH(host: ipAddress, port: Int(sshPort) ?? 22, username: username, password: password, serverName: serverName, onProgress: { p in
                    self.progressStep = p.step
                    self.progressMsg = p.message
                })
                DispatchQueue.main.async {
                    switch result {
                    case .success(var s):
                        s.countryCode = countryCode
                        vm.addServer(s)
                        self.progressStep = 5
                        self.isInstalling = false
                    case .failure(let e):
                        self.installError = e.localizedDescription
                        self.isInstalling = false
                    }
                }
            }
        }
    }
    
    private struct ImportLinkFlow: View {
        let dismiss: DismissAction
        @EnvironmentObject var vm: VPNManager
        @State private var linkText = ""
        @State private var errorMsg = ""
        
        var body: some View {
            VStack(alignment: .leading, spacing: 20) {
                Text("import_link_hint").foregroundColor(.gray)
                TextField("lionheart://import?key=...", text: $linkText)
                    .textFieldStyle(RoundedBorderTextFieldStyle())
                    .autocapitalization(.none)
                    .disableAutocorrection(true)
                
                if !errorMsg.isEmpty {
                    Text(errorMsg).foregroundColor(.red).font(.caption)
                }
                
                Button("import_btn") { importFromLink() }
                    .buttonStyle(PremiumButtonStyle())
                    .padding(.top)
                
                Spacer()
            }
            .padding()
            .navigationTitle("import_link")
        }
        
        private func importFromLink() {
            let t = linkText.trimmingCharacters(in: .whitespacesAndNewlines)
            if t.isEmpty { return }
            
            if t.starts(with: "lionheart://import?key=") {
                let key = t.replacingOccurrences(of: "lionheart://import?key=", with: "")
                if decodeSmartKey(key) != nil {
                    vm.importSmartKey(key)
                    dismiss()
                } else {
                    errorMsg = NSLocalizedString("error_invalid_link", comment: "")
                }
            } else if decodeSmartKey(t) != nil {
                vm.importSmartKey(t)
                dismiss()
            } else {
                errorMsg = NSLocalizedString("error_invalid_link", comment: "")
            }
        }
    }
}

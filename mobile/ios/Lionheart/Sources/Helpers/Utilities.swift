import SwiftUI
import CoreImage.CIFilterBuiltins
import Network

func formatSpeed(_ bps: Int64) -> String {
    let kbps = Double(bps) / 1024
    if kbps > 1024 {
        return String(format: "%.1f MB/s", kbps / 1024)
    }
    return String(format: "%.0f KB/s", kbps)
}

func formatBytes(_ b: Int64) -> String {
    let kb = Double(b) / 1024
    let mb = kb / 1024
    if mb > 1024 { return String(format: "%.2f GB", mb / 1024) }
    if mb > 1 { return String(format: "%.1f MB", mb) }
    return String(format: "%.0f KB", kb)
}

func formatDuration(_ interval: TimeInterval) -> String {
    let d = Int(interval)
    let h = d / 3600
    let m = (d % 3600) / 60
    let s = d % 60
    if h > 0 { return String(format: "%d:%02d:%02d", h, m, s) }
    return String(format: "%02d:%02d", m, s)
}

func decodeSmartKey(_ key: String) -> String? {
    let u = key.replacingOccurrences(of: "-", with: "+").replacingOccurrences(of: "_", with: "/")
    let p = u + String(repeating: "=", count: (4 - u.count % 4) % 4)
    guard let d = Data(base64Encoded: p) else { return nil }
    return String(data: d, encoding: .utf8)
}

func validateIPv4(_ ip: String) -> Bool {
    return IPv4Address(ip) != nil
}

@ViewBuilder
func flagView(_ code: String) -> some View {
    if !code.isEmpty {
        // ИСПРАВЛЕНИЕ: Теперь ищет картинки с маленькой буквы (ru, us, tj)
        Image(code.lowercased())
            .resizable()
            .aspectRatio(contentMode: .fit)
            .frame(width: 24, height: 18)
            .cornerRadius(3)
    } else {
        Image(systemName: "globe")
            .foregroundColor(.gray)
            .frame(width: 24, height: 18)
    }
}

func generateQRCode(from string: String) -> UIImage {
    let ctx = CIContext()
    let f = CIFilter.qrCodeGenerator()
    f.message = Data(string.utf8)
    if let out = f.outputImage {
        let scaled = out.transformed(by: CGAffineTransform(scaleX: 10, y: 10))
        if let cg = ctx.createCGImage(scaled, from: scaled.extent) {
            return UIImage(cgImage: cg)
        }
    }
    return UIImage(systemName: "xmark.circle") ?? UIImage()
}

func glassCard(cornerRadius: CGFloat = 24) -> some View {
    RoundedRectangle(cornerRadius: cornerRadius)
        .fill(Color(UIColor.secondarySystemGroupedBackground).opacity(0.8))
        .background(Material.ultraThin)
        .clipShape(RoundedRectangle(cornerRadius: cornerRadius))
}

func glassButton(cornerRadius: CGFloat = 20) -> some View {
    RoundedRectangle(cornerRadius: cornerRadius)
        .fill(Color.appAccent.opacity(0.15))
        .background(Material.thin)
        .clipShape(RoundedRectangle(cornerRadius: cornerRadius))
}

struct PremiumButtonStyle: ButtonStyle {
    var colors: [Color] = [Color.appAccent, Color(red: 0.25, green: 0.55, blue: 0.85)]
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.headline)
            .foregroundColor(.white)
            .frame(maxWidth: .infinity)
            .padding()
            .background(LinearGradient(colors: colors, startPoint: .leading, endPoint: .trailing))
            .cornerRadius(16)
            .scaleEffect(configuration.isPressed ? 0.96 : 1.0)
            .animation(.easeOut(duration: 0.2), value: configuration.isPressed)
    }
}

struct SecondaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.headline)
            .foregroundColor(.appAccent)
            .frame(maxWidth: .infinity)
            .padding()
            .background(Color.appAccent.opacity(0.15))
            .cornerRadius(16)
            .scaleEffect(configuration.isPressed ? 0.96 : 1.0)
            .animation(.easeOut(duration: 0.2), value: configuration.isPressed)
    }
}

struct StatCard: View {
    let title: LocalizedStringKey
    let value: String
    let icon: String
    let gradientColors: [Color]
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: icon)
                    .foregroundStyle(LinearGradient(colors: gradientColors, startPoint: .topLeading, endPoint: .bottomTrailing))
                    .font(.title2)
                Spacer()
            }
            VStack(alignment: .leading, spacing: 4) {
                Text(value)
                    .font(.system(.title3, design: .rounded).weight(.semibold))
                    .contentTransition(.numericText())
                Text(title)
                    .font(.caption)
                    .foregroundColor(.gray)
            }
        }
        .padding()
        .background(glassCard(cornerRadius: 20))
    }
}

struct StepProgressView: View {
    let totalSteps: Int
    let currentStep: Int
    var body: some View {
        HStack {
            ForEach(1...totalSteps, id: \.self) { step in
                ZStack {
                    Circle()
                        .fill(step <= currentStep ? Color.appAccent : Color.gray.opacity(0.3))
                        .frame(width: 32, height: 32)
                    Text("\(step)")
                        .font(.caption.bold())
                        .foregroundColor(step <= currentStep ? .white : .gray)
                }
                if step < totalSteps {
                    Rectangle()
                        .fill(step < currentStep ? Color.appAccent : Color.gray.opacity(0.3))
                        .frame(height: 2)
                }
            }
        }
    }
}

struct IPAddressField: View {
    let label: LocalizedStringKey
    @Binding var text: String
    @State private var isValid = true
    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            TextField(label, text: $text)
                .keyboardType(.numbersAndPunctuation)
                .autocapitalization(.none)
                .disableAutocorrection(true)
                .onChange(of: text) { newValue in
                    let sanitized = newValue.replacingOccurrences(of: ",", with: ".")
                    if sanitized != newValue { text = sanitized }
                    isValid = text.isEmpty || validateIPv4(text)
                }
            if !isValid {
                Text("Invalid IPv4 address")
                    .font(.caption)
                    .foregroundColor(.red)
            }
        }
    }
}

struct HelpableRow: View {
    let title: LocalizedStringKey
    let helpTitle: String
    let helpBody: String
    @State private var showHelp = false
    var body: some View {
        HStack {
            Text(title)
            Spacer()
            Button(action: { showHelp = true }) {
                Image(systemName: "questionmark.circle").foregroundColor(.appAccent)
            }
        }
        .alert(LocalizedStringKey(helpTitle), isPresented: $showHelp) {
            Button("ok", role: .cancel) { }
        } message: {
            Text(LocalizedStringKey(helpBody))
        }
    }
}

struct ToastModifier: ViewModifier {
    @Binding var isShowing: Bool
    let message: String
    let isError: Bool
    func body(content: Content) -> some View {
        ZStack {
            content
            if isShowing {
                VStack {
                    Spacer()
                    HStack {
                        Image(systemName: isError ? "exclamationmark.triangle.fill" : "checkmark.circle.fill")
                        Text(LocalizedStringKey(message))
                    }
                    .padding()
                    .background(isError ? Color.red : Color.green)
                    .foregroundColor(.white)
                    .cornerRadius(25)
                    .shadow(radius: 10)
                    .padding(.bottom, 30)
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                    .onAppear {
                        DispatchQueue.main.asyncAfter(deadline: .now() + 3) {
                            withAnimation { isShowing = false }
                        }
                    }
                }
                .zIndex(1)
            }
        }
    }
}

extension View {
    func toast(isShowing: Binding<Bool>, message: String, isError: Bool = false) -> some View {
        self.modifier(ToastModifier(isShowing: isShowing, message: message, isError: isError))
    }
}

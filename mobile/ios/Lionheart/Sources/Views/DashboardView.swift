import SwiftUI

struct DashboardView: View {
    @EnvironmentObject var vm: VPNManager
    @State private var glowActive = false
    
    var body: some View {
        // Мы используем ZStack, чтобы кнопка гарантированно была в математическом центре экрана
        ZStack {
            // Кнопка подключения (всегда по центру экрана)
            heroConnectButton
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)
            
            VStack {
                // Убрали верхний HStack с Lionheart и Info
                // Убрали Spacer() над подписью tagline
                // Убрали Text("tagline")
                
                Spacer() // Толкает статистику вниз
                
                // Статистика отображается только если есть подключение
                if vm.status == .connected && vm.activeServer != nil {
                    statsGrid
                        .padding(.bottom, 20)
                } else {
                    // Убрали Text с подсказкой про вкладку Серверы
                    // Просто оставляем Spacer чтобы кнопка была по центру.
                    Color.clear
                        .frame(height: 1)
                        .padding(.bottom, 20)
                }
            }
            .padding(.top, 10) // Небольшой отступ сверху для ZStack
        }
        .onAppear {
            withAnimation(.easeInOut(duration: 2.0).repeatForever(autoreverses: true)) {
                glowActive = true
            }
        }
    }
    
    private var heroConnectButton: some View {
        Button(action: {
            #if !targetEnvironment(simulator)
            UIImpactFeedbackGenerator(style: .medium).impactOccurred()
            #endif
            vm.toggleConnection()
        }) {
            ZStack {
                Circle()
                    .fill(LinearGradient(colors: buttonGradient, startPoint: .topLeading, endPoint: .bottomTrailing))
                    .frame(width: 170, height: 170)
                    .shadow(color: buttonGradient[0].opacity(glowActive ? 0.6 : 0.2), radius: glowActive ? 30 : 10)
                
                VStack(spacing: 12) {
                    Image(systemName: vm.status.iconName)
                        .font(.system(size: 40, weight: .semibold))
                        .foregroundColor(.white)
                        .symbolEffect(.bounce, value: vm.status)
                    
                    // Всегда показываем "Подключиться" для disconnected, чтобы было минималистично
                    Text(vm.status == .disconnected ? "Подключиться" : vm.status.displayName)
                        .font(.headline)
                        .fontWeight(.semibold)
                        .foregroundColor(.white)
                }
            }
        }
        .buttonStyle(PlainButtonStyle())
        .disabled(vm.activeServer == nil)
        .opacity(vm.activeServer == nil ? 0.5 : 1.0)
    }
    
    private var buttonGradient: [Color] {
        switch vm.status {
        case .connected: return [.green, .mint]
        case .connecting, .reconnecting: return [.orange, .yellow]
        case .error: return [.red, .pink]
        case .disconnected: return [Color.appAccent, .blue]
        }
    }
    
    private var statsGrid: some View {
        HStack(spacing: 20) {
            StatCard(title: "download", value: formatSpeed(vm.rxSpeed), icon: "arrow.down.circle.fill", gradientColors: [.blue, .cyan])
            StatCard(title: "upload", value: formatSpeed(vm.txSpeed), icon: "arrow.up.circle.fill", gradientColors: [.purple, .pink])
        }
        .padding(.horizontal)
    }
}

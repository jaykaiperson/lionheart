import SwiftUI

// MARK: - LogsView

struct LogsView: View {
    @EnvironmentObject var vm: VPNManager
    @State private var searchText = ""

    private var filteredLogs: [LogEntry] {
        let r = Array(vm.logs.reversed())
        if searchText.isEmpty { return r }
        return r.filter { $0.message.localizedCaseInsensitiveContains(searchText) || $0.level.localizedCaseInsensitiveContains(searchText) }
    }

    var body: some View {
        Group {
            if vm.logs.isEmpty {
                ContentUnavailableView { Label("logs", systemImage: "doc.text") } description: { Text("logs_hint") }
            } else {
                List {
                    ForEach(filteredLogs) { entry in
                        VStack(alignment: .leading, spacing: 4) {
                            HStack(spacing: 6) {
                                Circle().fill(colorFor(entry.level)).frame(width: 7, height: 7)
                                Text(entry.level.uppercased()).font(.caption2.bold()).foregroundStyle(colorFor(entry.level))
                                Spacer()
                                Text(entry.timestamp, style: .time).font(.caption2).foregroundStyle(.tertiary)
                            }
                            Text(entry.message).font(.caption).fontDesign(.monospaced).foregroundStyle(.secondary).lineLimit(3)
                        }.listRowSeparator(.hidden)
                    }
                }.listStyle(.plain)
            }
        }
        .searchable(text: $searchText, prompt: "Filter logs")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                if !vm.logs.isEmpty {
                    Menu {
                        Button { UIPasteboard.general.string = vm.logs.map { "[\($0.timestamp.formatted(date: .omitted, time: .standard))] \($0.level): \($0.message)" }.joined(separator: "\n"); vm.showSuccess("Copied") } label: { Label("Copy All", systemImage: "doc.on.doc") }
                        Button(role: .destructive) { vm.clearLogs() } label: { Label("Clear Logs", systemImage: "trash") }
                    } label: { Image(systemName: "ellipsis.circle") }
                }
            }
        }
    }

    private func colorFor(_ level: String) -> Color {
        switch level.uppercased() { case "ERROR": return .red; case "WARN": return .yellow; case "INFO": return .green; default: return .secondary }
    }
}

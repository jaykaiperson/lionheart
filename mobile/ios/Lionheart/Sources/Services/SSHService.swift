import Foundation
import Golib // Проверь, чтобы название совпадало с твоим фреймворком

actor SSHService {
    
    func installServer(
        host: String,
        port: Int,
        username: String,
        password: String,
        serverName: String,
        onProgress: @escaping (String, Int) -> Void
    ) async throws -> (smartKey: String, version: String) {
        
        let cleanHost = host.trimmingCharacters(in: .whitespacesAndNewlines)
        let cleanUser = username.trimmingCharacters(in: .whitespacesAndNewlines)
        let cleanPass = password.trimmingCharacters(in: .whitespacesAndNewlines)
        
        let progressTask = Task {
            try? await Task.sleep(nanoseconds: 2_000_000_000)
            if !Task.isCancelled { onProgress("ssh_step_connecting", 1) }
            try? await Task.sleep(nanoseconds: 4_000_000_000)
            if !Task.isCancelled { onProgress("ssh_step_installing", 3) }
            try? await Task.sleep(nanoseconds: 6_000_000_000)
            if !Task.isCancelled { onProgress("ssh_step_reading_key", 4) }
        }
        
        // Выносим тяжелый вызов Go в фоновый поток
        let result = try await Task.detached(priority: .userInitiated) {
            // Создаем переменную для перехвата ошибки от Go
            var error: NSError?
            
            // Функция ВОЗВРАЩАЕТ строку, а ошибку пишет в 5-й параметр
            let output = GolibInstallServer(cleanHost, port, cleanUser, cleanPass, &error)
            
            // Если Go вернул ошибку, пробрасываем её в Swift
            if let error = error {
                throw error
            }
            
            return output ?? ""
        }.value
        
        progressTask.cancel()
        onProgress("ssh_step_done", 5)
        
        // Парсим результат от Go (ключ|||версия)
        let parts = result.components(separatedBy: "|||")
        if parts.count == 2 {
            return (parts[0], parts[1])
        } else {
            throw SSHSetupError.noKeyFound
        }
    }
    
    func updateServer(
        host: String,
        port: Int,
        username: String,
        password: String,
        onProgress: @escaping (String, Int) -> Void
    ) async throws -> String {
        
        let cleanHost = host.trimmingCharacters(in: .whitespacesAndNewlines)
        let cleanUser = username.trimmingCharacters(in: .whitespacesAndNewlines)
        let cleanPass = password.trimmingCharacters(in: .whitespacesAndNewlines)
        
        let progressTask = Task {
            try? await Task.sleep(nanoseconds: 2_000_000_000)
            if !Task.isCancelled { onProgress("ssh_step_connecting", 1) }
            try? await Task.sleep(nanoseconds: 4_000_000_000)
            if !Task.isCancelled { onProgress("ssh_step_updating", 2) }
        }
        
        let version = try await Task.detached(priority: .userInitiated) {
            var error: NSError?
            
            let output = GolibUpdateServer(cleanHost, port, cleanUser, cleanPass, &error)
            
            if let error = error {
                throw error
            }
            
            return output ?? ""
        }.value
        
        progressTask.cancel()
        onProgress("ssh_step_done", 3)
        
        return version
    }
}

enum SSHSetupError: LocalizedError {
    case noKeyFound
    var errorDescription: String? {
        switch self {
        case .noKeyFound:
            return NSLocalizedString("ssh_error_no_key", comment: "")
        }
    }
}

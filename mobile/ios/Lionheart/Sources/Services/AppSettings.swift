import SwiftUI
import Combine

final class AppSettings: ObservableObject {
    @AppStorage("selectedLanguage") var languageCode: String = "system" {
        didSet { objectWillChange.send() }
    }

    var locale: Locale {
        languageCode == "system" ? .current : Locale(identifier: languageCode)
    }

    /// flagCode: ISO 3166 alpha-2 lowercase for flag asset lookup
    static let supportedLanguages: [(code: String, name: String, nativeName: String, flagCode: String)] = [
        ("system", "System",     "System",       ""),
        ("en",     "English",    "English",      "gb"),
        ("ru",     "Russian",    "Русский",      "ru"),
        ("be",     "Belarusian", "Беларуская",   "by"),
        ("tt",     "Tatar",      "Татарча",      "tt"),
    ]

    @AppStorage("hasCompletedOnboarding") var hasCompletedOnboarding: Bool = false
    @AppStorage("connectOnDemand") var connectOnDemand: Bool = false
    @AppStorage("showNotifications") var showNotifications: Bool = true
}

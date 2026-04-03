package com.lionheart.vpn.data

import android.content.ComponentName
import android.content.Context
import android.content.pm.PackageManager

data class DisguiseOption(
    val id: String,
    val label: String,
    val aliasName: String,
    val category: String // "color" or "disguise"
)

object AppDisguise {
    // Color variants of the lion icon
    val COLOR_OPTIONS = listOf(
        DisguiseOption("gold",  "Золотой",  ".AliasLionGold",  "color"),
        DisguiseOption("red",   "Красный",  ".AliasLionRed",   "color"),
        DisguiseOption("blue",  "Синий",    ".AliasLionBlue",  "color"),
        DisguiseOption("green", "Зелёный",  ".AliasLionGreen", "color"),
        DisguiseOption("white", "Белый",    ".AliasLionWhite", "color"),
    )

    // Disguise icons (hide the app)
    val DISGUISE_OPTIONS = listOf(
        DisguiseOption("calculator", "Calculator", ".AliasCalculator", "disguise"),
        DisguiseOption("clock",      "Clock",      ".AliasClock",      "disguise"),
        DisguiseOption("music",      "Music",      ".AliasMusic",      "disguise"),
        DisguiseOption("notes",      "Notes",      ".AliasNotes",      "disguise"),
        DisguiseOption("weather",    "Weather",    ".AliasWeather",    "disguise"),
    )

    // Default uses MainActivity directly
    val DEFAULT = DisguiseOption("default", "Lionheart", ".MainActivity", "color")

    val ALL_OPTIONS = listOf(DEFAULT) + COLOR_OPTIONS + DISGUISE_OPTIONS

    fun apply(context: Context, optionId: String) {
        val pm = context.packageManager
        val pkg = context.packageName

        ALL_OPTIONS.forEach { option ->
            val componentName = ComponentName(pkg, "$pkg${option.aliasName}")
            val newState = if (option.id == optionId) {
                PackageManager.COMPONENT_ENABLED_STATE_ENABLED
            } else {
                PackageManager.COMPONENT_ENABLED_STATE_DISABLED
            }
            try {
                pm.setComponentEnabledSetting(
                    componentName, newState, PackageManager.DONT_KILL_APP
                )
            } catch (_: Exception) {}
        }
    }

    fun getCurrent(context: Context): String {
        val pm = context.packageManager
        val pkg = context.packageName

        ALL_OPTIONS.forEach { option ->
            val componentName = ComponentName(pkg, "$pkg${option.aliasName}")
            try {
                val state = pm.getComponentEnabledSetting(componentName)
                if (state == PackageManager.COMPONENT_ENABLED_STATE_ENABLED) {
                    return option.id
                }
                if (state == PackageManager.COMPONENT_ENABLED_STATE_DEFAULT && option.id == "default") {
                    return "default"
                }
            } catch (_: Exception) {}
        }
        return "default"
    }
}

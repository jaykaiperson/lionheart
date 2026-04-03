# Lionheart VPN ProGuard Rules

# Go library
-keep class golib.** { *; }

# JSch (SSH)
-keep class com.jcraft.jsch.** { *; }
-dontwarn com.jcraft.jsch.**

# Gson
-keepattributes Signature
-keepattributes *Annotation*
-keep class com.google.gson.** { *; }
-keep class com.lionheart.vpn.data.ServerProfile { *; }
-keep class com.lionheart.vpn.data.SetupProgress { *; }

# Kotlin serialization
-keepclassmembers class * {
    @com.google.gson.annotations.SerializedName <fields>;
}

# AppCompat locale
-keep class androidx.appcompat.app.AppCompatDelegate { *; }

# Strip verbose logging in release
-assumenosideeffects class android.util.Log {
    public static int v(...);
    public static int d(...);
}
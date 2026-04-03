package com.lionheart.vpn.data
object AdBlockConfig {
    const val ADGUARD_DNS = "94.140.14.14"
    const val ADGUARD_DNS_V6 = "2a10:50c0::ad1:ff"
    const val CLOUDFLARE_SAFE_DNS = "1.1.1.2"
    fun getDns(adBlockEnabled: Boolean, userDns: String): String {
        return if (adBlockEnabled) ADGUARD_DNS else userDns
    }
}
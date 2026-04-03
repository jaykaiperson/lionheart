package com.lionheart.vpn.data
import java.time.LocalTime
import java.time.format.DateTimeFormatter
data class LogEntry(
    val time: String = LocalTime.now().format(DateTimeFormatter.ofPattern("HH:mm:ss")),
    val level: String,
    val message: String
)
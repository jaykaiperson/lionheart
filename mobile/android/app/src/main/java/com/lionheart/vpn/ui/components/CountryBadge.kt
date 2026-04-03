package com.lionheart.vpn.ui.components
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lionheart.vpn.R
@Composable
fun CountryBadge(
    countryCode: String,
    modifier: Modifier = Modifier,
    large: Boolean = false
) {
    val size = if (large) 56.dp else 36.dp
    val fontSize = if (large) 18.sp else 12.sp
    val code = countryCode.uppercase().take(2)
    Box(
        modifier = modifier
            .size(size)
            .clip(RoundedCornerShape(if (large) 16.dp else 10.dp))
            .background(MaterialTheme.colorScheme.primaryContainer),
        contentAlignment = Alignment.Center
    ) {
        Text(
            text = code,
            style = MaterialTheme.typography.labelLarge.copy(
                fontSize = fontSize,
                fontWeight = FontWeight.Bold,
                letterSpacing = 1.sp
            ),
            color = MaterialTheme.colorScheme.onPrimaryContainer
        )
    }
}
@Composable
fun countryCodeToName(code: String): String = when (code.uppercase()) {
    "US" -> stringResource(R.string.country_US)
    "DE" -> stringResource(R.string.country_DE)
    "NL" -> stringResource(R.string.country_NL)
    "FI" -> stringResource(R.string.country_FI)
    "SE" -> stringResource(R.string.country_SE)
    "GB" -> stringResource(R.string.country_GB)
    "FR" -> stringResource(R.string.country_FR)
    "JP" -> stringResource(R.string.country_JP)
    "SG" -> stringResource(R.string.country_SG)
    "CA" -> stringResource(R.string.country_CA)
    "AU" -> stringResource(R.string.country_AU)
    "RU" -> stringResource(R.string.country_RU)
    "KZ" -> stringResource(R.string.country_KZ)
    "LT" -> stringResource(R.string.country_LT)
    "LV" -> stringResource(R.string.country_LV)
    "EE" -> stringResource(R.string.country_EE)
    "PL" -> stringResource(R.string.country_PL)
    "UA" -> stringResource(R.string.country_UA)
    "TR" -> stringResource(R.string.country_TR)
    "IN" -> stringResource(R.string.country_IN)
    "BR" -> stringResource(R.string.country_BR)
    "HK" -> stringResource(R.string.country_HK)
    "BY" -> stringResource(R.string.country_BY)
    else -> code.uppercase()
}
package com.uisp.noc.wear.ui.theme

import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.wear.compose.material.Colors
import androidx.wear.compose.material.MaterialTheme
import androidx.wear.compose.material.Shapes

private val WearColorPalette = Colors(
    primary = PurplePrimary,
    primaryVariant = PurpleSecondary,
    secondary = PurpleSecondary,
    secondaryVariant = PurpleSecondary,
    surface = DarkSurface,
    background = DarkBackground,
    onPrimary = Color.Black,
    onSecondary = Color.White,
    onSurface = Color.White,
    onBackground = Color.White
)

@Composable
fun UISPNOCWearTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colors = WearColorPalette,
        typography = WearTypography,
        shapes = Shapes(),
        content = content
    )
}

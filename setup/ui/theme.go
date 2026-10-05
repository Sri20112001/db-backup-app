package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// VaultGuardTheme implements fyne.Theme matching the web design tokens:
// warm paper light mode, Cyberdeck Night dark mode (follows the OS/system
// theme variant automatically).
type VaultGuardTheme struct{}

var _ fyne.Theme = (*VaultGuardTheme)(nil)

// hexColor converts 6 or 8-character hex strings into color.RGBA
func hexColor(hex string) color.RGBA {
	var r, g, b, a uint8 = 0, 0, 0, 255
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) == 6 {
		fmtScanHex(hex, &r, &g, &b)
	} else if len(hex) == 8 {
		fmtScanHex8(hex, &r, &g, &b, &a)
	}
	return color.RGBA{R: r, G: g, B: b, A: a}
}

func fmtScanHex(s string, r, g, b *uint8) {
	var val uint32
	for i := 0; i < len(s); i++ {
		val = (val << 4) | fromHex(s[i])
	}
	*r = uint8((val >> 16) & 0xFF)
	*g = uint8((val >> 8) & 0xFF)
	*b = uint8(val & 0xFF)
}

func fmtScanHex8(s string, r, g, b, a *uint8) {
	var val uint32
	for i := 0; i < len(s); i++ {
		val = (val << 4) | fromHex(s[i])
	}
	*r = uint8((val >> 24) & 0xFF)
	*g = uint8((val >> 16) & 0xFF)
	*b = uint8((val >> 8) & 0xFF)
	*a = uint8(val & 0xFF)
}

func fromHex(c byte) uint32 {
	switch {
	case '0' <= c && c <= '9':
		return uint32(c - '0')
	case 'a' <= c && c <= 'f':
		return uint32(c - 'a' + 10)
	case 'A' <= c && c <= 'F':
		return uint32(c - 'A' + 10)
	}
	return 0
}

func (t *VaultGuardTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	isDark := variant == theme.VariantDark

	switch name {
	// Surface / Canvas Backgrounds
	case theme.ColorNameBackground:
		if isDark {
			return hexColor("#121316") // --color-surface / --color-background
		}
		return hexColor("#faf7f0")

	// Panels, Cards, Dialog overlays
	case theme.ColorNameOverlayBackground, theme.ColorNameMenuBackground:
		if isDark {
			return hexColor("#1b1e22") // --color-surface-container
		}
		return hexColor("#f5f0e6") // --color-surface-container-low

	// Primary brand accent (Orange / Amber)
	case theme.ColorNamePrimary:
		if isDark {
			return hexColor("#f59e0b") // Amber 500
		}
		return hexColor("#ea580c") // Orange 600

	// Text Colors
	case theme.ColorNameForeground:
		if isDark {
			return hexColor("#ece7df") // --color-on-surface
		}
		return hexColor("#1c1917") // --color-on-surface

	case theme.ColorNameForegroundOnPrimary:
		if isDark {
			return hexColor("#251a10") // --color-on-primary (Dark)
		}
		return hexColor("#ffffff") // --color-on-primary (Light)

	// Muted text / Labels
	case theme.ColorNamePlaceHolder:
		if isDark {
			return hexColor("#a39c90") // --color-on-surface-variant
		}
		return hexColor("#78716c") // --color-on-surface-variant

	// Borders & Dividers
	case theme.ColorNameSeparator:
		if isDark {
			return hexColor("#2c3036") // --color-outline-variant
		}
		return hexColor("#e7e0d3") // --color-outline-variant

	case theme.ColorNameShadow:
		return color.RGBA{0, 0, 0, 40}

	// Inputs & Form Controls
	case theme.ColorNameInputBackground:
		if isDark {
			return hexColor("#1b1e22") // --color-surface-container
		}
		return hexColor("#ffffff") // --color-surface-bright

	case theme.ColorNameInputBorder:
		if isDark {
			return hexColor("#78716c") // --color-outline
		}
		return hexColor("#a8a29e") // --color-outline

	// Buttons
	case theme.ColorNameButton:
		if isDark {
			return hexColor("#24272c") // --color-surface-container-high
		}
		return hexColor("#f0ebd9") // --color-surface-container

	case theme.ColorNameHover:
		if isDark {
			return hexColor("#2c3036") // --color-surface-container-highest
		}
		return hexColor("#e7e0d3") // --color-surface-container-high

	case theme.ColorNamePressed:
		if isDark {
			return hexColor("#78350f") // --color-primary-container
		}
		return hexColor("#ffedd5") // --color-primary-container

	// Semantic Status Colors
	case theme.ColorNameError:
		if isDark {
			return hexColor("#f87171") // --color-error
		}
		return hexColor("#dc2626")

	case theme.ColorNameSuccess:
		if isDark {
			return hexColor("#4ade80") // --color-success
		}
		return hexColor("#16a34a")

	case theme.ColorNameWarning:
		if isDark {
			return hexColor("#fbbf24") // --color-warning
		}
		return hexColor("#d97706")

	// Disabled Elements
	case theme.ColorNameDisabled:
		if isDark {
			return hexColor("#24272c")
		}
		return hexColor("#e7e0d3")

	case theme.ColorNameDisabledButton:
		if isDark {
			return hexColor("#1b1e22")
		}
		return hexColor("#f0ebd9")
	}

	return theme.DefaultTheme().Color(name, variant)
}

func (t *VaultGuardTheme) Font(style fyne.TextStyle) fyne.Resource {
	// If you bundle "Space Grotesk" or "JetBrains Mono" as static resources via fyne bundle:
	// if style.Monospace { return resourceJetBrainsMonoTtf }
	// return resourceSpaceGroteskTtf
	return theme.DefaultTheme().Font(style)
}

func (t *VaultGuardTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *VaultGuardTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	// NOTE: Fyne v2.5 exposes no corner-radius token (default widget
	// rounding applies); the 0.25rem/4px intent is documented, not enforced.
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInlineIcon:
		return 18
	case theme.SizeNameText:
		return 14 // Matches your 14px base font
	case theme.SizeNameHeadingText:
		return 18
	case theme.SizeNameSubHeadingText:
		return 15
	case theme.SizeNameCaptionText:
		return 11
	default:
		return theme.DefaultTheme().Size(name)
	}
}

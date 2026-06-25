package ui

import (
	"image/color"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type appTheme struct {
	base       fyne.Theme
	regular    fyne.Resource
	bold       fyne.Resource
	italic     fyne.Resource
	boldItalic fyne.Resource
}

func newAppTheme() fyne.Theme {
	fontDirectory := filepath.Join(os.Getenv("WINDIR"), "Fonts")
	return &appTheme{
		base:       theme.DefaultTheme(),
		regular:    loadFont(filepath.Join(fontDirectory, "segoeui.ttf")),
		bold:       loadFont(filepath.Join(fontDirectory, "segoeuib.ttf")),
		italic:     loadFont(filepath.Join(fontDirectory, "segoeuii.ttf")),
		boldItalic: loadFont(filepath.Join(fontDirectory, "segoeuiz.ttf")),
	}
}

func (t *appTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 245, G: 247, B: 250, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 30, G: 41, B: 59, A: 255}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 31, G: 78, B: 120, A: 255}
	case theme.ColorNameInputBackground, theme.ColorNameMenuBackground:
		return color.White
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return color.NRGBA{R: 203, G: 213, B: 225, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 100, G: 116, B: 139, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 148, G: 163, B: 184, A: 255}
	}
	return t.base.Color(name, theme.VariantLight)
}

func (t *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Monospace || style.Symbol {
		return t.base.Font(style)
	}
	switch {
	case style.Bold && style.Italic && t.boldItalic != nil:
		return t.boldItalic
	case style.Bold && t.bold != nil:
		return t.bold
	case style.Italic && t.italic != nil:
		return t.italic
	case t.regular != nil:
		return t.regular
	default:
		return t.base.Font(style)
	}
}

func (t *appTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *appTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 24
	case theme.SizeNameSubHeadingText:
		return 18
	case theme.SizeNameCaptionText:
		return 12
	}
	return t.base.Size(name)
}

func loadFont(path string) fyne.Resource {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource(filepath.Base(path), data)
}

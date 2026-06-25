package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type DropCard struct {
	widget.BaseWidget

	title    string
	accent   color.Color
	minSize  fyne.Size
	onTapped func()
	hovered  bool
	enabled  bool
}

func NewDropCard(title string, accent color.Color, minSize fyne.Size, onTapped func()) *DropCard {
	card := &DropCard{
		title:    title,
		accent:   accent,
		minSize:  minSize,
		onTapped: onTapped,
		enabled:  true,
	}
	card.ExtendBaseWidget(card)
	return card
}

func (c *DropCard) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(color.White)
	background.StrokeColor = c.accent
	background.StrokeWidth = 2

	titleOne := canvas.NewText("", c.accent)
	titleOne.Alignment = fyne.TextAlignCenter
	titleOne.TextSize = 19
	titleOne.TextStyle = fyne.TextStyle{Bold: true}

	titleTwo := canvas.NewText("", c.accent)
	titleTwo.Alignment = fyne.TextAlignCenter
	titleTwo.TextSize = 19
	titleTwo.TextStyle = fyne.TextStyle{Bold: true}

	button := canvas.NewRectangle(c.accent)
	buttonText := canvas.NewText("Choisir un CSV", color.White)
	buttonText.Alignment = fyne.TextAlignCenter
	buttonText.TextSize = 14

	renderer := &dropCardRenderer{
		card:       c,
		background: background,
		titleOne:   titleOne,
		titleTwo:   titleTwo,
		button:     button,
		buttonText: buttonText,
		objects:    []fyne.CanvasObject{background, titleOne, titleTwo, button, buttonText},
	}
	renderer.Refresh()
	return renderer
}

func (c *DropCard) MinSize() fyne.Size {
	return c.minSize
}

func (c *DropCard) SetTitle(title string) {
	c.title = title
	c.Refresh()
}

func (c *DropCard) SetEnabled(enabled bool) {
	c.enabled = enabled
	c.Refresh()
}

func (c *DropCard) Tapped(_ *fyne.PointEvent) {
	if c.enabled && c.onTapped != nil {
		c.onTapped()
	}
}

func (c *DropCard) Cursor() desktop.Cursor {
	if !c.enabled {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

func (c *DropCard) MouseIn(_ *desktop.MouseEvent) {
	c.hovered = true
	c.Refresh()
}

func (c *DropCard) MouseMoved(_ *desktop.MouseEvent) {}

func (c *DropCard) MouseOut() {
	c.hovered = false
	c.Refresh()
}

type dropCardRenderer struct {
	card       *DropCard
	background *canvas.Rectangle
	titleOne   *canvas.Text
	titleTwo   *canvas.Text
	button     *canvas.Rectangle
	buttonText *canvas.Text
	objects    []fyne.CanvasObject
}

func (r *dropCardRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	r.background.Move(fyne.NewPos(0, 0))

	groupHeight := float32(94)
	start := max32(8, (size.Height-groupHeight)/2)
	lineHeight := float32(23)
	r.titleOne.Resize(fyne.NewSize(size.Width-32, lineHeight))
	r.titleOne.Move(fyne.NewPos(16, start+2))
	r.titleTwo.Resize(fyne.NewSize(size.Width-32, lineHeight))
	r.titleTwo.Move(fyne.NewPos(16, start+25))

	buttonSize := fyne.NewSize(150, 36)
	buttonPosition := fyne.NewPos((size.Width-buttonSize.Width)/2, start+58)
	r.button.Resize(buttonSize)
	r.button.Move(buttonPosition)
	r.buttonText.Resize(buttonSize)
	r.buttonText.Move(buttonPosition)
}

func (r *dropCardRenderer) MinSize() fyne.Size {
	return r.card.minSize
}

func (r *dropCardRenderer) Refresh() {
	lines := strings.SplitN(r.card.title, "\n", 2)
	r.titleOne.Text = lines[0]
	r.titleTwo.Text = ""
	if len(lines) == 2 {
		r.titleTwo.Text = lines[1]
	}

	background := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	if r.card.hovered && r.card.enabled {
		background = color.NRGBA{R: 248, G: 250, B: 252, A: 255}
	}
	if !r.card.enabled {
		background = color.NRGBA{R: 245, G: 247, B: 250, A: 255}
	}
	r.background.FillColor = background
	r.background.StrokeColor = r.card.accent
	r.titleOne.Color = r.card.accent
	r.titleTwo.Color = r.card.accent
	r.button.FillColor = r.card.accent

	canvas.Refresh(r.background)
	canvas.Refresh(r.titleOne)
	canvas.Refresh(r.titleTwo)
	canvas.Refresh(r.button)
	canvas.Refresh(r.buttonText)
}

func (r *dropCardRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *dropCardRenderer) Destroy() {}

func max32(left, right float32) float32 {
	if left > right {
		return left
	}
	return right
}

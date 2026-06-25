package ui

import "fyne.io/fyne/v2"

type mainLayout struct{}

func (mainLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(820, 700)
}

func (mainLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 6 {
		return
	}

	const padding = float32(28)
	const headerHeight = float32(72)
	const optionsHeight = float32(64)
	const noteHeight = float32(34)

	contentWidth := size.Width - padding*2
	contentHeight := size.Height - padding*2
	flexible := contentHeight - headerHeight - optionsHeight - noteHeight
	topHeight := flexible * 0.42
	bottomHeight := flexible * 0.31
	statusHeight := flexible - topHeight - bottomHeight

	y := padding
	place(objects[0], padding, y, contentWidth, headerHeight)
	y += headerHeight
	place(objects[1], padding, y, contentWidth, optionsHeight)
	y += optionsHeight
	place(objects[2], padding, y, contentWidth, topHeight)
	y += topHeight
	place(objects[3], padding, y, contentWidth, bottomHeight)
	y += bottomHeight
	place(objects[4], padding, y, contentWidth, noteHeight)
	y += noteHeight
	place(objects[5], padding, y, contentWidth, statusHeight)
}

type topCardsLayout struct{}

func (topCardsLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(760, 180)
	}
	left := objects[0].MinSize()
	right := objects[1].MinSize()
	return fyne.NewSize(left.Width+right.Width+16, max32(left.Height, right.Height)+16)
}

func (topCardsLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	const gap = float32(16)
	const verticalPadding = float32(8)
	width := (size.Width - gap) / 2
	height := size.Height - verticalPadding*2
	place(objects[0], 0, verticalPadding, width, height)
	place(objects[1], width+gap, verticalPadding, width, height)
}

type bottomCardLayout struct{}

func (bottomCardLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(460, 120)
	}
	minimum := objects[0].MinSize()
	return fyne.NewSize(minimum.Width, minimum.Height+16)
}

func (bottomCardLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	width := size.Width * 0.60
	if width < objects[0].MinSize().Width {
		width = objects[0].MinSize().Width
	}
	if width > size.Width {
		width = size.Width
	}
	height := size.Height - 16
	place(objects[0], (size.Width-width)/2, 8, width, height)
}

func place(object fyne.CanvasObject, x, y, width, height float32) {
	object.Move(fyne.NewPos(x, y))
	object.Resize(fyne.NewSize(width, height))
}

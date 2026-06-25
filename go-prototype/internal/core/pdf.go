package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	marotoCore "github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type PDFExporter struct{}

func NewPDFExporter() *PDFExporter {
	return &PDFExporter{}
}

func (e *PDFExporter) Save(dataset ExportDataset, outputPath string) error {
	fontFamily, customFonts := pdfFonts()
	fontSize := 7.6
	if len(dataset.Columns) > 6 {
		fontSize = 7.2
	}

	builder := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithOrientation(orientation.Vertical).
		WithLeftMargin(8).
		WithTopMargin(8).
		WithRightMargin(8).
		WithBottomMargin(14).
		WithMaxGridSize(100).
		WithDefaultFont(&props.Font{
			Family: fontFamily,
			Style:  fontstyle.Normal,
			Size:   fontSize,
		}).
		WithPageNumber(props.PageNumber{
			Pattern: "{current}/{total}",
			Place:   props.Bottom,
			Family:  fontFamily,
			Style:   fontstyle.Normal,
			Size:    8,
		}).
		WithAuthor("Liste Tiime", true).
		WithTitle(dataset.Treatment.Label, true).
		WithCompression(true)
	if len(customFonts) > 0 {
		builder = builder.WithCustomFonts(customFonts)
	}

	document := maroto.New(builder.Build())
	header := pdfHeaderRow(dataset.Columns, fontFamily, fontSize)
	if err := document.RegisterHeader(header); err != nil {
		return err
	}
	for _, dataRow := range dataset.Rows {
		document.AddRows(pdfDataRow(dataset.Columns, dataRow, fontFamily, fontSize))
	}

	generated, err := document.Generate()
	if err != nil {
		return err
	}
	return generated.Save(outputPath)
}

func pdfFonts() (string, []*entity.CustomFont) {
	fontsDirectory := filepath.Join(os.Getenv("WINDIR"), "Fonts")
	normalPath := filepath.Join(fontsDirectory, "segoeui.ttf")
	boldPath := filepath.Join(fontsDirectory, "segoeuib.ttf")
	normalBytes, normalErr := os.ReadFile(normalPath)
	boldBytes, boldErr := os.ReadFile(boldPath)
	if normalErr != nil || boldErr != nil {
		return fontfamily.Arial, nil
	}
	return "Segoe UI", []*entity.CustomFont{
		{Family: "Segoe UI", Style: fontstyle.Normal, Bytes: normalBytes},
		{Family: "Segoe UI", Style: fontstyle.Bold, Bytes: boldBytes},
	}
}

func pdfHeaderRow(columns []ExportColumn, family string, fontSize float64) marotoCore.Row {
	headerColor := &props.Color{Red: 31, Green: 78, Blue: 120}
	white := &props.Color{Red: 255, Green: 255, Blue: 255}
	borderColor := &props.Color{Red: 128, Green: 128, Blue: 128}
	widths := pdfColumnWidths(columns)
	columnsToAdd := make([]marotoCore.Col, 0, len(columns))
	for index, column := range columns {
		cell := col.New(widths[index]).Add(text.New(column.Header, props.Text{
			Top:    1.7,
			Left:   0.8,
			Right:  0.8,
			Family: family,
			Style:  fontstyle.Bold,
			Size:   fontSize,
			Align:  align.Center,
			Color:  white,
		})).WithStyle(&props.Cell{
			BackgroundColor: headerColor,
			BorderColor:     borderColor,
			BorderType:      border.Full,
			BorderThickness: 0.25,
		})
		columnsToAdd = append(columnsToAdd, cell)
	}
	return row.New(7).Add(columnsToAdd...)
}

func pdfDataRow(columns []ExportColumn, data ProcessedRow, family string, fontSize float64) marotoCore.Row {
	height := pdfRowHeight(data.MovementLabel, len(columns))
	widths := pdfColumnWidths(columns)
	borderColor := &props.Color{Red: 128, Green: 128, Blue: 128}
	columnsToAdd := make([]marotoCore.Col, 0, len(columns))

	for index, column := range columns {
		value := pdfValue(column.Kind, data)
		top := maxFloat(1.2, (height-3.3)/2)
		if column.Kind == ColumnMovement {
			top = 1.2
		}
		textAlign := align.Left
		if column.Kind == ColumnPrimaryAmount || column.Kind == ColumnSecondaryAmount {
			textAlign = align.Center
		}
		cell := col.New(widths[index]).Add(text.New(value, props.Text{
			Top:             top,
			Left:            0.8,
			Right:           0.8,
			Family:          family,
			Style:           fontstyle.Normal,
			Size:            fontSize,
			Align:           textAlign,
			VerticalPadding: 0.25,
		}))
		if strings.TrimSpace(value) != "" {
			cell = cell.WithStyle(&props.Cell{
				BorderColor:     borderColor,
				BorderType:      border.Full,
				BorderThickness: 0.25,
			})
		}
		columnsToAdd = append(columnsToAdd, cell)
	}
	return row.New(height).Add(columnsToAdd...)
}

func pdfColumnWidths(columns []ExportColumn) []int {
	if len(columns) > 6 {
		return []int{15, 8, 5, 40, 9, 9, 14}
	}
	return []int{17, 9, 6, 43, 11, 14}
}

func pdfRowHeight(movement string, columnCount int) float64 {
	charactersPerLine := 64
	if columnCount > 6 {
		charactersPerLine = 58
	}
	length := len([]rune(strings.TrimSpace(movement)))
	if length < 1 {
		length = 1
	}
	lines := (length + charactersPerLine - 1) / charactersPerLine
	height := 7.0 + float64(lines-1)*3.6
	if height > 23 {
		height = 23
	}
	return height
}

func pdfValue(kind ColumnKind, data ProcessedRow) string {
	switch kind {
	case ColumnAccountLabel:
		return data.AccountLabel
	case ColumnDate:
		if data.Date != nil {
			return data.Date.Format("02/01/2006")
		}
	case ColumnJournal:
		return data.Journal
	case ColumnMovement:
		return data.MovementLabel
	case ColumnPrimaryAmount:
		if data.PrimaryAmount != nil {
			return FormatAmount(*data.PrimaryAmount) + " €"
		}
	case ColumnSecondaryAmount:
		if data.SecondaryAmount != nil {
			return FormatAmount(*data.SecondaryAmount) + " €"
		}
	case ColumnRemark:
		return data.Remark
	}
	return ""
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func validatePDF(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		return fmt.Errorf("PDF invalide")
	}
	return nil
}

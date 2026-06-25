package core

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

type ExcelExporter struct{}

func NewExcelExporter() *ExcelExporter {
	return &ExcelExporter{}
}

func (e *ExcelExporter) Save(dataset ExportDataset, outputPath string) error {
	workbook := excelize.NewFile()
	defer workbook.Close()

	defaultSheet := workbook.GetSheetName(0)
	sheet := SanitizeWorksheetName(dataset.Treatment.SheetName)
	if err := workbook.SetSheetName(defaultSheet, sheet); err != nil {
		return err
	}
	if err := workbook.SetDefaultFont(DefaultFontName); err != nil {
		return err
	}

	styles, err := buildExcelStyles(workbook)
	if err != nil {
		return err
	}
	if err := writeExcelHeaders(workbook, sheet, dataset.Columns, styles.header); err != nil {
		return err
	}
	if err := writeExcelRows(workbook, sheet, dataset, styles); err != nil {
		return err
	}
	if err := applyExcelLayout(workbook, sheet, dataset); err != nil {
		return err
	}
	return workbook.SaveAs(outputPath)
}

type excelStyles struct {
	header   int
	text     int
	movement int
	date     int
	amount   int
}

func buildExcelStyles(workbook *excelize.File) (excelStyles, error) {
	borders := []excelize.Border{
		{Type: "left", Color: BorderGray, Style: 1},
		{Type: "top", Color: BorderGray, Style: 1},
		{Type: "right", Color: BorderGray, Style: 1},
		{Type: "bottom", Color: BorderGray, Style: 1},
	}
	header, err := workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Family: DefaultFontName,
			Size:   HeaderFontSize,
			Color:  "FFFFFF",
		},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{HeaderBlue}},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: borders,
	})
	if err != nil {
		return excelStyles{}, err
	}

	base := excelize.Style{
		Font:      &excelize.Font{Family: DefaultFontName, Size: DataFontSize},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    borders,
	}
	textStyle, err := workbook.NewStyle(&base)
	if err != nil {
		return excelStyles{}, err
	}

	movementStyle := base
	movementStyle.Alignment = &excelize.Alignment{Vertical: "center", WrapText: true}
	movement, err := workbook.NewStyle(&movementStyle)
	if err != nil {
		return excelStyles{}, err
	}

	dateStyle := base
	dateFormat := "dd/mm/yyyy"
	dateStyle.CustomNumFmt = &dateFormat
	date, err := workbook.NewStyle(&dateStyle)
	if err != nil {
		return excelStyles{}, err
	}

	amountStyle := base
	amountFormat := "#,##0.00 \"€\""
	amountStyle.CustomNumFmt = &amountFormat
	amountStyle.Alignment = &excelize.Alignment{Horizontal: "center", Vertical: "center"}
	amount, err := workbook.NewStyle(&amountStyle)
	if err != nil {
		return excelStyles{}, err
	}

	return excelStyles{
		header:   header,
		text:     textStyle,
		movement: movement,
		date:     date,
		amount:   amount,
	}, nil
}

func writeExcelHeaders(workbook *excelize.File, sheet string, columns []ExportColumn, style int) error {
	for columnIndex, column := range columns {
		cell, _ := excelize.CoordinatesToCellName(columnIndex+1, 1)
		if err := workbook.SetCellValue(sheet, cell, column.Header); err != nil {
			return err
		}
		if err := workbook.SetCellStyle(sheet, cell, cell, style); err != nil {
			return err
		}
	}
	return workbook.SetRowHeight(sheet, 1, 22)
}

func writeExcelRows(workbook *excelize.File, sheet string, dataset ExportDataset, styles excelStyles) error {
	for rowIndex, row := range dataset.Rows {
		excelRow := rowIndex + 2
		if err := workbook.SetRowHeight(sheet, excelRow, calculateRowHeight(row.MovementLabel)); err != nil {
			return err
		}

		for columnIndex, column := range dataset.Columns {
			value, hasValue := excelCellValue(column.Kind, row)
			if !hasValue {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(columnIndex+1, excelRow)
			if err := workbook.SetCellValue(sheet, cell, value); err != nil {
				return err
			}
			style := styles.text
			switch column.Kind {
			case ColumnMovement:
				style = styles.movement
			case ColumnDate:
				style = styles.date
			case ColumnPrimaryAmount, ColumnSecondaryAmount:
				style = styles.amount
			}
			if err := workbook.SetCellStyle(sheet, cell, cell, style); err != nil {
				return err
			}
		}
	}
	return nil
}

func excelCellValue(kind ColumnKind, row ProcessedRow) (any, bool) {
	switch kind {
	case ColumnAccountLabel:
		return row.AccountLabel, strings.TrimSpace(row.AccountLabel) != ""
	case ColumnDate:
		if row.Date == nil {
			return nil, false
		}
		return *row.Date, true
	case ColumnJournal:
		return row.Journal, strings.TrimSpace(row.Journal) != ""
	case ColumnMovement:
		return row.MovementLabel, strings.TrimSpace(row.MovementLabel) != ""
	case ColumnPrimaryAmount:
		if row.PrimaryAmount == nil {
			return nil, false
		}
		value, _ := row.PrimaryAmount.Float64()
		return value, true
	case ColumnSecondaryAmount:
		if row.SecondaryAmount == nil {
			return nil, false
		}
		value, _ := row.SecondaryAmount.Float64()
		return value, true
	case ColumnRemark:
		return row.Remark, strings.TrimSpace(row.Remark) != ""
	default:
		return nil, false
	}
}

func applyExcelLayout(workbook *excelize.File, sheet string, dataset ExportDataset) error {
	for index, column := range dataset.Columns {
		width := 14.0
		switch column.Kind {
		case ColumnAccountLabel:
			width = 24
		case ColumnDate:
			width = 12
		case ColumnJournal:
			width = 9
		case ColumnMovement:
			width = 76
		case ColumnPrimaryAmount, ColumnSecondaryAmount:
			width = 16
		case ColumnRemark:
			width = 24
		}
		name, _ := excelize.ColumnNumberToName(index + 1)
		if err := workbook.SetColWidth(sheet, name, name, width); err != nil {
			return err
		}
	}

	lastRow := len(dataset.Rows) + 1
	lastColumn, _ := excelize.ColumnNumberToName(len(dataset.Columns))
	rangeRef := fmt.Sprintf("A1:%s%d", lastColumn, lastRow)
	if err := workbook.AutoFilter(sheet, rangeRef, []excelize.AutoFilterOptions{}); err != nil {
		return err
	}
	if err := workbook.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return err
	}

	showGridLines := false
	if err := workbook.SetSheetView(sheet, 0, &excelize.ViewOptions{ShowGridLines: &showGridLines}); err != nil {
		return err
	}
	fitToPage := true
	if err := workbook.SetSheetProps(sheet, &excelize.SheetPropsOptions{FitToPage: &fitToPage}); err != nil {
		return err
	}

	paperA4 := 9
	orientation := "portrait"
	fitWidth := 1
	fitHeight := 0
	if err := workbook.SetPageLayout(sheet, &excelize.PageLayoutOptions{
		Size:          &paperA4,
		Orientation:   &orientation,
		FitToWidth:    &fitWidth,
		FitToHeight:   &fitHeight,
		BlackAndWhite: boolPtr(false),
	}); err != nil {
		return err
	}

	left, right, top, bottom, footer := 0.3, 0.3, 0.5, 0.5, 0.2
	if err := workbook.SetPageMargins(sheet, &excelize.PageLayoutMarginsOptions{
		Left:   &left,
		Right:  &right,
		Top:    &top,
		Bottom: &bottom,
		Footer: &footer,
	}); err != nil {
		return err
	}
	if err := workbook.SetHeaderFooter(sheet, &excelize.HeaderFooterOptions{
		OddFooter:  "&C&P/&N",
		EvenFooter: "&C&P/&N",
	}); err != nil {
		return err
	}

	quotedSheet := "'" + strings.ReplaceAll(sheet, "'", "''") + "'"
	if err := workbook.SetDefinedName(&excelize.DefinedName{
		Name:     "_xlnm.Print_Area",
		RefersTo: fmt.Sprintf("%s!$A$1:$%s$%d", quotedSheet, lastColumn, lastRow),
		Scope:    sheet,
	}); err != nil {
		return err
	}
	return workbook.SetDefinedName(&excelize.DefinedName{
		Name:     "_xlnm.Print_Titles",
		RefersTo: quotedSheet + "!$1:$1",
		Scope:    sheet,
	})
}

func boolPtr(value bool) *bool {
	return &value
}

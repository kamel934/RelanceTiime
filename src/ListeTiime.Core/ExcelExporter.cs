using ClosedXML.Excel;

namespace ListeTiime.Core;

public sealed class ExcelExporter
{
    public void Save(ExportDataset dataset, string outputPath)
    {
        using var workbook = new XLWorkbook();
        var worksheet = workbook.Worksheets.Add(TextTools.SanitizeWorksheetName(dataset.Treatment.SheetName));
        worksheet.ShowGridLines = false;

        WriteHeaders(worksheet, dataset.Columns);
        WriteRows(worksheet, dataset);
        ApplyColumnWidths(worksheet, dataset.Columns);
        ApplyPageSetup(worksheet, dataset.Rows.Count + 1, dataset.Columns.Count);

        workbook.SaveAs(outputPath);
    }

    private static void WriteHeaders(IXLWorksheet worksheet, IReadOnlyList<ExportColumn> columns)
    {
        for (var columnIndex = 0; columnIndex < columns.Count; columnIndex++)
        {
            var cell = worksheet.Cell(1, columnIndex + 1);
            cell.Value = columns[columnIndex].Header;
            cell.Style.Font.FontName = AppConstants.DefaultFontName;
            cell.Style.Font.FontSize = AppConstants.HeaderFontSize;
            cell.Style.Font.Bold = true;
            cell.Style.Font.FontColor = XLColor.White;
            cell.Style.Fill.BackgroundColor = XLColor.FromHtml(AppConstants.HeaderBlue);
            cell.Style.Alignment.Horizontal = XLAlignmentHorizontalValues.Center;
            cell.Style.Alignment.Vertical = XLAlignmentVerticalValues.Center;
            ApplyBorder(cell);
        }

        worksheet.Row(1).Height = 22d;
    }

    private static void WriteRows(IXLWorksheet worksheet, ExportDataset dataset)
    {
        for (var rowIndex = 0; rowIndex < dataset.Rows.Count; rowIndex++)
        {
            var row = dataset.Rows[rowIndex];
            var excelRow = rowIndex + 2;
            worksheet.Row(excelRow).Height = CalculateRowHeight(row.MovementLabel);

            for (var columnIndex = 0; columnIndex < dataset.Columns.Count; columnIndex++)
            {
                var column = dataset.Columns[columnIndex];
                var cell = worksheet.Cell(excelRow, columnIndex + 1);
                var hasValue = SetCellValue(cell, column.Kind, row);

                cell.Style.Font.FontName = AppConstants.DefaultFontName;
                cell.Style.Font.FontSize = AppConstants.DataFontSize;
                cell.Style.Alignment.Vertical = XLAlignmentVerticalValues.Center;

                if (column.Kind == ExportColumnKind.Movement)
                {
                    cell.Style.Alignment.WrapText = true;
                }

                if (column.Kind is ExportColumnKind.PrimaryAmount or ExportColumnKind.SecondaryAmount)
                {
                    cell.Style.Alignment.Horizontal = XLAlignmentHorizontalValues.Center;
                    cell.Style.NumberFormat.Format = "#,##0.00 \"€\"";
                }

                if (column.Kind == ExportColumnKind.Date)
                {
                    cell.Style.DateFormat.Format = "dd/mm/yyyy";
                }

                if (hasValue)
                {
                    ApplyBorder(cell);
                }
            }
        }
    }

    private static bool SetCellValue(IXLCell cell, ExportColumnKind kind, ProcessedRow row)
    {
        switch (kind)
        {
            case ExportColumnKind.AccountLabel:
                cell.Value = row.AccountLabel;
                return !string.IsNullOrWhiteSpace(row.AccountLabel);
            case ExportColumnKind.Date:
                if (row.Date is null)
                {
                    return false;
                }

                cell.Value = row.Date.Value;
                return true;
            case ExportColumnKind.Journal:
                cell.Value = row.Journal;
                return !string.IsNullOrWhiteSpace(row.Journal);
            case ExportColumnKind.Movement:
                cell.Value = row.MovementLabel;
                return !string.IsNullOrWhiteSpace(row.MovementLabel);
            case ExportColumnKind.PrimaryAmount:
                if (row.PrimaryAmount is null)
                {
                    return false;
                }

                cell.Value = row.PrimaryAmount.Value;
                return true;
            case ExportColumnKind.SecondaryAmount:
                if (row.SecondaryAmount is null)
                {
                    return false;
                }

                cell.Value = row.SecondaryAmount.Value;
                return true;
            case ExportColumnKind.Remark:
                cell.Value = row.Remark;
                return !string.IsNullOrWhiteSpace(row.Remark);
            default:
                throw new ArgumentOutOfRangeException(nameof(kind), kind, null);
        }
    }

    private static void ApplyColumnWidths(IXLWorksheet worksheet, IReadOnlyList<ExportColumn> columns)
    {
        for (var index = 0; index < columns.Count; index++)
        {
            var width = columns[index].Kind switch
            {
                ExportColumnKind.AccountLabel => 24d,
                ExportColumnKind.Date => 12d,
                ExportColumnKind.Journal => 9d,
                ExportColumnKind.Movement => 76d,
                ExportColumnKind.PrimaryAmount => 16d,
                ExportColumnKind.SecondaryAmount => 16d,
                ExportColumnKind.Remark => 24d,
                _ => 14d
            };

            worksheet.Column(index + 1).Width = width;
        }
    }

    private static void ApplyPageSetup(IXLWorksheet worksheet, int lastRow, int lastColumn)
    {
        var range = worksheet.Range(1, 1, lastRow, lastColumn);
        range.SetAutoFilter();
        worksheet.SheetView.FreezeRows(1);

        worksheet.PageSetup.PageOrientation = XLPageOrientation.Portrait;
        worksheet.PageSetup.PaperSize = XLPaperSize.A4Paper;
        worksheet.PageSetup.FitToPages(1, 0);
        worksheet.PageSetup.PrintAreas.Clear();
        worksheet.PageSetup.PrintAreas.Add(1, 1, lastRow, lastColumn);
        worksheet.PageSetup.SetRowsToRepeatAtTop(1, 1);
        worksheet.PageSetup.Margins.Left = 0.3;
        worksheet.PageSetup.Margins.Right = 0.3;
        worksheet.PageSetup.Margins.Top = 0.5;
        worksheet.PageSetup.Margins.Bottom = 0.5;
        worksheet.PageSetup.Margins.Footer = 0.2;
        worksheet.PageSetup.Footer.Center.AddText("&P/&N");
    }

    private static void ApplyBorder(IXLCell cell)
    {
        var border = cell.Style.Border;
        border.OutsideBorder = XLBorderStyleValues.Thin;
        border.InsideBorder = XLBorderStyleValues.Thin;
        border.OutsideBorderColor = XLColor.FromHtml(AppConstants.BorderGray);
        border.InsideBorderColor = XLColor.FromHtml(AppConstants.BorderGray);
    }

    private static double CalculateRowHeight(string movementLabel)
    {
        var length = string.IsNullOrWhiteSpace(movementLabel) ? 1 : movementLabel.Length;
        var lineCount = Math.Max(1, (int)Math.Ceiling(length / (double)AppConstants.MovementCharsPerLine));

        if (lineCount == 1)
        {
            return AppConstants.BaseRowHeight;
        }

        return Math.Min(
            AppConstants.MaxRowHeight,
            AppConstants.BaseRowHeight + ((lineCount - 1) * AppConstants.WrappedLineHeight) + AppConstants.RowHeightPadding);
    }
}

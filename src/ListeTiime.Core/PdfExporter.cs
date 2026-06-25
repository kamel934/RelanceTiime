using MigraDoc.DocumentObjectModel;
using MigraDoc.DocumentObjectModel.Tables;
using MigraDoc.Rendering;

namespace ListeTiime.Core;

public sealed class PdfExporter
{
    public void Save(ExportDataset dataset, string outputPath)
    {
        var document = BuildDocument(dataset);
        var renderer = new PdfDocumentRenderer
        {
            Document = document
        };
        renderer.RenderDocument();
        renderer.Save(outputPath);
    }

    private static Document BuildDocument(ExportDataset dataset)
    {
        var document = new Document
        {
            Info =
            {
                Title = dataset.Treatment.Label,
                Author = "Liste Tiime"
            }
        };

        document.Styles["Normal"]!.Font.Name = AppConstants.DefaultFontName;
        document.Styles["Normal"]!.Font.Size = dataset.Columns.Count > 6 ? 7.2 : 7.6;

        var section = document.AddSection();
        section.PageSetup.PageFormat = PageFormat.A4;
        section.PageSetup.Orientation = Orientation.Portrait;
        section.PageSetup.LeftMargin = Unit.FromMillimeter(8);
        section.PageSetup.RightMargin = Unit.FromMillimeter(8);
        section.PageSetup.TopMargin = Unit.FromMillimeter(8);
        section.PageSetup.BottomMargin = Unit.FromMillimeter(14);
        section.PageSetup.FooterDistance = Unit.FromMillimeter(5);

        var footer = section.Footers.Primary.AddParagraph();
        footer.Format.Alignment = ParagraphAlignment.Center;
        footer.Format.Font.Size = 8;
        footer.AddPageField();
        footer.AddText("/");
        footer.AddNumPagesField();

        var table = section.AddTable();
        table.Borders.Width = 0.25;
        table.Borders.Color = Color.FromRgb(128, 128, 128);
        table.Format.Alignment = ParagraphAlignment.Left;
        table.Rows.LeftIndent = 0;

        foreach (var width in CalculateColumnWidths(dataset.Columns))
        {
            var column = table.AddColumn(Unit.FromMillimeter(width));
            column.Format.Alignment = ParagraphAlignment.Left;
        }

        AddHeaderRow(table, dataset.Columns);
        foreach (var row in dataset.Rows)
        {
            AddDataRow(table, dataset.Columns, row);
        }

        return document;
    }

    private static void AddHeaderRow(Table table, IReadOnlyList<ExportColumn> columns)
    {
        var header = table.AddRow();
        header.HeadingFormat = true;
        header.Shading.Color = Color.FromRgb(31, 78, 120);
        header.Format.Font.Color = Colors.White;
        header.Format.Font.Bold = true;
        header.Format.Alignment = ParagraphAlignment.Center;
        header.VerticalAlignment = VerticalAlignment.Center;

        for (var index = 0; index < columns.Count; index++)
        {
            var paragraph = header.Cells[index].AddParagraph(columns[index].Header);
            paragraph.Format.Alignment = ParagraphAlignment.Center;
            header.Cells[index].Format.LeftIndent = Unit.FromMillimeter(1);
            header.Cells[index].Format.RightIndent = Unit.FromMillimeter(1);
        }
    }

    private static void AddDataRow(Table table, IReadOnlyList<ExportColumn> columns, ProcessedRow row)
    {
        var pdfRow = table.AddRow();
        pdfRow.VerticalAlignment = VerticalAlignment.Center;

        for (var index = 0; index < columns.Count; index++)
        {
            var column = columns[index];
            var cell = pdfRow.Cells[index];
            var paragraph = cell.AddParagraph(FormatValue(column.Kind, row));
            paragraph.Format.Alignment = ShouldCenter(column.Kind)
                ? ParagraphAlignment.Center
                : ParagraphAlignment.Left;
            cell.Format.LeftIndent = Unit.FromMillimeter(1);
            cell.Format.RightIndent = Unit.FromMillimeter(1);
            cell.Format.SpaceBefore = Unit.FromMillimeter(0.5);
            cell.Format.SpaceAfter = Unit.FromMillimeter(0.5);
        }
    }

    private static string FormatValue(ExportColumnKind kind, ProcessedRow row)
    {
        return kind switch
        {
            ExportColumnKind.AccountLabel => row.AccountLabel,
            ExportColumnKind.Date => row.Date?.ToString("dd/MM/yyyy", AppConstants.FrenchCulture) ?? string.Empty,
            ExportColumnKind.Journal => row.Journal,
            ExportColumnKind.Movement => row.MovementLabel,
            ExportColumnKind.PrimaryAmount => row.PrimaryAmount is null ? string.Empty : $"{TextTools.FormatAmount(row.PrimaryAmount.Value)} €",
            ExportColumnKind.SecondaryAmount => row.SecondaryAmount is null ? string.Empty : $"{TextTools.FormatAmount(row.SecondaryAmount.Value)} €",
            ExportColumnKind.Remark => row.Remark,
            _ => string.Empty
        };
    }

    private static bool ShouldCenter(ExportColumnKind kind)
    {
        return kind is ExportColumnKind.PrimaryAmount or ExportColumnKind.SecondaryAmount;
    }

    private static IReadOnlyList<double> CalculateColumnWidths(IReadOnlyList<ExportColumn> columns)
    {
        var totalWidth = 194d;
        var weights = columns.Count > 6
            ? new[] { 1.2, 0.72, 0.48, 2.7, 0.92, 0.92, 1.2 }
            : new[] { 1.35, 0.78, 0.52, 3.1, 1.05, 1.35 };
        var totalWeight = weights.Sum();
        return weights.Take(columns.Count).Select(weight => totalWidth * weight / totalWeight).ToList();
    }
}

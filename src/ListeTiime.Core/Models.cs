using System.Globalization;

namespace ListeTiime.Core;

[Flags]
public enum OutputFormats
{
    None = 0,
    Excel = 1,
    Pdf = 2
}

public enum LedgerType
{
    Supplier,
    Customer
}

public enum LedgerMode
{
    Auto,
    Supplier,
    Customer
}

public enum TreatmentRole
{
    First,
    Second,
    Both
}

public enum TreatmentKind
{
    SupplierPayments,
    SupplierInvoices,
    CustomerReceipts,
    CustomerInvoices
}

public enum ExportColumnKind
{
    AccountLabel,
    Date,
    Journal,
    Movement,
    PrimaryAmount,
    SecondaryAmount,
    Remark
}

public sealed record FileMetadata(string Client, string Year);

public sealed record CsvLedgerEntry(
    string AccountNumber,
    string AccountLabel,
    DateTime? Date,
    string Journal,
    string PieceNumber,
    string MovementLabel,
    decimal Debit,
    decimal Credit);

public sealed record TreatmentDefinition(
    TreatmentKind Kind,
    LedgerType LedgerType,
    TreatmentRole Role,
    string Label,
    string OutputLabel,
    string SheetName,
    string AmountLabel,
    string PrimaryColumn,
    string SecondaryColumn);

public sealed record ProcessedRow(
    string AccountLabel,
    DateTime? Date,
    string Journal,
    string MovementLabel,
    decimal? PrimaryAmount,
    decimal? SecondaryAmount,
    string Remark);

public sealed record ExportColumn(string Header, ExportColumnKind Kind);

public sealed record ExportDataset(
    TreatmentDefinition Treatment,
    IReadOnlyList<ProcessedRow> Rows,
    IReadOnlyList<ExportColumn> Columns)
{
    public decimal TotalPrimary => Rows.Sum(row => row.PrimaryAmount ?? 0m);
}

public sealed record GeneratedDocument(
    string SourcePath,
    TreatmentDefinition Treatment,
    int RowCount,
    decimal Total,
    string? ExcelPath,
    string? PdfPath)
{
    public bool HasOutput => ExcelPath is not null || PdfPath is not null;
}

public sealed record BatchResult(IReadOnlyList<GeneratedDocument> Documents)
{
    public bool HasOutput => Documents.Any(document => document.HasOutput);
}

public sealed record ProcessingOptions(
    OutputFormats Formats,
    LedgerMode LedgerMode,
    TreatmentRole Role,
    Func<string, FileMetadata?>? MetadataProvider = null);

public static class AppConstants
{
    public const string AppName = "Liste Tiime";
    public const string SettingsDirectoryName = "RelanceTiime";
    public const string SettingsFileName = "settings.json";
    public const string DefaultFontName = "Segoe UI";
    public const double DataFontSize = 11d;
    public const double HeaderFontSize = 11d;
    public const int MovementCharsPerLine = 64;
    public const double BaseRowHeight = 22d;
    public const double WrappedLineHeight = 14d;
    public const double RowHeightPadding = 4d;
    public const double MaxRowHeight = 70d;
    public const string HeaderBlue = "1F4E78";
    public const string BorderGray = "808080";

    public static readonly CultureInfo FrenchCulture = CultureInfo.GetCultureInfo("fr-FR");
}

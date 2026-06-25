using System.Text;
using ClosedXML.Excel;
using ListeTiime.Core;

namespace ListeTiime.Tests;

public sealed class LedgerProcessorTests : IDisposable
{
    private readonly string _tempDirectory = Path.Combine(Path.GetTempPath(), "liste-tiime-tests", Guid.NewGuid().ToString("N"));

    public LedgerProcessorTests()
    {
        Directory.CreateDirectory(_tempDirectory);
    }

    [Fact]
    public void SupplierPayments_GeneratesExcelAndPdf_WithoutEmptyRefundColumn()
    {
        var csvPath = WriteCsv(
            "grand_livre_2025-01-01_2025-12-31_jennah_boutique.csv",
            new[]
            {
                Row("401000", "BRINKS", "19/09/2025", "BQ", "", "18/09/25 18H56 PAIEMENT SERV DRA", "250,00", "0"),
                Row("401000", "BRINKS", "19/09/2025", "BQ", "", "18/09/25 18H57 PAIEMENT SERV DRA", "250,00", "0"),
                Row("401001", "MS STYLE", "11/09/2025", "BNP", "", "VIR SCT INST EMIS MOTIF 20250907 BEN SAS MS SERR", "840,00", "0"),
                Row("401002", "FOURNISSEUR AO", "12/09/2025", "AO", "", "A exclure du traitement paiements", "999,00", "0"),
                Row("401003", "FACTURE", "13/09/2025", "AC", "", "A exclure du traitement paiements", "0", "100,00")
            });

        var result = new LedgerProcessor().ProcessFiles(
            [csvPath],
            new ProcessingOptions(OutputFormats.Excel | OutputFormats.Pdf, LedgerMode.Auto, TreatmentRole.First));

        var document = Assert.Single(result.Documents);
        Assert.True(document.HasOutput);
        Assert.Equal(3, document.RowCount);
        Assert.Equal(1340m, document.Total);
        Assert.NotNull(document.ExcelPath);
        Assert.NotNull(document.PdfPath);
        Assert.True(File.Exists(document.ExcelPath));
        Assert.True(File.Exists(document.PdfPath));
        var pdfBytes = File.ReadAllBytes(document.PdfPath!);
        Assert.True(pdfBytes.Take(5).SequenceEqual("%PDF-"u8.ToArray()));

        using var workbook = new XLWorkbook(document.ExcelPath);
        var worksheet = workbook.Worksheet(1);
        Assert.Equal("Paiements", worksheet.Cell(1, 5).GetString());
        Assert.Equal("Remarque", worksheet.Cell(1, 6).GetString());
        Assert.DoesNotContain("Rmbrsmts", Headers(worksheet));
        Assert.Equal(76d, Math.Round(worksheet.Column(4).Width));
        Assert.True(worksheet.Row(2).Height >= 22d);
        Assert.Equal("Manque facture", worksheet.Cell(2, 6).GetString());
    }

    [Fact]
    public void SupplierInvoices_HidesEmptyAvoirsColumn()
    {
        var csvPath = WriteCsv(
            "grand_livre_2025-01-01_2025-12-31_jennah_boutique.csv",
            new[]
            {
                Row("401000", "FOURNISSEUR", "19/09/2025", "AC", "FAC001", "FACTURE ACHAT", "0", "1000,00")
            });

        var result = new LedgerProcessor().ProcessFiles(
            [csvPath],
            new ProcessingOptions(OutputFormats.Excel, LedgerMode.Auto, TreatmentRole.Second));

        var document = Assert.Single(result.Documents);
        using var workbook = new XLWorkbook(document.ExcelPath);
        var worksheet = workbook.Worksheet(1);
        Assert.Equal(["Libellé du compte", "Date", "Journal", "Libellé mouvement", "Factures", "Remarque"], Headers(worksheet));
        Assert.Equal("Facture sans paiement", worksheet.Cell(2, 6).GetString());
    }

    [Fact]
    public void SupplierInvoices_KeepsAvoirsColumnAndRemarkWhenDebitExists()
    {
        var csvPath = WriteCsv(
            "grand_livre_2025-01-01_2025-12-31_jennah_boutique.csv",
            new[]
            {
                Row("401000", "FOURNISSEUR", "19/09/2025", "AC", "FAC001", "FACTURE ACHAT", "0", "1000,00"),
                Row("401000", "FOURNISSEUR", "20/09/2025", "AT", "AV001", "AVOIR ACHAT", "120,00", "0")
            });

        var result = new LedgerProcessor().ProcessFiles(
            [csvPath],
            new ProcessingOptions(OutputFormats.Excel, LedgerMode.Auto, TreatmentRole.Second));

        var document = Assert.Single(result.Documents);
        using var workbook = new XLWorkbook(document.ExcelPath);
        var worksheet = workbook.Worksheet(1);
        Assert.Contains("Avoirs", Headers(worksheet));
        var remarks = worksheet.Column(7).CellsUsed().Skip(1).Select(cell => cell.GetString()).ToArray();
        Assert.Contains("Avoir sans rmbrsmt", remarks);
        Assert.Contains("Facture sans paiement", remarks);
    }

    [Fact]
    public void CustomerLedger_AutoDetectsAndGeneratesBothLists()
    {
        var csvPath = WriteCsv(
            "grand_livre_2025-01-01_2025-12-31_saint_ouen_auto_services.csv",
            new[]
            {
                Row("411000", "CLIENT A", "02/01/2025", "BQ", "", "VIREMENT CLIENT A", "0", "600,00"),
                Row("411000", "CLIENT A", "03/01/2025", "VI", "FV001", "FACTURE VENTE", "1000,00", "0"),
                Row("411001", "CLIENT B", "04/01/2025", "VE", "FV002", "FACTURE VENTE", "500,00", "0"),
                Row("411001", "CLIENT B", "05/01/2025", "BQ", "LET001", "ENCAISSEMENT LETTRE", "0", "50,00")
            });

        var result = new LedgerProcessor().ProcessFiles(
            [csvPath],
            new ProcessingOptions(OutputFormats.Excel, LedgerMode.Auto, TreatmentRole.Both));

        Assert.Equal(2, result.Documents.Count);
        var receipts = result.Documents.Single(document => document.Treatment.Kind == TreatmentKind.CustomerReceipts);
        var invoices = result.Documents.Single(document => document.Treatment.Kind == TreatmentKind.CustomerInvoices);
        Assert.Equal(1, receipts.RowCount);
        Assert.Equal(600m, receipts.Total);
        Assert.Equal(2, invoices.RowCount);
        Assert.Equal(1500m, invoices.Total);

        using var receiptsWorkbook = new XLWorkbook(receipts.ExcelPath);
        Assert.Contains("Encaissements", Headers(receiptsWorkbook.Worksheet(1)));
        Assert.DoesNotContain("Remboursements", Headers(receiptsWorkbook.Worksheet(1)));

        using var invoicesWorkbook = new XLWorkbook(invoices.ExcelPath);
        Assert.Contains("Factures", Headers(invoicesWorkbook.Worksheet(1)));
        Assert.DoesNotContain("Avoirs", Headers(invoicesWorkbook.Worksheet(1)));
    }

    [Fact]
    public void CustomerLedger_KeepsRefundAndAvoirColumnsWhenNeeded()
    {
        var csvPath = WriteCsv(
            "grand_livre_2025-01-01_2025-12-31_saint_ouen_auto_services.csv",
            new[]
            {
                Row("411000", "CLIENT A", "02/01/2025", "BQ", "", "REMBOURSEMENT CLIENT A", "20,00", "0"),
                Row("411000", "CLIENT A", "03/01/2025", "VT", "AVV001", "AVOIR VENTE", "0", "30,00")
            });

        var result = new LedgerProcessor().ProcessFiles(
            [csvPath],
            new ProcessingOptions(OutputFormats.Excel, LedgerMode.Customer, TreatmentRole.Both));

        var receipts = result.Documents.Single(document => document.Treatment.Kind == TreatmentKind.CustomerReceipts);
        var invoices = result.Documents.Single(document => document.Treatment.Kind == TreatmentKind.CustomerInvoices);

        using var receiptsWorkbook = new XLWorkbook(receipts.ExcelPath);
        var receiptsWorksheet = receiptsWorkbook.Worksheet(1);
        Assert.Contains("Remboursements", Headers(receiptsWorksheet));
        Assert.Equal("Manque avoir de vente", receiptsWorksheet.Cell(2, 7).GetString());

        using var invoicesWorkbook = new XLWorkbook(invoices.ExcelPath);
        var invoicesWorksheet = invoicesWorkbook.Worksheet(1);
        Assert.Contains("Avoirs", Headers(invoicesWorksheet));
        Assert.Equal("Avoir de vente sans rmbrsmt", invoicesWorksheet.Cell(2, 7).GetString());
    }

    [Fact]
    public void EmptyTreatment_DoesNotCreateFile()
    {
        var csvPath = WriteCsv(
            "grand_livre_2025-01-01_2025-12-31_jennah_boutique.csv",
            new[]
            {
                Row("401000", "BRINKS", "19/09/2025", "BQ", "", "PAIEMENT", "250,00", "0")
            });

        var result = new LedgerProcessor().ProcessFiles(
            [csvPath],
            new ProcessingOptions(OutputFormats.Excel | OutputFormats.Pdf, LedgerMode.Auto, TreatmentRole.Second));

        var document = Assert.Single(result.Documents);
        Assert.False(document.HasOutput);
        Assert.Null(document.ExcelPath);
        Assert.Null(document.PdfPath);
        Assert.Empty(Directory.EnumerateFiles(_tempDirectory, "*.xlsx"));
        Assert.Empty(Directory.EnumerateFiles(_tempDirectory, "*.pdf"));
    }

    public void Dispose()
    {
        if (Directory.Exists(_tempDirectory))
        {
            Directory.Delete(_tempDirectory, recursive: true);
        }
    }

    private string WriteCsv(string fileName, IEnumerable<string[]> rows)
    {
        Encoding.RegisterProvider(CodePagesEncodingProvider.Instance);
        var path = Path.Combine(_tempDirectory, fileName);
        var lines = new List<string>
        {
            "N° Compte;Libellé du compte;Date;Journal;N° de pièce;Libellé mouvement;Montant Débit;Montant Crédit"
        };
        lines.AddRange(rows.Select(row => string.Join(';', row)));
        File.WriteAllLines(path, lines, Encoding.GetEncoding(1252));
        return path;
    }

    private static string[] Row(
        string account,
        string accountLabel,
        string date,
        string journal,
        string piece,
        string movement,
        string debit,
        string credit)
    {
        return [account, accountLabel, date, journal, piece, movement, debit, credit];
    }

    private static string[] Headers(IXLWorksheet worksheet)
    {
        return worksheet.Row(1).CellsUsed().Select(cell => cell.GetString()).ToArray();
    }
}

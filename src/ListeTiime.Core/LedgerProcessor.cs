namespace ListeTiime.Core;

public sealed class LedgerProcessor
{
    private readonly ExcelExporter _excelExporter = new();
    private readonly PdfExporter _pdfExporter = new();

    public BatchResult ProcessFiles(IEnumerable<string> paths, ProcessingOptions options)
    {
        if (options.Formats == OutputFormats.None)
        {
            throw new InvalidOperationException("Sélectionnez au moins un format.");
        }

        var pathList = paths.Where(path => !string.IsNullOrWhiteSpace(path)).ToList();
        if (pathList.Count == 0)
        {
            return new BatchResult([]);
        }

        var documents = new List<GeneratedDocument>();
        var errors = new List<string>();

        foreach (var path in pathList)
        {
            try
            {
                documents.AddRange(ProcessFile(path, options));
            }
            catch (Exception exception)
            {
                errors.Add($"{Path.GetFileName(path)} : {exception.Message}");
            }
        }

        if (errors.Count > 0)
        {
            throw new InvalidOperationException(string.Join(Environment.NewLine, errors));
        }

        return new BatchResult(documents);
    }

    public IReadOnlyList<GeneratedDocument> ProcessFile(string path, ProcessingOptions options)
    {
        if (!File.Exists(path))
        {
            throw new FileNotFoundException("Fichier introuvable.", path);
        }

        if (!string.Equals(Path.GetExtension(path), ".csv", StringComparison.OrdinalIgnoreCase))
        {
            throw new InvalidOperationException("Le fichier n'est pas un CSV.");
        }

        var metadata = TextTools.TryParseGrandLivreMetadata(path)
            ?? options.MetadataProvider?.Invoke(path)
            ?? throw new InvalidOperationException(
                "Nom de fichier non standard. Format attendu : grand_livre_2025-01-01_2025-12-31_nom_client.csv");

        var entries = CsvLedgerReader.Read(path);
        var ledgerType = options.LedgerMode switch
        {
            LedgerMode.Supplier => LedgerType.Supplier,
            LedgerMode.Customer => LedgerType.Customer,
            LedgerMode.Auto => CsvLedgerReader.DetectLedgerType(entries),
            _ => throw new ArgumentOutOfRangeException(nameof(options.LedgerMode), options.LedgerMode, null)
        };

        var documents = new List<GeneratedDocument>();
        foreach (var treatment in TreatmentCatalog.For(ledgerType, options.Role))
        {
            documents.Add(ProcessTreatment(path, metadata, entries, treatment, options.Formats));
        }

        return documents;
    }

    public GeneratedDocument ProcessTreatment(
        string csvPath,
        FileMetadata metadata,
        IReadOnlyList<CsvLedgerEntry> entries,
        TreatmentDefinition treatment,
        OutputFormats formats)
    {
        var rows = ApplyTreatment(entries, treatment)
            .OrderBy(row => row.AccountLabel.ToUpperInvariant())
            .ThenBy(row => row.Date ?? DateTime.MaxValue)
            .ThenBy(row => row.Journal)
            .ThenBy(row => row.PrimaryAmount ?? 0m)
            .ThenBy(row => row.SecondaryAmount ?? 0m)
            .ToList();

        if (rows.Count == 0)
        {
            return new GeneratedDocument(csvPath, treatment, 0, 0m, null, null);
        }

        var columns = TreatmentCatalog.BuildColumns(treatment, rows);
        var dataset = new ExportDataset(treatment, rows, columns);
        var outputStem = SanitizeFileName($"{metadata.Client} - {treatment.OutputLabel} {metadata.Year}");
        var paths = ReserveOutputPaths(Path.GetDirectoryName(csvPath) ?? Environment.CurrentDirectory, outputStem, formats);

        if (paths.ExcelPath is not null)
        {
            _excelExporter.Save(dataset, paths.ExcelPath);
        }

        if (paths.PdfPath is not null)
        {
            _pdfExporter.Save(dataset, paths.PdfPath);
        }

        return new GeneratedDocument(csvPath, treatment, rows.Count, dataset.TotalPrimary, paths.ExcelPath, paths.PdfPath);
    }

    public static string BuildSummary(BatchResult result)
    {
        var created = result.Documents.Where(document => document.HasOutput).ToList();
        var skipped = result.Documents.Where(document => !document.HasOutput).ToList();

        if (created.Count == 0)
        {
            return skipped.Count == 0
                ? "Aucun fichier généré."
                : "Aucun fichier créé : aucune ligne trouvée.";
        }

        var lines = new List<string>
        {
            created.SelectMany(document => new[] { document.ExcelPath, document.PdfPath }).Count(path => path is not null) == 1
                ? "Fichier généré :"
                : "Fichiers générés :"
        };

        foreach (var document in created)
        {
            lines.Add(
                $"- {document.Treatment.Label} : {document.RowCount} lignes, {document.Treatment.AmountLabel} : {TextTools.FormatAmount(document.Total)} €");
            if (document.ExcelPath is not null)
            {
                lines.Add($"  Excel : {document.ExcelPath}");
            }

            if (document.PdfPath is not null)
            {
                lines.Add($"  PDF : {document.PdfPath}");
            }
        }

        if (skipped.Count > 0)
        {
            lines.Add(string.Empty);
            lines.Add("Aucun fichier créé pour :");
            lines.AddRange(skipped.Select(document => $"- {document.Treatment.Label} : aucune ligne trouvée"));
        }

        return string.Join(Environment.NewLine, lines);
    }

    private static IEnumerable<ProcessedRow> ApplyTreatment(
        IEnumerable<CsvLedgerEntry> entries,
        TreatmentDefinition treatment)
    {
        foreach (var entry in entries)
        {
            var row = treatment.Kind switch
            {
                TreatmentKind.SupplierPayments => BuildSupplierPaymentRow(entry),
                TreatmentKind.SupplierInvoices => BuildSupplierInvoiceRow(entry),
                TreatmentKind.CustomerReceipts => BuildCustomerReceiptRow(entry),
                TreatmentKind.CustomerInvoices => BuildCustomerInvoiceRow(entry),
                _ => throw new ArgumentOutOfRangeException(nameof(treatment.Kind), treatment.Kind, null)
            };

            if (row is not null)
            {
                yield return row;
            }
        }
    }

    private static ProcessedRow? BuildSupplierPaymentRow(CsvLedgerEntry entry)
    {
        if (!string.IsNullOrWhiteSpace(entry.PieceNumber)
            || TreatmentCatalog.ExcludedSupplierPaymentJournals.Contains(entry.Journal))
        {
            return null;
        }

        return new ProcessedRow(
            entry.AccountLabel,
            entry.Date,
            entry.Journal,
            entry.MovementLabel,
            TextTools.NullIfZero(entry.Debit),
            TextTools.NullIfZero(entry.Credit),
            entry.Debit > 0m ? "Manque facture" : "Manque avoir");
    }

    private static ProcessedRow? BuildSupplierInvoiceRow(CsvLedgerEntry entry)
    {
        if (!TreatmentCatalog.SupplierInvoiceJournals.Contains(entry.Journal))
        {
            return null;
        }

        return new ProcessedRow(
            entry.AccountLabel,
            entry.Date,
            entry.Journal,
            entry.MovementLabel,
            TextTools.NullIfZero(entry.Credit),
            TextTools.NullIfZero(entry.Debit),
            entry.Credit > 0m ? "Facture sans paiement" : "Avoir sans rmbrsmt");
    }

    private static ProcessedRow? BuildCustomerReceiptRow(CsvLedgerEntry entry)
    {
        if (!string.IsNullOrWhiteSpace(entry.PieceNumber)
            || TreatmentCatalog.CustomerReceiptExcludedJournals.Contains(entry.Journal))
        {
            return null;
        }

        return new ProcessedRow(
            entry.AccountLabel,
            entry.Date,
            entry.Journal,
            entry.MovementLabel,
            TextTools.NullIfZero(entry.Credit),
            TextTools.NullIfZero(entry.Debit),
            entry.Credit > 0m ? "Manque facture de vente" : "Manque avoir de vente");
    }

    private static ProcessedRow? BuildCustomerInvoiceRow(CsvLedgerEntry entry)
    {
        if (!TreatmentCatalog.SaleJournals.Contains(entry.Journal))
        {
            return null;
        }

        return new ProcessedRow(
            entry.AccountLabel,
            entry.Date,
            entry.Journal,
            entry.MovementLabel,
            TextTools.NullIfZero(entry.Debit),
            TextTools.NullIfZero(entry.Credit),
            entry.Debit > 0m ? "Facture de vente sans paiement" : "Avoir de vente sans rmbrsmt");
    }

    private static OutputPaths ReserveOutputPaths(string directory, string stem, OutputFormats formats)
    {
        for (var suffix = 1; suffix < 10_000; suffix++)
        {
            var displaySuffix = suffix == 1 ? string.Empty : $" - {suffix}";
            var excelPath = formats.HasFlag(OutputFormats.Excel)
                ? Path.Combine(directory, $"{stem}{displaySuffix}.xlsx")
                : null;
            var pdfPath = formats.HasFlag(OutputFormats.Pdf)
                ? Path.Combine(directory, $"{stem}{displaySuffix}.pdf")
                : null;

            if ((excelPath is null || !File.Exists(excelPath))
                && (pdfPath is null || !File.Exists(pdfPath)))
            {
                return new OutputPaths(excelPath, pdfPath);
            }
        }

        throw new IOException("Impossible de trouver un nom de fichier disponible.");
    }

    private static string SanitizeFileName(string name)
    {
        var invalid = Path.GetInvalidFileNameChars().ToHashSet();
        var cleaned = new string(name.Select(character => invalid.Contains(character) ? ' ' : character).ToArray());
        while (cleaned.Contains("  ", StringComparison.Ordinal))
        {
            cleaned = cleaned.Replace("  ", " ", StringComparison.Ordinal);
        }

        return cleaned.Trim();
    }

    private sealed record OutputPaths(string? ExcelPath, string? PdfPath);
}

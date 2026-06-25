using System.Text;
using Microsoft.VisualBasic.FileIO;

namespace ListeTiime.Core;

public static class CsvLedgerReader
{
    private static readonly string[] RequiredHeaders =
    [
        "libelle_du_compte",
        "date",
        "journal",
        "n_de_piece",
        "libelle_mouvement",
        "montant_debit",
        "montant_credit"
    ];

    public static IReadOnlyList<CsvLedgerEntry> Read(string path)
    {
        Encoding.RegisterProvider(CodePagesEncodingProvider.Instance);

        using var parser = new TextFieldParser(path, Encoding.GetEncoding(1252));
        parser.TextFieldType = FieldType.Delimited;
        parser.SetDelimiters(";");
        parser.HasFieldsEnclosedInQuotes = true;
        parser.TrimWhiteSpace = false;

        if (parser.EndOfData)
        {
            throw new InvalidDataException("Le CSV est vide.");
        }

        var headers = parser.ReadFields() ?? [];
        var headerMap = BuildHeaderMap(headers);
        var missing = RequiredHeaders.Where(header => !headerMap.ContainsKey(header)).ToList();
        if (missing.Count > 0)
        {
            throw new InvalidDataException($"Colonnes manquantes dans le CSV : {string.Join(", ", missing)}");
        }

        var entries = new List<CsvLedgerEntry>();
        while (!parser.EndOfData)
        {
            var fields = parser.ReadFields();
            if (fields is null || fields.All(string.IsNullOrWhiteSpace))
            {
                continue;
            }

            entries.Add(new CsvLedgerEntry(
                Get(fields, headerMap, "n_compte"),
                Get(fields, headerMap, "libelle_du_compte").Trim(),
                TextTools.ParseDate(Get(fields, headerMap, "date")),
                Get(fields, headerMap, "journal").Trim().ToUpperInvariant(),
                Get(fields, headerMap, "n_de_piece").Trim(),
                Get(fields, headerMap, "libelle_mouvement").Trim(),
                TextTools.ParseAmount(Get(fields, headerMap, "montant_debit")),
                TextTools.ParseAmount(Get(fields, headerMap, "montant_credit"))));
        }

        return entries;
    }

    public static LedgerType DetectLedgerType(IReadOnlyList<CsvLedgerEntry> entries)
    {
        var supplierAccounts = 0;
        var customerAccounts = 0;
        var supplierJournals = 0;
        var customerJournals = 0;

        foreach (var entry in entries)
        {
            var account = new string(entry.AccountNumber.Where(char.IsDigit).ToArray());
            if (account.StartsWith("401", StringComparison.Ordinal))
            {
                supplierAccounts++;
            }
            else if (account.StartsWith("411", StringComparison.Ordinal))
            {
                customerAccounts++;
            }

            if (TreatmentCatalog.SupplierDetectionJournals.Contains(entry.Journal))
            {
                supplierJournals++;
            }

            if (TreatmentCatalog.SaleJournals.Contains(entry.Journal))
            {
                customerJournals++;
            }
        }

        if (supplierAccounts > customerAccounts)
        {
            return LedgerType.Supplier;
        }

        if (customerAccounts > supplierAccounts)
        {
            return LedgerType.Customer;
        }

        if (supplierJournals > customerJournals)
        {
            return LedgerType.Supplier;
        }

        if (customerJournals > supplierJournals)
        {
            return LedgerType.Customer;
        }

        throw new InvalidOperationException(
            "Impossible de reconnaître automatiquement le grand livre. Choisissez Fournisseur ou Client dans l'application.");
    }

    private static Dictionary<string, int> BuildHeaderMap(string[] headers)
    {
        var map = new Dictionary<string, int>(StringComparer.OrdinalIgnoreCase);
        for (var index = 0; index < headers.Length; index++)
        {
            var normalized = TextTools.NormalizeHeader(headers[index]);
            if (!map.ContainsKey(normalized))
            {
                map[normalized] = index;
            }
        }

        return map;
    }

    private static string Get(string[] fields, IReadOnlyDictionary<string, int> headerMap, string normalizedHeader)
    {
        if (!headerMap.TryGetValue(normalizedHeader, out var index) || index >= fields.Length)
        {
            return string.Empty;
        }

        return fields[index] ?? string.Empty;
    }
}

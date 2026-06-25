namespace ListeTiime.Core;

public static class TreatmentCatalog
{
    public static readonly HashSet<string> ExcludedSupplierPaymentJournals = new(StringComparer.OrdinalIgnoreCase)
    {
        "AT", "AC", "AO", "OD", "CA", "AN", "RV"
    };

    public static readonly HashSet<string> SaleJournals = new(StringComparer.OrdinalIgnoreCase)
    {
        "VI", "VE", "VT"
    };

    public static readonly HashSet<string> SupplierInvoiceJournals = new(StringComparer.OrdinalIgnoreCase)
    {
        "AC", "AT"
    };

    public static readonly HashSet<string> CustomerReceiptExcludedJournals =
        new(ExcludedSupplierPaymentJournals.Concat(SaleJournals), StringComparer.OrdinalIgnoreCase);

    public static readonly HashSet<string> SupplierDetectionJournals = new(StringComparer.OrdinalIgnoreCase)
    {
        "AC", "AT", "AO"
    };

    public static readonly TreatmentDefinition SupplierPayments = new(
        TreatmentKind.SupplierPayments,
        LedgerType.Supplier,
        TreatmentRole.First,
        "Paiements sans facture",
        "PAIEMENTS SANS FACTURE",
        "Paiements sans facture",
        "total paiements",
        "Paiements",
        "Rmbrsmts");

    public static readonly TreatmentDefinition SupplierInvoices = new(
        TreatmentKind.SupplierInvoices,
        LedgerType.Supplier,
        TreatmentRole.Second,
        "Factures sans paiements",
        "FACTURES SANS PAIEMENTS",
        "Factures sans paiements",
        "total factures",
        "Factures",
        "Avoirs");

    public static readonly TreatmentDefinition CustomerReceipts = new(
        TreatmentKind.CustomerReceipts,
        LedgerType.Customer,
        TreatmentRole.First,
        "Encaissements sans facture de ventes",
        "ENCAISSEMENTS SANS FACTURE DE VENTES",
        "Encaissements sans facture",
        "total encaissements",
        "Encaissements",
        "Remboursements");

    public static readonly TreatmentDefinition CustomerInvoices = new(
        TreatmentKind.CustomerInvoices,
        LedgerType.Customer,
        TreatmentRole.Second,
        "Factures de ventes sans paiement",
        "FACTURES DE VENTES SANS PAIEMENT",
        "Factures ventes sans paiement",
        "total factures",
        "Factures",
        "Avoirs");

    public static IReadOnlyList<TreatmentDefinition> For(LedgerType ledgerType, TreatmentRole role)
    {
        return role switch
        {
            TreatmentRole.Both => [Resolve(ledgerType, TreatmentRole.First), Resolve(ledgerType, TreatmentRole.Second)],
            TreatmentRole.First => [Resolve(ledgerType, TreatmentRole.First)],
            TreatmentRole.Second => [Resolve(ledgerType, TreatmentRole.Second)],
            _ => throw new ArgumentOutOfRangeException(nameof(role), role, null)
        };
    }

    public static TreatmentDefinition Resolve(LedgerType ledgerType, TreatmentRole role)
    {
        return (ledgerType, role) switch
        {
            (LedgerType.Supplier, TreatmentRole.First) => SupplierPayments,
            (LedgerType.Supplier, TreatmentRole.Second) => SupplierInvoices,
            (LedgerType.Customer, TreatmentRole.First) => CustomerReceipts,
            (LedgerType.Customer, TreatmentRole.Second) => CustomerInvoices,
            _ => throw new InvalidOperationException("Traitement introuvable.")
        };
    }

    public static IReadOnlyList<ExportColumn> BuildColumns(TreatmentDefinition treatment, IReadOnlyList<ProcessedRow> rows)
    {
        var columns = new List<ExportColumn>
        {
            new("Libellé du compte", ExportColumnKind.AccountLabel),
            new("Date", ExportColumnKind.Date),
            new("Journal", ExportColumnKind.Journal),
            new("Libellé mouvement", ExportColumnKind.Movement),
            new(treatment.PrimaryColumn, ExportColumnKind.PrimaryAmount)
        };

        if (rows.Any(row => row.SecondaryAmount is not null and not 0m))
        {
            columns.Add(new ExportColumn(treatment.SecondaryColumn, ExportColumnKind.SecondaryAmount));
        }

        columns.Add(new ExportColumn("Remarque", ExportColumnKind.Remark));
        return columns;
    }
}

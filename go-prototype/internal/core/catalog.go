package core

var excludedSupplierPaymentJournals = stringSet("AT", "AC", "AO", "OD", "CA", "AN", "RV")
var saleJournals = stringSet("VI", "VE", "VT")
var supplierInvoiceJournals = stringSet("AC", "AT")
var supplierDetectionJournals = stringSet("AC", "AT", "AO")

var customerReceiptExcludedJournals = func() map[string]struct{} {
	result := stringSet("AT", "AC", "AO", "OD", "CA", "AN", "RV")
	for journal := range saleJournals {
		result[journal] = struct{}{}
	}
	return result
}()

var treatmentSupplierPayments = TreatmentDefinition{
	Kind:            SupplierPayments,
	LedgerType:      LedgerSupplier,
	Role:            RoleFirst,
	Label:           "Paiements sans facture",
	OutputLabel:     "PAIEMENTS SANS FACTURE",
	SheetName:       "Paiements sans facture",
	AmountLabel:     "total paiements",
	PrimaryColumn:   "Paiements",
	SecondaryColumn: "Rmbrsmts",
}

var treatmentSupplierInvoices = TreatmentDefinition{
	Kind:            SupplierInvoices,
	LedgerType:      LedgerSupplier,
	Role:            RoleSecond,
	Label:           "Factures sans paiements",
	OutputLabel:     "FACTURES SANS PAIEMENTS",
	SheetName:       "Factures sans paiements",
	AmountLabel:     "total factures",
	PrimaryColumn:   "Factures",
	SecondaryColumn: "Avoirs",
}

var treatmentCustomerReceipts = TreatmentDefinition{
	Kind:            CustomerReceipts,
	LedgerType:      LedgerCustomer,
	Role:            RoleFirst,
	Label:           "Encaissements sans facture de ventes",
	OutputLabel:     "ENCAISSEMENTS SANS FACTURE DE VENTES",
	SheetName:       "Encaissements sans facture",
	AmountLabel:     "total encaissements",
	PrimaryColumn:   "Encaissements",
	SecondaryColumn: "Remboursements",
}

var treatmentCustomerInvoices = TreatmentDefinition{
	Kind:            CustomerInvoices,
	LedgerType:      LedgerCustomer,
	Role:            RoleSecond,
	Label:           "Factures de ventes sans paiement",
	OutputLabel:     "FACTURES DE VENTES SANS PAIEMENT",
	SheetName:       "Factures ventes sans paiement",
	AmountLabel:     "total factures",
	PrimaryColumn:   "Factures",
	SecondaryColumn: "Avoirs",
}

func stringSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func hasValue(set map[string]struct{}, value string) bool {
	_, found := set[value]
	return found
}

func treatmentsFor(ledgerType LedgerType, role TreatmentRole) []TreatmentDefinition {
	switch role {
	case RoleBoth:
		return []TreatmentDefinition{
			resolveTreatment(ledgerType, RoleFirst),
			resolveTreatment(ledgerType, RoleSecond),
		}
	case RoleFirst, RoleSecond:
		return []TreatmentDefinition{resolveTreatment(ledgerType, role)}
	default:
		panic("rôle de traitement invalide")
	}
}

func resolveTreatment(ledgerType LedgerType, role TreatmentRole) TreatmentDefinition {
	switch {
	case ledgerType == LedgerSupplier && role == RoleFirst:
		return treatmentSupplierPayments
	case ledgerType == LedgerSupplier && role == RoleSecond:
		return treatmentSupplierInvoices
	case ledgerType == LedgerCustomer && role == RoleFirst:
		return treatmentCustomerReceipts
	case ledgerType == LedgerCustomer && role == RoleSecond:
		return treatmentCustomerInvoices
	default:
		panic("traitement introuvable")
	}
}

func buildColumns(treatment TreatmentDefinition, rows []ProcessedRow) []ExportColumn {
	columns := []ExportColumn{
		{Header: "Libellé du compte", Kind: ColumnAccountLabel},
		{Header: "Date", Kind: ColumnDate},
		{Header: "Journal", Kind: ColumnJournal},
		{Header: "Libellé mouvement", Kind: ColumnMovement},
		{Header: treatment.PrimaryColumn, Kind: ColumnPrimaryAmount},
	}

	for _, row := range rows {
		if row.SecondaryAmount != nil && !row.SecondaryAmount.IsZero() {
			columns = append(columns, ExportColumn{Header: treatment.SecondaryColumn, Kind: ColumnSecondaryAmount})
			break
		}
	}
	columns = append(columns, ExportColumn{Header: "Remarque", Kind: ColumnRemark})
	return columns
}

package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type DatasetExporter interface {
	Save(dataset ExportDataset, outputPath string) error
}

type Processor struct {
	ExcelExporter DatasetExporter
	PDFExporter   DatasetExporter
}

func NewProcessor(excelExporter, pdfExporter DatasetExporter) *Processor {
	return &Processor{ExcelExporter: excelExporter, PDFExporter: pdfExporter}
}

func (p *Processor) ProcessFiles(paths []string, options ProcessingOptions) (BatchResult, error) {
	if options.Formats == OutputNone {
		return BatchResult{}, fmt.Errorf("sélectionnez au moins un format")
	}

	result := BatchResult{}
	var failures []string
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		documents, err := p.ProcessFile(path, options)
		if err != nil {
			failures = append(failures, filepath.Base(path)+" : "+err.Error())
			continue
		}
		result.Documents = append(result.Documents, documents...)
	}
	if len(failures) > 0 {
		return result, errors.New(strings.Join(failures, "\n"))
	}
	return result, nil
}

func (p *Processor) ProcessFile(path string, options ProcessingOptions) ([]GeneratedDocument, error) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("fichier introuvable")
	}
	if !strings.EqualFold(filepath.Ext(path), ".csv") {
		return nil, fmt.Errorf("le fichier n'est pas un CSV")
	}

	metadata := TryParseMetadata(path)
	if metadata == nil && options.MetadataProvider != nil {
		metadata, err = options.MetadataProvider(path)
		if err != nil {
			return nil, err
		}
	}
	if metadata == nil {
		return nil, fmt.Errorf(
			"nom de fichier non standard. Format attendu : grand_livre_2025-01-01_2025-12-31_nom_client.csv",
		)
	}

	entries, err := ReadLedger(path)
	if err != nil {
		return nil, err
	}

	var ledgerType LedgerType
	switch options.LedgerMode {
	case LedgerModeSupplier:
		ledgerType = LedgerSupplier
	case LedgerModeCustomer:
		ledgerType = LedgerCustomer
	default:
		ledgerType, err = DetectLedgerType(entries)
		if err != nil {
			return nil, err
		}
	}

	documents := make([]GeneratedDocument, 0, 2)
	for _, treatment := range treatmentsFor(ledgerType, options.Role) {
		document, err := p.processTreatment(path, *metadata, entries, treatment, options.Formats)
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, nil
}

func (p *Processor) processTreatment(
	csvPath string,
	metadata FileMetadata,
	entries []LedgerEntry,
	treatment TreatmentDefinition,
	formats OutputFormats,
) (GeneratedDocument, error) {
	rows := applyTreatment(entries, treatment)
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]
		if comparison := strings.Compare(strings.ToUpper(left.AccountLabel), strings.ToUpper(right.AccountLabel)); comparison != 0 {
			return comparison < 0
		}
		if comparison := compareDates(left.Date, right.Date); comparison != 0 {
			return comparison < 0
		}
		if comparison := strings.Compare(left.Journal, right.Journal); comparison != 0 {
			return comparison < 0
		}
		if comparison := compareOptionalDecimals(left.PrimaryAmount, right.PrimaryAmount); comparison != 0 {
			return comparison < 0
		}
		return compareOptionalDecimals(left.SecondaryAmount, right.SecondaryAmount) < 0
	})

	if len(rows) == 0 {
		return GeneratedDocument{
			SourcePath: csvPath,
			Treatment:  treatment,
			Total:      decimal.Zero,
		}, nil
	}

	dataset := ExportDataset{
		Treatment: treatment,
		Rows:      rows,
		Columns:   buildColumns(treatment, rows),
	}
	stem := sanitizeFileName(metadata.Client + " - " + treatment.OutputLabel + " " + metadata.Year)
	excelPath, pdfPath, err := reserveOutputPaths(filepath.Dir(csvPath), stem, formats)
	if err != nil {
		return GeneratedDocument{}, err
	}

	if excelPath != "" {
		if p.ExcelExporter == nil {
			return GeneratedDocument{}, fmt.Errorf("exporteur Excel indisponible")
		}
		if err := p.ExcelExporter.Save(dataset, excelPath); err != nil {
			return GeneratedDocument{}, err
		}
	}
	if pdfPath != "" {
		if p.PDFExporter == nil {
			return GeneratedDocument{}, fmt.Errorf("exporteur PDF indisponible")
		}
		if err := p.PDFExporter.Save(dataset, pdfPath); err != nil {
			return GeneratedDocument{}, err
		}
	}

	return GeneratedDocument{
		SourcePath: csvPath,
		Treatment:  treatment,
		RowCount:   len(rows),
		Total:      dataset.TotalPrimary(),
		ExcelPath:  excelPath,
		PDFPath:    pdfPath,
	}, nil
}

func applyTreatment(entries []LedgerEntry, treatment TreatmentDefinition) []ProcessedRow {
	rows := make([]ProcessedRow, 0, len(entries))
	for _, entry := range entries {
		var row *ProcessedRow
		switch treatment.Kind {
		case SupplierPayments:
			row = buildSupplierPayment(entry)
		case SupplierInvoices:
			row = buildSupplierInvoice(entry)
		case CustomerReceipts:
			row = buildCustomerReceipt(entry)
		case CustomerInvoices:
			row = buildCustomerInvoice(entry)
		}
		if row != nil {
			rows = append(rows, *row)
		}
	}
	return rows
}

func buildSupplierPayment(entry LedgerEntry) *ProcessedRow {
	if entry.PieceNumber != "" || hasValue(excludedSupplierPaymentJournals, entry.Journal) {
		return nil
	}
	remark := "Manque avoir"
	if entry.Debit.GreaterThan(decimal.Zero) {
		remark = "Manque facture"
	}
	return &ProcessedRow{
		AccountLabel:    entry.AccountLabel,
		Date:            entry.Date,
		Journal:         entry.Journal,
		MovementLabel:   entry.MovementLabel,
		PrimaryAmount:   NullIfZero(entry.Debit),
		SecondaryAmount: NullIfZero(entry.Credit),
		Remark:          remark,
	}
}

func buildSupplierInvoice(entry LedgerEntry) *ProcessedRow {
	if !hasValue(supplierInvoiceJournals, entry.Journal) {
		return nil
	}
	remark := "Avoir sans rmbrsmt"
	if entry.Credit.GreaterThan(decimal.Zero) {
		remark = "Facture sans paiement"
	}
	return &ProcessedRow{
		AccountLabel:    entry.AccountLabel,
		Date:            entry.Date,
		Journal:         entry.Journal,
		MovementLabel:   entry.MovementLabel,
		PrimaryAmount:   NullIfZero(entry.Credit),
		SecondaryAmount: NullIfZero(entry.Debit),
		Remark:          remark,
	}
}

func buildCustomerReceipt(entry LedgerEntry) *ProcessedRow {
	if entry.PieceNumber != "" || hasValue(customerReceiptExcludedJournals, entry.Journal) {
		return nil
	}
	remark := "Manque avoir de vente"
	if entry.Credit.GreaterThan(decimal.Zero) {
		remark = "Manque facture de vente"
	}
	return &ProcessedRow{
		AccountLabel:    entry.AccountLabel,
		Date:            entry.Date,
		Journal:         entry.Journal,
		MovementLabel:   entry.MovementLabel,
		PrimaryAmount:   NullIfZero(entry.Credit),
		SecondaryAmount: NullIfZero(entry.Debit),
		Remark:          remark,
	}
}

func buildCustomerInvoice(entry LedgerEntry) *ProcessedRow {
	if !hasValue(saleJournals, entry.Journal) {
		return nil
	}
	remark := "Avoir de vente sans rmbrsmt"
	if entry.Debit.GreaterThan(decimal.Zero) {
		remark = "Facture de vente sans paiement"
	}
	return &ProcessedRow{
		AccountLabel:    entry.AccountLabel,
		Date:            entry.Date,
		Journal:         entry.Journal,
		MovementLabel:   entry.MovementLabel,
		PrimaryAmount:   NullIfZero(entry.Debit),
		SecondaryAmount: NullIfZero(entry.Credit),
		Remark:          remark,
	}
}

func reserveOutputPaths(directory, stem string, formats OutputFormats) (string, string, error) {
	for suffix := 1; suffix < 10_000; suffix++ {
		displaySuffix := ""
		if suffix > 1 {
			displaySuffix = fmt.Sprintf(" - %d", suffix)
		}
		excelPath := ""
		pdfPath := ""
		if formats.Has(OutputExcel) {
			excelPath = filepath.Join(directory, stem+displaySuffix+".xlsx")
		}
		if formats.Has(OutputPDF) {
			pdfPath = filepath.Join(directory, stem+displaySuffix+".pdf")
		}
		if (excelPath == "" || !fileExists(excelPath)) && (pdfPath == "" || !fileExists(pdfPath)) {
			return excelPath, pdfPath, nil
		}
	}
	return "", "", fmt.Errorf("impossible de trouver un nom de fichier disponible")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func compareDates(left, right *time.Time) int {
	switch {
	case left == nil && right == nil:
		return 0
	case left == nil:
		return 1
	case right == nil:
		return -1
	case left.Before(*right):
		return -1
	case left.After(*right):
		return 1
	default:
		return 0
	}
}

func compareOptionalDecimals(left, right *decimal.Decimal) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return decimal.Zero.Cmp(*right)
	}
	if right == nil {
		return left.Cmp(decimal.Zero)
	}
	return left.Cmp(*right)
}

func BuildSummary(result BatchResult) string {
	var created []GeneratedDocument
	var skipped []GeneratedDocument
	fileCount := 0
	for _, document := range result.Documents {
		if document.HasOutput() {
			created = append(created, document)
			if document.ExcelPath != "" {
				fileCount++
			}
			if document.PDFPath != "" {
				fileCount++
			}
		} else {
			skipped = append(skipped, document)
		}
	}

	if len(created) == 0 {
		if len(skipped) == 0 {
			return "Aucun fichier généré."
		}
		return "Aucun fichier créé : aucune ligne trouvée."
	}

	title := "Fichiers générés :"
	if fileCount == 1 {
		title = "Fichier généré :"
	}
	lines := []string{title}
	for _, document := range created {
		lines = append(lines, fmt.Sprintf(
			"- %s : %d lignes, %s : %s €",
			document.Treatment.Label,
			document.RowCount,
			document.Treatment.AmountLabel,
			FormatAmount(document.Total),
		))
		if document.ExcelPath != "" {
			lines = append(lines, "  Excel : "+document.ExcelPath)
		}
		if document.PDFPath != "" {
			lines = append(lines, "  PDF : "+document.PDFPath)
		}
	}
	if len(skipped) > 0 {
		lines = append(lines, "", "Aucun fichier créé pour :")
		for _, document := range skipped {
			lines = append(lines, "- "+document.Treatment.Label+" : aucune ligne trouvée")
		}
	}
	return strings.Join(lines, "\n")
}

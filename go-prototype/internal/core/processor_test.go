package core

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func TestSupplierPaymentsGeneratesExcelAndPDFWithoutEmptyRefundColumn(t *testing.T) {
	directory := t.TempDir()
	csvPath := writeTestCSV(t, directory,
		"grand_livre_2025-01-01_2025-12-31_jennah_boutique.csv",
		[][]string{
			testRow("401000", "BRINKS", "19/09/2025", "BQ", "", "18/09/25 18H56 PAIEMENT SERV DRA", "250,00", "0"),
			testRow("401000", "BRINKS", "19/09/2025", "BQ", "", "18/09/25 18H57 PAIEMENT SERV DRA", "250,00", "0"),
			testRow("401001", "MS STYLE", "11/09/2025", "BNP", "", "VIR SCT INST EMIS MOTIF 20250907 BEN SAS MS SERR", "840,00", "0"),
			testRow("401002", "FOURNISSEUR AO", "12/09/2025", "AO", "", "A exclure du traitement paiements", "999,00", "0"),
			testRow("401003", "FACTURE", "13/09/2025", "AC", "", "A exclure du traitement paiements", "0", "100,00"),
		},
	)

	result, err := testProcessor().ProcessFiles([]string{csvPath}, ProcessingOptions{
		Formats:    OutputExcel | OutputPDF,
		LedgerMode: LedgerAuto,
		Role:       RoleFirst,
	})
	if err != nil {
		t.Fatal(err)
	}

	document := singleDocument(t, result)
	if !document.HasOutput() || document.RowCount != 3 || !document.Total.Equal(decimalFromString(t, "1340")) {
		t.Fatalf("résultat inattendu : %+v", document)
	}
	if err := validatePDF(document.PDFPath); err != nil {
		t.Fatal(err)
	}

	workbook := openWorkbook(t, document.ExcelPath)
	defer workbook.Close()
	sheet := workbook.GetSheetName(0)
	headers := workbookHeaders(t, workbook, sheet)
	if !slices.Equal(headers, []string{"Libellé du compte", "Date", "Journal", "Libellé mouvement", "Paiements", "Remarque"}) {
		t.Fatalf("en-têtes inattendus : %v", headers)
	}
	if contains(headers, "Rmbrsmts") {
		t.Fatal("la colonne Rmbrsmts ne devait pas être présente")
	}
	width, err := workbook.GetColWidth(sheet, "D")
	if err != nil || int(width+0.5) != 76 {
		t.Fatalf("largeur D inattendue : %f (%v)", width, err)
	}
	height, err := workbook.GetRowHeight(sheet, 2)
	if err != nil || height < 22 {
		t.Fatalf("hauteur inattendue : %f (%v)", height, err)
	}
	remark, _ := workbook.GetCellValue(sheet, "F2")
	if remark != "Manque facture" {
		t.Fatalf("remarque inattendue : %s", remark)
	}
	layout, err := workbook.GetPageLayout(sheet)
	if err != nil || layout.Orientation == nil || *layout.Orientation != "portrait" ||
		layout.FitToWidth == nil || *layout.FitToWidth != 1 {
		t.Fatalf("mise en page inattendue : %+v (%v)", layout, err)
	}
	footer, err := workbook.GetHeaderFooter(sheet)
	if err != nil || footer == nil || footer.OddFooter != "&C&P/&N" {
		t.Fatalf("pied de page inattendu : %+v (%v)", footer, err)
	}
}

func TestSupplierInvoicesHidesEmptyAvoirsColumn(t *testing.T) {
	directory := t.TempDir()
	csvPath := writeTestCSV(t, directory,
		"grand_livre_2025-01-01_2025-12-31_jennah_boutique.csv",
		[][]string{
			testRow("401000", "FOURNISSEUR", "19/09/2025", "AC", "FAC001", "FACTURE ACHAT", "0", "1000,00"),
		},
	)
	result, err := testProcessor().ProcessFiles([]string{csvPath}, ProcessingOptions{
		Formats: OutputExcel, LedgerMode: LedgerAuto, Role: RoleSecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	document := singleDocument(t, result)
	workbook := openWorkbook(t, document.ExcelPath)
	defer workbook.Close()
	sheet := workbook.GetSheetName(0)
	headers := workbookHeaders(t, workbook, sheet)
	expected := []string{"Libellé du compte", "Date", "Journal", "Libellé mouvement", "Factures", "Remarque"}
	if !slices.Equal(headers, expected) {
		t.Fatalf("en-têtes inattendus : %v", headers)
	}
	remark, _ := workbook.GetCellValue(sheet, "F2")
	if remark != "Facture sans paiement" {
		t.Fatalf("remarque inattendue : %s", remark)
	}
}

func TestSupplierInvoicesKeepsAvoirsColumnAndRemark(t *testing.T) {
	directory := t.TempDir()
	csvPath := writeTestCSV(t, directory,
		"grand_livre_2025-01-01_2025-12-31_jennah_boutique.csv",
		[][]string{
			testRow("401000", "FOURNISSEUR", "19/09/2025", "AC", "FAC001", "FACTURE ACHAT", "0", "1000,00"),
			testRow("401000", "FOURNISSEUR", "20/09/2025", "AT", "AV001", "AVOIR ACHAT", "120,00", "0"),
		},
	)
	result, err := testProcessor().ProcessFiles([]string{csvPath}, ProcessingOptions{
		Formats: OutputExcel, LedgerMode: LedgerAuto, Role: RoleSecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	document := singleDocument(t, result)
	workbook := openWorkbook(t, document.ExcelPath)
	defer workbook.Close()
	sheet := workbook.GetSheetName(0)
	if !contains(workbookHeaders(t, workbook, sheet), "Avoirs") {
		t.Fatal("la colonne Avoirs est absente")
	}
	remarks := columnValues(t, workbook, sheet, "G", 2, 3)
	if !contains(remarks, "Avoir sans rmbrsmt") || !contains(remarks, "Facture sans paiement") {
		t.Fatalf("remarques inattendues : %v", remarks)
	}
}

func TestCustomerLedgerAutoDetectsAndGeneratesBothLists(t *testing.T) {
	directory := t.TempDir()
	csvPath := writeTestCSV(t, directory,
		"grand_livre_2025-01-01_2025-12-31_saint_ouen_auto_services.csv",
		[][]string{
			testRow("411000", "CLIENT A", "02/01/2025", "BQ", "", "VIREMENT CLIENT A", "0", "600,00"),
			testRow("411000", "CLIENT A", "03/01/2025", "VI", "FV001", "FACTURE VENTE", "1000,00", "0"),
			testRow("411001", "CLIENT B", "04/01/2025", "VE", "FV002", "FACTURE VENTE", "500,00", "0"),
			testRow("411001", "CLIENT B", "05/01/2025", "BQ", "LET001", "ENCAISSEMENT LETTRE", "0", "50,00"),
		},
	)
	result, err := testProcessor().ProcessFiles([]string{csvPath}, ProcessingOptions{
		Formats: OutputExcel, LedgerMode: LedgerAuto, Role: RoleBoth,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Documents) != 2 {
		t.Fatalf("nombre de documents inattendu : %d", len(result.Documents))
	}
	receipts := documentByKind(t, result, CustomerReceipts)
	invoices := documentByKind(t, result, CustomerInvoices)
	if receipts.RowCount != 1 || !receipts.Total.Equal(decimalFromString(t, "600")) {
		t.Fatalf("encaissements inattendus : %+v", receipts)
	}
	if invoices.RowCount != 2 || !invoices.Total.Equal(decimalFromString(t, "1500")) {
		t.Fatalf("factures inattendues : %+v", invoices)
	}

	receiptsWorkbook := openWorkbook(t, receipts.ExcelPath)
	receiptsHeaders := workbookHeaders(t, receiptsWorkbook, receiptsWorkbook.GetSheetName(0))
	receiptsWorkbook.Close()
	if !contains(receiptsHeaders, "Encaissements") || contains(receiptsHeaders, "Remboursements") {
		t.Fatalf("colonnes encaissements inattendues : %v", receiptsHeaders)
	}

	invoicesWorkbook := openWorkbook(t, invoices.ExcelPath)
	invoicesHeaders := workbookHeaders(t, invoicesWorkbook, invoicesWorkbook.GetSheetName(0))
	invoicesWorkbook.Close()
	if !contains(invoicesHeaders, "Factures") || contains(invoicesHeaders, "Avoirs") {
		t.Fatalf("colonnes factures inattendues : %v", invoicesHeaders)
	}
}

func TestCustomerLedgerKeepsRefundAndAvoirColumns(t *testing.T) {
	directory := t.TempDir()
	csvPath := writeTestCSV(t, directory,
		"grand_livre_2025-01-01_2025-12-31_saint_ouen_auto_services.csv",
		[][]string{
			testRow("411000", "CLIENT A", "02/01/2025", "BQ", "", "REMBOURSEMENT CLIENT A", "20,00", "0"),
			testRow("411000", "CLIENT A", "03/01/2025", "VT", "AVV001", "AVOIR VENTE", "0", "30,00"),
		},
	)
	result, err := testProcessor().ProcessFiles([]string{csvPath}, ProcessingOptions{
		Formats: OutputExcel, LedgerMode: LedgerModeCustomer, Role: RoleBoth,
	})
	if err != nil {
		t.Fatal(err)
	}
	receipts := documentByKind(t, result, CustomerReceipts)
	invoices := documentByKind(t, result, CustomerInvoices)

	receiptsWorkbook := openWorkbook(t, receipts.ExcelPath)
	receiptsSheet := receiptsWorkbook.GetSheetName(0)
	if !contains(workbookHeaders(t, receiptsWorkbook, receiptsSheet), "Remboursements") {
		t.Fatal("la colonne Remboursements est absente")
	}
	receiptRemark, _ := receiptsWorkbook.GetCellValue(receiptsSheet, "G2")
	receiptsWorkbook.Close()
	if receiptRemark != "Manque avoir de vente" {
		t.Fatalf("remarque remboursement inattendue : %s", receiptRemark)
	}

	invoicesWorkbook := openWorkbook(t, invoices.ExcelPath)
	invoicesSheet := invoicesWorkbook.GetSheetName(0)
	if !contains(workbookHeaders(t, invoicesWorkbook, invoicesSheet), "Avoirs") {
		t.Fatal("la colonne Avoirs est absente")
	}
	invoiceRemark, _ := invoicesWorkbook.GetCellValue(invoicesSheet, "G2")
	invoicesWorkbook.Close()
	if invoiceRemark != "Avoir de vente sans rmbrsmt" {
		t.Fatalf("remarque avoir inattendue : %s", invoiceRemark)
	}
}

func TestEmptyTreatmentDoesNotCreateFile(t *testing.T) {
	directory := t.TempDir()
	csvPath := writeTestCSV(t, directory,
		"grand_livre_2025-01-01_2025-12-31_jennah_boutique.csv",
		[][]string{
			testRow("401000", "BRINKS", "19/09/2025", "BQ", "", "PAIEMENT", "250,00", "0"),
		},
	)
	result, err := testProcessor().ProcessFiles([]string{csvPath}, ProcessingOptions{
		Formats: OutputExcel | OutputPDF, LedgerMode: LedgerAuto, Role: RoleSecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	document := singleDocument(t, result)
	if document.HasOutput() || document.ExcelPath != "" || document.PDFPath != "" {
		t.Fatalf("un fichier vide a été généré : %+v", document)
	}
	files, _ := filepath.Glob(filepath.Join(directory, "*.*"))
	for _, path := range files {
		if filepath.Ext(path) == ".xlsx" || filepath.Ext(path) == ".pdf" {
			t.Fatalf("fichier inattendu : %s", path)
		}
	}
}

func TestSettingsStoreSavesAndLoadsFormatsAndMode(t *testing.T) {
	store := NewSettingsStore(t.TempDir())
	settings := UserSettings{}
	settings.SetFormats(OutputPDF)
	settings.SetLedgerMode(LedgerModeCustomer)
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}

	loaded := store.Load(OutputExcel, LedgerAuto, false)
	if loaded.Excel || !loaded.PDF || loaded.ParsedLedgerMode() != LedgerModeCustomer {
		t.Fatalf("réglages inattendus : %+v", loaded)
	}
}

func TestSettingsStoreDirectModeFallsBackToExcel(t *testing.T) {
	store := NewSettingsStore(t.TempDir())
	settings := UserSettings{}
	settings.SetFormats(OutputNone)
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}

	loaded := store.Load(OutputExcel|OutputPDF, LedgerAuto, true)
	if !loaded.Excel || loaded.PDF {
		t.Fatalf("repli inattendu : %+v", loaded)
	}
}

func testProcessor() *Processor {
	return NewProcessor(NewExcelExporter(), NewPDFExporter())
}

func writeTestCSV(t *testing.T, directory, name string, rows [][]string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(transform.NewWriter(file, charmap.Windows1252.NewEncoder()))
	writer.Comma = ';'
	if err := writer.Write([]string{
		"N° Compte", "Libellé du compte", "Date", "Journal", "N° de pièce",
		"Libellé mouvement", "Montant Débit", "Montant Crédit",
	}); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			t.Fatal(err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatal(err)
	}
	return path
}

func testRow(account, label, date, journal, piece, movement, debit, credit string) []string {
	return []string{account, label, date, journal, piece, movement, debit, credit}
}

func singleDocument(t *testing.T, result BatchResult) GeneratedDocument {
	t.Helper()
	if len(result.Documents) != 1 {
		t.Fatalf("nombre de documents inattendu : %d", len(result.Documents))
	}
	return result.Documents[0]
}

func documentByKind(t *testing.T, result BatchResult, kind TreatmentKind) GeneratedDocument {
	t.Helper()
	for _, document := range result.Documents {
		if document.Treatment.Kind == kind {
			return document
		}
	}
	t.Fatalf("traitement introuvable : %d", kind)
	return GeneratedDocument{}
}

func decimalFromString(t *testing.T, value string) decimal.Decimal {
	t.Helper()
	amount, err := decimal.NewFromString(value)
	if err != nil {
		t.Fatal(err)
	}
	return amount
}

func openWorkbook(t *testing.T, path string) *excelize.File {
	t.Helper()
	workbook, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return workbook
}

func workbookHeaders(t *testing.T, workbook *excelize.File, sheet string) []string {
	t.Helper()
	rows, err := workbook.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		t.Fatalf("lecture des en-têtes : %v", err)
	}
	return rows[0]
}

func columnValues(t *testing.T, workbook *excelize.File, sheet, column string, first, last int) []string {
	t.Helper()
	values := make([]string, 0, last-first+1)
	for row := first; row <= last; row++ {
		value, err := workbook.GetCellValue(sheet, column+strconv.Itoa(row))
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	return values
}

func contains(values []string, expected string) bool {
	return slices.Contains(values, expected)
}

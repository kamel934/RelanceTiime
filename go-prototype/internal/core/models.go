package core

import (
	"time"

	"github.com/shopspring/decimal"
)

type OutputFormats uint8

const (
	OutputNone  OutputFormats = 0
	OutputExcel OutputFormats = 1 << iota
	OutputPDF
)

func (f OutputFormats) Has(value OutputFormats) bool {
	return f&value != 0
}

type LedgerType int

const (
	LedgerSupplier LedgerType = iota
	LedgerCustomer
)

type LedgerMode int

const (
	LedgerAuto LedgerMode = iota
	LedgerModeSupplier
	LedgerModeCustomer
)

type TreatmentRole int

const (
	RoleFirst TreatmentRole = iota
	RoleSecond
	RoleBoth
)

type TreatmentKind int

const (
	SupplierPayments TreatmentKind = iota
	SupplierInvoices
	CustomerReceipts
	CustomerInvoices
)

type ColumnKind int

const (
	ColumnAccountLabel ColumnKind = iota
	ColumnDate
	ColumnJournal
	ColumnMovement
	ColumnPrimaryAmount
	ColumnSecondaryAmount
	ColumnRemark
)

type FileMetadata struct {
	Client string
	Year   string
}

type LedgerEntry struct {
	AccountNumber string
	AccountLabel  string
	Date          *time.Time
	Journal       string
	PieceNumber   string
	MovementLabel string
	Debit         decimal.Decimal
	Credit        decimal.Decimal
}

type TreatmentDefinition struct {
	Kind            TreatmentKind
	LedgerType      LedgerType
	Role            TreatmentRole
	Label           string
	OutputLabel     string
	SheetName       string
	AmountLabel     string
	PrimaryColumn   string
	SecondaryColumn string
}

type ProcessedRow struct {
	AccountLabel    string
	Date            *time.Time
	Journal         string
	MovementLabel   string
	PrimaryAmount   *decimal.Decimal
	SecondaryAmount *decimal.Decimal
	Remark          string
}

type ExportColumn struct {
	Header string
	Kind   ColumnKind
}

type ExportDataset struct {
	Treatment TreatmentDefinition
	Rows      []ProcessedRow
	Columns   []ExportColumn
}

func (d ExportDataset) TotalPrimary() decimal.Decimal {
	total := decimal.Zero
	for _, row := range d.Rows {
		if row.PrimaryAmount != nil {
			total = total.Add(*row.PrimaryAmount)
		}
	}
	return total
}

type GeneratedDocument struct {
	SourcePath string
	Treatment  TreatmentDefinition
	RowCount   int
	Total      decimal.Decimal
	ExcelPath  string
	PDFPath    string
}

func (d GeneratedDocument) HasOutput() bool {
	return d.ExcelPath != "" || d.PDFPath != ""
}

type BatchResult struct {
	Documents []GeneratedDocument
}

func (r BatchResult) HasOutput() bool {
	for _, document := range r.Documents {
		if document.HasOutput() {
			return true
		}
	}
	return false
}

type MetadataProvider func(path string) (*FileMetadata, error)

type ProcessingOptions struct {
	Formats          OutputFormats
	LedgerMode       LedgerMode
	Role             TreatmentRole
	MetadataProvider MetadataProvider
}

const (
	AppName              = "Liste Tiime Go"
	SettingsDirectory    = "RelanceTiime"
	SettingsFile         = "settings.json"
	DefaultFontName      = "Segoe UI"
	DataFontSize         = 11.0
	HeaderFontSize       = 11.0
	MovementCharsPerLine = 64
	BaseRowHeight        = 22.0
	WrappedLineHeight    = 14.0
	RowHeightPadding     = 4.0
	MaxRowHeight         = 70.0
	HeaderBlue           = "1F4E78"
	BorderGray           = "808080"
)

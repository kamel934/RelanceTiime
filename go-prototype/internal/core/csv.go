package core

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

var requiredHeaders = []string{
	"libelle_du_compte",
	"date",
	"journal",
	"n_de_piece",
	"libelle_mouvement",
	"montant_debit",
	"montant_credit",
}

func ReadLedger(path string) ([]LedgerEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(transform.NewReader(file, charmap.Windows1252.NewDecoder()))
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = false

	headers, err := reader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("le CSV est vide")
	}
	if err != nil {
		return nil, fmt.Errorf("lecture des en-têtes CSV : %w", err)
	}

	headerMap := make(map[string]int, len(headers))
	for index, header := range headers {
		normalized := NormalizeHeader(header)
		if _, exists := headerMap[normalized]; !exists {
			headerMap[normalized] = index
		}
	}
	var missing []string
	for _, required := range requiredHeaders {
		if _, found := headerMap[required]; !found {
			missing = append(missing, required)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("colonnes manquantes dans le CSV : %s", strings.Join(missing, ", "))
	}

	var entries []LedgerEntry
	for {
		fields, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("lecture du CSV : %w", readErr)
		}
		if allEmpty(fields) {
			continue
		}

		date, err := ParseDate(getField(fields, headerMap, "date"))
		if err != nil {
			return nil, err
		}
		debit, err := ParseAmount(getField(fields, headerMap, "montant_debit"))
		if err != nil {
			return nil, err
		}
		credit, err := ParseAmount(getField(fields, headerMap, "montant_credit"))
		if err != nil {
			return nil, err
		}

		entries = append(entries, LedgerEntry{
			AccountNumber: getField(fields, headerMap, "n_compte"),
			AccountLabel:  strings.TrimSpace(getField(fields, headerMap, "libelle_du_compte")),
			Date:          date,
			Journal:       strings.ToUpper(strings.TrimSpace(getField(fields, headerMap, "journal"))),
			PieceNumber:   strings.TrimSpace(getField(fields, headerMap, "n_de_piece")),
			MovementLabel: strings.TrimSpace(getField(fields, headerMap, "libelle_mouvement")),
			Debit:         debit,
			Credit:        credit,
		})
	}
	return entries, nil
}

func DetectLedgerType(entries []LedgerEntry) (LedgerType, error) {
	supplierAccounts := 0
	customerAccounts := 0
	supplierJournals := 0
	customerJournals := 0

	for _, entry := range entries {
		var digits strings.Builder
		for _, character := range entry.AccountNumber {
			if character >= '0' && character <= '9' {
				digits.WriteRune(character)
			}
		}
		account := digits.String()
		switch {
		case strings.HasPrefix(account, "401"):
			supplierAccounts++
		case strings.HasPrefix(account, "411"):
			customerAccounts++
		}

		if hasValue(supplierDetectionJournals, entry.Journal) {
			supplierJournals++
		}
		if hasValue(saleJournals, entry.Journal) {
			customerJournals++
		}
	}

	switch {
	case supplierAccounts > customerAccounts:
		return LedgerSupplier, nil
	case customerAccounts > supplierAccounts:
		return LedgerCustomer, nil
	case supplierJournals > customerJournals:
		return LedgerSupplier, nil
	case customerJournals > supplierJournals:
		return LedgerCustomer, nil
	default:
		return 0, fmt.Errorf(
			"impossible de reconnaître automatiquement le grand livre. Choisissez Fournisseur ou Client dans l'application",
		)
	}
}

func allEmpty(fields []string) bool {
	for _, field := range fields {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}
	return true
}

func getField(fields []string, headerMap map[string]int, header string) string {
	index, found := headerMap[header]
	if !found || index >= len(fields) {
		return ""
	}
	return fields[index]
}

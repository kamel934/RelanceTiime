package core

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/shopspring/decimal"
	"golang.org/x/text/unicode/norm"
)

var grandLivrePattern = regexp.MustCompile(
	`(?i)^grand_livre_(\d{4})-\d{2}-\d{2}_(\d{4})-\d{2}-\d{2}_(.+?)(?: \(\d+\))?\.csv$`,
)

func NormalizeHeader(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var builder strings.Builder
	previousUnderscore := false
	for _, character := range norm.NFD.String(value) {
		if unicode.Is(unicode.Mn, character) {
			continue
		}
		character = unicode.ToLower(character)
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			builder.WriteRune(character)
			previousUnderscore = false
		} else if !previousUnderscore {
			builder.WriteByte('_')
			previousUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func ParseAmount(value string) (decimal.Decimal, error) {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return decimal.Zero, nil
	}

	replacer := strings.NewReplacer(
		"\u00a0", "",
		"\u202f", "",
		" ", "",
		"€", "",
	)
	cleaned = replacer.Replace(cleaned)
	if strings.Contains(cleaned, ",") {
		cleaned = strings.ReplaceAll(cleaned, ".", "")
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	}

	amount, err := decimal.NewFromString(cleaned)
	if err != nil {
		return decimal.Zero, fmt.Errorf("montant invalide : %s", value)
	}
	return amount, nil
}

func ParseDate(value string) (*time.Time, error) {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return nil, nil
	}

	for _, layout := range []string{"02/01/2006", "2006-01-02", "02-01-2006"} {
		if parsed, err := time.ParseInLocation(layout, cleaned, time.Local); err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("date invalide : %s", value)
}

func NullIfZero(amount decimal.Decimal) *decimal.Decimal {
	if amount.IsZero() {
		return nil
	}
	value := amount
	return &value
}

func FormatAmount(amount decimal.Decimal) string {
	parts := strings.Split(amount.StringFixed(2), ".")
	integer := parts[0]
	sign := ""
	if strings.HasPrefix(integer, "-") {
		sign = "-"
		integer = strings.TrimPrefix(integer, "-")
	}

	for index := len(integer) - 3; index > 0; index -= 3 {
		integer = integer[:index] + " " + integer[index:]
	}
	return sign + integer + "," + parts[1]
}

func TryParseMetadata(path string) *FileMetadata {
	match := grandLivrePattern.FindStringSubmatch(filepath.Base(path))
	if match == nil {
		return nil
	}

	year := match[1]
	if match[1] != match[2] {
		year = match[1] + "-" + match[2]
	}
	client := strings.ToUpper(strings.TrimSpace(
		strings.NewReplacer("_", " ", "-", " ").Replace(match[3]),
	))
	for strings.Contains(client, "  ") {
		client = strings.ReplaceAll(client, "  ", " ")
	}
	return &FileMetadata{Client: client, Year: year}
}

func SanitizeWorksheetName(name string) string {
	replacer := strings.NewReplacer(
		"[", " ", "]", " ", "*", " ", "?", " ", "/", " ", "\\", " ", ":", " ",
	)
	cleaned := strings.TrimSpace(replacer.Replace(name))
	runes := []rune(cleaned)
	if len(runes) > 31 {
		return string(runes[:31])
	}
	return cleaned
}

func sanitizeFileName(name string) string {
	var builder strings.Builder
	for _, character := range name {
		switch {
		case character < 32:
			builder.WriteByte(' ')
		case strings.ContainsRune(`<>:"/\|?*`, character):
			builder.WriteByte(' ')
		default:
			builder.WriteRune(character)
		}
	}
	cleaned := strings.TrimSpace(builder.String())
	for strings.Contains(cleaned, "  ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}
	return cleaned
}

func calculateRowHeight(movement string) float64 {
	length := len([]rune(strings.TrimSpace(movement)))
	if length == 0 {
		length = 1
	}
	lineCount := (length + MovementCharsPerLine - 1) / MovementCharsPerLine
	if lineCount <= 1 {
		return BaseRowHeight
	}
	height := BaseRowHeight + float64(lineCount-1)*WrappedLineHeight + RowHeightPadding
	if height > MaxRowHeight {
		return MaxRowHeight
	}
	return height
}

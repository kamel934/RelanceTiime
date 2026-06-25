package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type UserSettings struct {
	Excel      bool   `json:"excel"`
	PDF        bool   `json:"pdf"`
	LedgerMode string `json:"ledger_mode"`
}

func (s UserSettings) Formats() OutputFormats {
	formats := OutputNone
	if s.Excel {
		formats |= OutputExcel
	}
	if s.PDF {
		formats |= OutputPDF
	}
	return formats
}

func (s UserSettings) ParsedLedgerMode() LedgerMode {
	switch strings.ToLower(s.LedgerMode) {
	case "supplier":
		return LedgerModeSupplier
	case "customer":
		return LedgerModeCustomer
	default:
		return LedgerAuto
	}
}

func (s *UserSettings) SetFormats(formats OutputFormats) {
	s.Excel = formats.Has(OutputExcel)
	s.PDF = formats.Has(OutputPDF)
}

func (s *UserSettings) SetLedgerMode(mode LedgerMode) {
	switch mode {
	case LedgerModeSupplier:
		s.LedgerMode = "supplier"
	case LedgerModeCustomer:
		s.LedgerMode = "customer"
	default:
		s.LedgerMode = "auto"
	}
}

type SettingsStore struct {
	path string
}

func NewSettingsStore(baseDirectory string) *SettingsStore {
	directory := baseDirectory
	if directory == "" {
		directory = filepath.Join(os.Getenv("APPDATA"), SettingsDirectory)
	}
	return &SettingsStore{path: filepath.Join(directory, SettingsFile)}
}

func (s *SettingsStore) Load(defaultFormats OutputFormats, defaultMode LedgerMode, fixInvalid bool) UserSettings {
	settings := UserSettings{}
	data, err := os.ReadFile(s.path)
	if err != nil || json.Unmarshal(data, &settings) != nil {
		settings.SetFormats(defaultFormats)
		settings.SetLedgerMode(defaultMode)
	}
	if settings.LedgerMode == "" {
		settings.SetLedgerMode(defaultMode)
	}
	if fixInvalid && settings.Formats() == OutputNone {
		settings.SetFormats(OutputExcel)
	}
	return settings
}

func (s *SettingsStore) Save(settings UserSettings) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

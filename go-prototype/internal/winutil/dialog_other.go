//go:build !windows

package winutil

import "fmt"

func SelectCSVFiles() ([]string, error) {
	return nil, fmt.Errorf("sélecteur de fichiers disponible uniquement sous Windows")
}

func ShowError(title, message string) {}

func IsCSV(path string) bool {
	return false
}

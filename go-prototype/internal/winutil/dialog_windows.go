//go:build windows

package winutil

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ofnExplorer         = 0x00080000
	ofnFileMustExist    = 0x00001000
	ofnPathMustExist    = 0x00000800
	ofnAllowMultiSelect = 0x00000200
	ofnNoChangeDir      = 0x00000008
)

type openFileName struct {
	structSize       uint32
	owner            windows.Handle
	instance         windows.Handle
	filter           *uint16
	customFilter     *uint16
	maxCustomFilter  uint32
	filterIndex      uint32
	file             *uint16
	maxFile          uint32
	fileTitle        *uint16
	maxFileTitle     uint32
	initialDirectory *uint16
	title            *uint16
	flags            uint32
	fileOffset       uint16
	fileExtension    uint16
	defaultExtension *uint16
	customData       uintptr
	hook             uintptr
	templateName     *uint16
	reserved         uintptr
	reservedDword    uint32
	flagsEx          uint32
}

var (
	comdlg32           = windows.NewLazySystemDLL("comdlg32.dll")
	getOpenFileName    = comdlg32.NewProc("GetOpenFileNameW")
	user32             = windows.NewLazySystemDLL("user32.dll")
	messageBox         = user32.NewProc("MessageBoxW")
	getOpenFileNameErr = comdlg32.NewProc("CommDlgExtendedError")
)

func SelectCSVFiles() ([]string, error) {
	buffer := make([]uint16, 65536)
	filter := syscall.StringToUTF16("Exports CSV (*.csv)\x00*.csv\x00Tous les fichiers (*.*)\x00*.*\x00")
	title, _ := windows.UTF16PtrFromString("Choisir un grand livre CSV")
	defaultExtension, _ := windows.UTF16PtrFromString("csv")

	dialog := openFileName{
		filter:           &filter[0],
		filterIndex:      1,
		file:             &buffer[0],
		maxFile:          uint32(len(buffer)),
		title:            title,
		defaultExtension: defaultExtension,
		flags:            ofnExplorer | ofnFileMustExist | ofnPathMustExist | ofnAllowMultiSelect | ofnNoChangeDir,
	}
	dialog.structSize = uint32(unsafe.Sizeof(dialog))

	result, _, callErr := getOpenFileName.Call(uintptr(unsafe.Pointer(&dialog)))
	if result == 0 {
		extended, _, _ := getOpenFileNameErr.Call()
		if extended == 0 {
			return nil, nil
		}
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return nil, callErr
		}
		return nil, syscall.Errno(extended)
	}

	parts := splitUTF16Buffer(buffer)
	if len(parts) == 0 {
		return nil, nil
	}
	if len(parts) == 1 {
		return parts, nil
	}

	directory := parts[0]
	paths := make([]string, 0, len(parts)-1)
	for _, name := range parts[1:] {
		paths = append(paths, filepath.Join(directory, name))
	}
	return paths, nil
}

func ShowError(title, message string) {
	titlePointer, _ := windows.UTF16PtrFromString(title)
	messagePointer, _ := windows.UTF16PtrFromString(message)
	const mbOK = 0x00000000
	const mbIconError = 0x00000010
	messageBox.Call(
		0,
		uintptr(unsafe.Pointer(messagePointer)),
		uintptr(unsafe.Pointer(titlePointer)),
		mbOK|mbIconError,
	)
}

func splitUTF16Buffer(buffer []uint16) []string {
	var result []string
	start := 0
	for index, character := range buffer {
		if character != 0 {
			continue
		}
		if index == start {
			break
		}
		result = append(result, windows.UTF16ToString(buffer[start:index]))
		start = index + 1
	}
	return result
}

func IsCSV(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".csv")
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kamel934/RelanceTiime/go-prototype/internal/core"
	"github.com/kamel934/RelanceTiime/go-prototype/internal/ui"
	"github.com/kamel934/RelanceTiime/go-prototype/internal/winutil"
)

func main() {
	runtime.LockOSThread()

	noDialog := false
	var files []string
	for _, argument := range os.Args[1:] {
		if strings.EqualFold(argument, "--no-dialog") {
			noDialog = true
			continue
		}
		files = append(files, argument)
	}

	if len(files) > 0 {
		os.Exit(runDirect(files, noDialog))
	}
	ui.NewApplication().Run()
}

func runDirect(files []string, noDialog bool) int {
	settings := core.NewSettingsStore("").Load(
		core.OutputExcel|core.OutputPDF,
		core.LedgerAuto,
		true,
	)
	processor := core.NewProcessor(core.NewExcelExporter(), core.NewPDFExporter())
	result, err := processor.ProcessFiles(files, core.ProcessingOptions{
		Formats:    settings.Formats(),
		LedgerMode: settings.ParsedLedgerMode(),
		Role:       core.RoleFirst,
	})
	if err == nil {
		if noDialog {
			fmt.Println(core.BuildSummary(result))
		}
		return 0
	}

	logPath := filepath.Join(os.TempDir(), "relance_tiime_go_error.log")
	_ = os.WriteFile(logPath, []byte(err.Error()), 0o644)
	if noDialog {
		fmt.Fprintln(os.Stderr, err)
	} else {
		winutil.ShowError("Liste Tiime Go", err.Error())
	}
	return 1
}
